package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/catalyst"
	"github.com/richman/backend/internal/analysis/confidence"
	"github.com/richman/backend/internal/analysis/position"
	"github.com/richman/backend/internal/analysis/synthesis"
	"github.com/richman/backend/internal/analysis/trend"
	"github.com/richman/backend/internal/analysis/weight"
	"github.com/richman/backend/internal/datasource"
	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

// Service orchestrates the full analysis pipeline.
type Service struct {
	holdingRepo     *repo.HoldingRepo
	cardRepo        *repo.DecisionCardRepo
	resultRepo      *repo.AnalysisResultRepo
	fetcher         *datasource.Fetcher
	trendCalc       *trend.Calculator
	posCalc         *position.Calculator
	catCalc         *catalyst.Calculator
	llmEnhancer     *catalyst.LLMEnhancer
	synthesizer     *synthesis.Synthesizer
	weightMgr       *weight.Manager
	confCalc        *confidence.Calculator
	matrix          *analysis.Matrix
	taskStore       *TaskStore
	logger          *zap.Logger
	analysisTimeout time.Duration
	semaphore       chan struct{}
}

// Deps holds all dependencies for the analysis Service.
type Deps struct {
	HoldingRepo     *repo.HoldingRepo
	CardRepo        *repo.DecisionCardRepo
	ResultRepo      *repo.AnalysisResultRepo
	Fetcher         *datasource.Fetcher
	TrendCalc       *trend.Calculator
	PosCalc         *position.Calculator
	CatCalc         *catalyst.Calculator
	LLMEnhancer     *catalyst.LLMEnhancer
	Synthesizer     *synthesis.Synthesizer
	WeightMgr       *weight.Manager
	ConfCalc        *confidence.Calculator
	Matrix          *analysis.Matrix
	TaskStore       *TaskStore
	Logger          *zap.Logger
	AnalysisTimeout time.Duration
	MaxConcurrent   int
}

// NewService creates a new analysis Service.
func NewService(deps *Deps) *Service {
	var sem chan struct{}
	if deps.MaxConcurrent > 0 {
		sem = make(chan struct{}, deps.MaxConcurrent)
	}
	timeout := deps.AnalysisTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Service{
		holdingRepo:     deps.HoldingRepo,
		cardRepo:        deps.CardRepo,
		resultRepo:      deps.ResultRepo,
		fetcher:         deps.Fetcher,
		trendCalc:       deps.TrendCalc,
		posCalc:         deps.PosCalc,
		catCalc:         deps.CatCalc,
		llmEnhancer:     deps.LLMEnhancer,
		synthesizer:     deps.Synthesizer,
		weightMgr:       deps.WeightMgr,
		confCalc:        deps.ConfCalc,
		matrix:          deps.Matrix,
		taskStore:       deps.TaskStore,
		logger:          deps.Logger,
		analysisTimeout: timeout,
		semaphore:       sem,
	}
}

// GetTaskStore returns the task store for external status queries.
func (s *Service) GetTaskStore() *TaskStore {
	return s.taskStore
}

