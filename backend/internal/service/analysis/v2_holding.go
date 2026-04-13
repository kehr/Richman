package analysis

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/richson"
	"go.uber.org/zap"
)

// ConcentrationLevel describes the risk tier for a given asset-type exposure.
type ConcentrationLevel string

const (
	// ConcentrationRed indicates exposure > 30%.
	ConcentrationRed ConcentrationLevel = "red"
	// ConcentrationOrange indicates exposure > 20%.
	ConcentrationOrange ConcentrationLevel = "orange"
	// ConcentrationBlue indicates exposure > 10%.
	ConcentrationBlue ConcentrationLevel = "blue"
	// ConcentrationGreen indicates exposure within the acceptable range (<=10%).
	ConcentrationGreen ConcentrationLevel = "green"
)

// ComputeConcentration returns the concentration level and a message for a
// given total exposure value (sum of position_ratio for an asset type).
// Thresholds: >30% red, >20% orange, >10% blue, else green.
func ComputeConcentration(totalExposure float64) (ConcentrationLevel, string) {
	switch {
	case totalExposure > 30:
		return ConcentrationRed, fmt.Sprintf("exposure %.1f%% exceeds 30%% threshold, high concentration risk", totalExposure)
	case totalExposure > 20:
		return ConcentrationOrange, fmt.Sprintf("exposure %.1f%% exceeds 20%% threshold, moderate concentration risk", totalExposure)
	case totalExposure > 10:
		return ConcentrationBlue, fmt.Sprintf("exposure %.1f%% exceeds 10%% threshold, low concentration risk", totalExposure)
	default:
		return ConcentrationGreen, fmt.Sprintf("exposure %.1f%% within acceptable range", totalExposure)
	}
}

// V2HoldingAnalyzer orchestrates the per-holding analysis flow that calls the
// richson sidecar and persists the resulting decision card. An in-flight
// deduplication map prevents concurrent analyses for the same user+holding.
type V2HoldingAnalyzer struct {
	holdingRepo   *repo.HoldingRepo
	analysisRepo  *repo.AssetAnalysisReadRepo
	userRepo      *repo.UserRepo
	llmConfigRepo *repo.LLMConfigRepo
	cardRepo      *repo.DecisionCardRepo
	richsonClient *richson.Client
	crypto        *llm.Crypto
	logger        *zap.Logger

	// inFlight prevents concurrent analyses for the same user+holding pair.
	// Key format: "<userID>:<holdingID>".
	inFlight sync.Map
}

// V2HoldingAnalyzerDeps holds all dependencies for V2HoldingAnalyzer.
type V2HoldingAnalyzerDeps struct {
	HoldingRepo   *repo.HoldingRepo
	AnalysisRepo  *repo.AssetAnalysisReadRepo
	UserRepo      *repo.UserRepo
	LLMConfigRepo *repo.LLMConfigRepo
	CardRepo      *repo.DecisionCardRepo
	RichsonClient *richson.Client
	Crypto        *llm.Crypto
	Logger        *zap.Logger
}

// NewV2HoldingAnalyzer constructs a V2HoldingAnalyzer from the given deps.
func NewV2HoldingAnalyzer(deps *V2HoldingAnalyzerDeps) *V2HoldingAnalyzer {
	return &V2HoldingAnalyzer{
		holdingRepo:   deps.HoldingRepo,
		analysisRepo:  deps.AnalysisRepo,
		userRepo:      deps.UserRepo,
		llmConfigRepo: deps.LLMConfigRepo,
		cardRepo:      deps.CardRepo,
		richsonClient: deps.RichsonClient,
		crypto:        deps.Crypto,
		logger:        deps.Logger,
	}
}