// TriggerAnalysis starts an async analysis for all holdings of a user.
// It returns a task ID immediately and runs the analysis in the background.
func (s *Service) TriggerAnalysis(ctx context.Context, userID int64, taskID string) {
	s.taskStore.Create(taskID, userID)

	// Use a detached context so the background work is not canceled when the
	// HTTP request completes.
	bgCtx := context.WithoutCancel(ctx)

	go func() {
		s.taskStore.UpdateProgress(taskID, 0.05)

		holdings, err := s.holdingRepo.ListHoldingsByUser(bgCtx, userID)
		if err != nil {
			s.logger.Error("failed to list holdings for analysis",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
			s.taskStore.Fail(taskID, err)
			return
		}

		if len(holdings) == 0 {
			s.taskStore.Fail(taskID, fmt.Errorf("no holdings found for user"))
			return
		}

		total := float64(len(holdings))
		for i := range holdings {
			progress := 0.1 + (float64(i)/total)*0.85
			s.taskStore.UpdateProgress(taskID, progress)

			ctxHolding, cancel := s.holdingContext(bgCtx)
			_, analyzeErr := s.AnalyzeHolding(ctxHolding, userID, &holdings[i])
			cancel()
			if analyzeErr != nil {
				s.logger.Error("failed to analyze holding",
					zap.Int64("holding_id", holdings[i].HoldingID),
					zap.String("asset", holdings[i].AssetCode),
					zap.Error(analyzeErr),
				)
				// Continue with other holdings even if one fails.
			}
		}

		s.taskStore.Complete(taskID)
	}()
}

// AnalyzeHolding runs the full analysis pipeline for a single holding.
func (s *Service) AnalyzeHolding(
	ctx context.Context, userID int64, holding *model.Holding,
) (*model.DecisionCard, error) {
	release := s.acquireSlot()
	defer release()
	if _, ok := ctx.Deadline(); !ok && s.analysisTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.analysisTimeout)
		defer cancel()
	}

	s.logger.Info("starting analysis",
		zap.Int64("holding_id", holding.HoldingID),
		zap.String("asset", holding.AssetCode),
	)

	// Step 1: Fetch data.
	data, err := s.fetcher.FetchAssetData(ctx, holding.AssetCode, holding.AssetType)
	if err != nil {
		return nil, fmt.Errorf("fetch data: %w", err)
	}

	// Step 2: Calculate trend.
	trendResult, err := s.trendCalc.Calculate(data.Prices)
	if err != nil {
		s.logger.Warn("trend calculation failed, using neutral default",
			zap.String("asset", holding.AssetCode),
			zap.Error(err),
		)
		trendResult = analysis.TrendResult{
			Direction: analysis.DirectionSideways,
			Strength:  0,
			Summary:   "Insufficient data for trend analysis.",
		}
	}

	// Step 3: Calculate position (valuation).
	// The position calculator needs historical valuation data. For the MVP
	// we only have current valuation from the fetcher, so we build a minimal
	// history from prices. A full history source will be added later.
	posResult := analysis.PositionResult{
		Assessment: analysis.DirectionNeutral,
		Percentile: 0.5,
		Summary:    "Valuation data not available.",
	}
	if data.Valuation != nil {
		// Build a simple price-based percentile for gold_etf; for A-share use
		// the position calculator if we have enough data.
		posCalcResult, posErr := s.posCalc.Calculate(
			holding.AssetType,
			data.Valuation,
			nil, // no historical valuation series in MVP
			data.Prices,
		)
		if posErr != nil {
			s.logger.Warn("position calculation failed, using neutral default",
				zap.String("asset", holding.AssetCode),
				zap.Error(posErr),
			)
		} else {
			posResult = posCalcResult
		}
	} else if holding.AssetType == "gold_etf" {
		// Gold uses price percentile, no valuation data needed.
		posCalcResult, posErr := s.posCalc.Calculate(
			holding.AssetType,
			nil,
			nil,
			data.Prices,
		)
		if posErr == nil {
			posResult = posCalcResult
		}
	}

	// Step 4: Calculate catalyst (base).
	catResult := s.catCalc.Calculate(data.Events, nil)

	// Step 5: LLM enhance catalyst.
	hasLLM := false
	if s.llmEnhancer != nil {
		enhanced, enhErr := s.llmEnhancer.Enhance(ctx, catResult, holding.AssetCode, holding.AssetType)
		if enhErr == nil && enhanced != nil {
			catResult = *enhanced
			hasLLM = true
		}
	}

	// Step 6: Get weights.
	weights, err := s.weightMgr.GetBaseWeights(holding.AssetType)
	if err != nil {
		s.logger.Warn("failed to get weights, using equal weights",
			zap.String("type", holding.AssetType),
			zap.Error(err),
		)
		weights = analysis.WeightConfig{Trend: 0.33, Position: 0.34, Catalyst: 0.33}
	}

	// Step 7: Calculate confidence.
	conf := s.confCalc.Calculate(confidence.Input{
		Trend:          &trendResult,
		Position:       &posResult,
		Catalyst:       &catResult,
		HasLLMCatalyst: hasLLM,
	})

	// Step 8: Decide recommendation.
	recommendation := s.matrix.Decide(trendResult, posResult, catResult, weights)

	// Step 9: Synthesize card content.
	costPrice, _ := holding.CostPrice.Float64()
	posRatio, _ := holding.PositionRatio.Float64()

	synthOutput, err := s.synthesizer.Synthesize(ctx, &synthesis.SynthesisInput{
		AssetCode:      holding.AssetCode,
		AssetType:      holding.AssetType,
		AssetName:      holding.AssetName,
		Trend:          trendResult,
		Position:       posResult,
		Catalyst:       catResult,
		Weights:        weights,
		Confidence:     conf,
		Recommendation: recommendation,
		CostPrice:      costPrice,
		PositionRatio:  posRatio,
	})
	if err != nil {
		return nil, fmt.Errorf("synthesize: %w", err)
	}

	now := time.Now()

	// Build decision card.
	card := &model.DecisionCard{
		UserID:            userID,
		HoldingID:         holding.HoldingID,
		AssetCode:         holding.AssetCode,
		AssetName:         holding.AssetName,
		AssetType:         holding.AssetType,
		CostPrice:         costPrice,
		PositionRatio:     posRatio,
		TrendDirection:    string(trendResult.Direction),
		TrendSummary:      synthOutput.TrendSummary,
		PositionDirection: string(posResult.Assessment),
		PositionSummary:   synthOutput.PositionSummary,
		CatalystDirection: string(catResult.Direction),
		CatalystSummary:   synthOutput.CatalystSummary,
		Confidence:        conf,
		Recommendation:    string(recommendation),
		ActionAdvice:      synthOutput.ActionAdvice,
		DetailedAdvice:    synthOutput.DetailedAdvice,
		RiskWarnings:      synthOutput.RiskWarnings,
		TodayHighlights:   synthOutput.TodayHighlights,
		WeightTrend:       weights.Trend,
		WeightPosition:    weights.Position,
		WeightCatalyst:    weights.Catalyst,
		AnalyzedAt:        now,
	}

	// Step 10: Persist raw analysis result.
	rawResult := analysis.AnalysisResult{
		AssetCode:      holding.AssetCode,
		AssetType:      holding.AssetType,
		Trend:          trendResult,
		Position:       posResult,
		Catalyst:       catResult,
		Weights:        weights,
		Confidence:     conf,
		Recommendation: recommendation,
		AnalyzedAt:     now,
	}
	rawJSON, _ := json.Marshal(rawResult)
	_, saveErr := s.resultRepo.CreateAnalysisResult(ctx, userID, holding.HoldingID, holding.AssetCode, string(rawJSON))
	if saveErr != nil {
		s.logger.Warn("failed to save analysis result", zap.Error(saveErr))
	}

	// Step 11: Persist decision card.
	saved, err := s.cardRepo.CreateDecisionCard(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("save decision card: %w", err)
	}

	s.logger.Info("analysis completed",
		zap.Int64("holding_id", holding.HoldingID),
		zap.String("recommendation", string(recommendation)),
		zap.Float64("confidence", conf),
	)

	return saved, nil
}

func (s *Service) holdingContext(parent context.Context) (context.Context, context.CancelFunc) {
	if s.analysisTimeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, s.analysisTimeout)
}

func (s *Service) acquireSlot() func() {
	if s.semaphore == nil {
		return func() {}
	}
	s.semaphore <- struct{}{}
	return func() {
		<-s.semaphore
	}
}