// AnalyzeHolding executes the 7-step holding analysis flow:
//  1. Idempotency guard via sync.Map (returns 409 if already in-flight)
//  2-5. Parallel errgroup: holding, latest analysis, peer exposure, user+LLM config
//  6. Call richson POST /analyze/holding
//  7. Persist execution plan to rm_decision_cards
func (a *V2HoldingAnalyzer) AnalyzeHolding(
	ctx context.Context, userID, holdingID int64,
) (*model.DecisionCard, error) {
	// Step 1: idempotency lock.
	key := fmt.Sprintf("%d:%d", userID, holdingID)
	if _, loaded := a.inFlight.LoadOrStore(key, struct{}{}); loaded {
		return nil, model.NewAppError(http.StatusConflict, "ANALYSIS_IN_PROGRESS",
			"analysis already in progress for this holding")
	}
	defer a.inFlight.Delete(key)

	// Step 2-5: parallel data gathering.
	var (
		holding        *model.Holding
		latestAnalysis *model.AssetAnalysis
		peerExposure   float64
		riskPreference string
		userLanguage   string
		llmCfg         *model.LLMConfig
	)

	eg, egCtx := errgroup.WithContext(ctx)

	// Step 2: load holding.
	eg.Go(func() error {
		var err error
		holding, err = a.holdingRepo.GetHoldingByID(egCtx, holdingID)
		if err != nil {
			return fmt.Errorf("load holding: %w", err)
		}
		if holding == nil {
			return model.NewAppError(http.StatusNotFound, "HOLDING_NOT_FOUND",
				fmt.Sprintf("holding %d not found", holdingID))
		}
		if holding.UserID != userID {
			return model.NewAppError(http.StatusForbidden, "HOLDING_ACCESS_DENIED",
				"holding does not belong to this user")
		}
		return nil
	})

	// Steps 3-5 require holding.AssetCode and holding.AssetType which are not
	// yet available. We wait for step 2, then launch the remaining goroutines.
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	eg2, egCtx2 := errgroup.WithContext(ctx)

	// Step 3: latest analysis for the holding's asset.
	eg2.Go(func() error {
		var err error
		latestAnalysis, err = a.analysisRepo.GetLatestByAssetCode(egCtx2, holding.AssetCode)
		if err != nil {
			return fmt.Errorf("load latest analysis: %w", err)
		}
		return nil
	})

	// Step 4a: peer exposure (sum of position_ratio for same asset_type).
	eg2.Go(func() error {
		var err error
		peerExposure, err = a.holdingRepo.GetExposureByAssetType(egCtx2, userID, holding.AssetType)
		if err != nil {
			return fmt.Errorf("load peer exposure: %w", err)
		}
		return nil
	})

	// Step 4b: user risk preference and language.
	eg2.Go(func() error {
		var err error
		riskPreference, err = a.userRepo.GetRiskPreference(egCtx2, userID)
		if err != nil {
			return fmt.Errorf("load risk preference: %w", err)
		}
		userLanguage, err = a.userRepo.GetLanguage(egCtx2, userID)
		if err != nil {
			return fmt.Errorf("load user language: %w", err)
		}
		return nil
	})

	// Step 5: optional user LLM config (absence is not an error).
	eg2.Go(func() error {
		var err error
		llmCfg, err = a.llmConfigRepo.GetActiveByUserID(egCtx2, userID)
		if err != nil && !isLLMConfigNotFound(err) {
			return fmt.Errorf("load llm config: %w", err)
		}
		// nil llmCfg means the user has no custom config; richson uses system default.
		return nil
	})

	if err := eg2.Wait(); err != nil {
		return nil, err
	}

	if latestAnalysis == nil {
		return nil, model.NewAppError(http.StatusUnprocessableEntity, "NO_ANALYSIS",
			fmt.Sprintf("no analysis found for asset %s", holding.AssetCode))
	}

	// Build richson LLMConfig from user's encrypted config (if present).
	var richsonLLMCfg *richson.LLMConfig
	if llmCfg != nil && a.crypto != nil {
		richsonLLMCfg = a.buildRichsonLLMConfig(llmCfg)
	}

	costStr := holding.CostPrice.String()
	posRatio, _ := holding.PositionRatio.Float64()
	qty, _ := holding.Quantity.Float64()

	// Step 6: call richson.
	req := richson.AnalyzeHoldingRequest{
		AssetCode:       holding.AssetCode,
		AssetAnalysisID: strconv.FormatInt(latestAnalysis.AssetAnalysisID, 10),
		Holding: richson.HoldingInput{
			HoldingID:     strconv.FormatInt(holding.HoldingID, 10),
			CostPrice:     costStr,
			PositionRatio: posRatio,
			Quantity:      qty,
		},
		RiskPreference: riskPreference,
		PeerExposure:   peerExposure,
		Language:       userLanguage,
		LLMConfig:      richsonLLMCfg,
	}

	a.logger.Info("calling richson analyze holding",
		zap.Int64("user_id", userID),
		zap.Int64("holding_id", holdingID),
		zap.String("asset_code", holding.AssetCode),
	)

	resp, err := a.richsonClient.AnalyzeHolding(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("richson analyze holding: %w", err)
	}

	// Step 7: persist the execution plan as a decision card.
	card, err := a.persistDecisionCard(ctx, userID, holding, latestAnalysis, resp)
	if err != nil {
		return nil, err
	}

	a.logger.Info("holding analysis complete",
		zap.Int64("user_id", userID),
		zap.Int64("holding_id", holdingID),
		zap.Int64("card_id", card.CardID),
	)

	return card, nil
}

// persistDecisionCard converts a richson response into a model.DecisionCard
// and persists it via the card repo.
func (a *V2HoldingAnalyzer) persistDecisionCard(
	ctx context.Context,
	userID int64,
	holding *model.Holding,
	analysis *model.AssetAnalysis,
	resp *richson.HoldingAnalysisResponse,
) (*model.DecisionCard, error) {
	plan := resp.ExecutionPlan

	// Determine concentration level and message.
	posRatio, _ := holding.PositionRatio.Float64()
	_, concMsg := ComputeConcentration(posRatio)

	// Build the target position ratio: use plan.TargetPosition as percentage.
	targetPosRatio := plan.TargetPosition

	// Build risk warnings from the plan action label and no-trigger note.
	riskWarnings := []string{}
	if plan.NoTriggerNote != nil && *plan.NoTriggerNote != "" {
		riskWarnings = append(riskWarnings, *plan.NoTriggerNote)
	}

	currentPrice := 0.0
	if analysis.PriceAtAnalysis != nil {
		currentPrice = *analysis.PriceAtAnalysis
	}
	qty, _ := holding.Quantity.Float64()
	posRatioDecimal := posRatio

	synthSource := "richson"
	card := &model.DecisionCard{
		UserID:              userID,
		HoldingID:           holding.HoldingID,
		AssetCode:           holding.AssetCode,
		AssetName:           holding.AssetName,
		AssetType:           holding.AssetType,
		CostPrice:           holding.CostPrice.InexactFloat64(),
		CurrentPrice:        currentPrice,
		Quantity:            qty,
		PositionRatio:       posRatioDecimal,
		TrendDirection:      plan.DefaultAction,
		TrendSummary:        concMsg,
		PositionDirection:   plan.Action,
		PositionSummary:     plan.ActionLabel,
		CatalystDirection:   plan.ConcentrationLevel,
		CatalystSummary:     plan.ConcentrationMessage,
		Confidence:          0,
		ActionAdvice:        plan.ActionLabel,
		DetailedAdvice:      plan.ActionLabel,
		RiskWarnings:        riskWarnings,
		TodayHighlights:     "",
		WeightTrend:         0,
		WeightPosition:      0,
		WeightCatalyst:      0,
		AnalyzedAt:          time.Now().UTC(),
		ActionLevel:         0,
		TargetPositionRatio: targetPosRatio,
		BadgeState:          "",
		SynthesisSource:     &synthSource,
	}

	inserted, err := a.cardRepo.CreateDecisionCard(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("persist decision card: %w", err)
	}
	return inserted, nil
}

// buildRichsonLLMConfig decrypts the user's LLM config and converts it to the
// richson wire format. Returns nil when decryption fails (richson uses system
// default in that case).
func (a *V2HoldingAnalyzer) buildRichsonLLMConfig(cfg *model.LLMConfig) *richson.LLMConfig {
	plaintext, err := a.crypto.Decrypt(cfg.APIKeyCipher, cfg.APIKeyNonce)
	if err != nil {
		a.logger.Warn("failed to decrypt user llm api key, falling back to system default",
			zap.Int64("user_id", cfg.UserID),
			zap.Error(err),
		)
		return nil
	}
	defer func() {
		for i := range plaintext {
			plaintext[i] = 0
		}
	}()

	rc := &richson.LLMConfig{
		Provider: string(cfg.ProviderType),
		Model:    cfg.Model,
		APIKey:   string(plaintext),
	}
	if cfg.BaseURL != nil {
		rc.APIBase = cfg.BaseURL
	}
	return rc
}

// isLLMConfigNotFound checks if the error is an llm config not found sentinel.
// This avoids importing the llm package solely for the error value.
func isLLMConfigNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "llm: config not found"
}
