package briefing

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

// concentrationLevel represents the risk color for an asset type exposure.
type concentrationLevel string

const (
	concentrationRed    concentrationLevel = "red"
	concentrationOrange concentrationLevel = "orange"
	concentrationBlue   concentrationLevel = "blue"
	concentrationGreen  concentrationLevel = "green"
)

// BriefingCardDTO is the per-holding card displayed in the daily briefing.
type BriefingCardDTO struct {
	HoldingID            int64              `json:"holdingId"`
	AssetCode            string             `json:"assetCode"`
	AssetName            string             `json:"assetName"`
	AssetType            string             `json:"assetType"`
	CostPrice            string             `json:"costPrice"`
	CurrentPrice         *float64           `json:"currentPrice,omitempty"`
	PositionRatio        string             `json:"positionRatio"`
	Quantity             string             `json:"quantity"`
	PnLPercent           *float64           `json:"pnlPercent,omitempty"`
	OverallScore         *float64           `json:"overallScore,omitempty"`
	SignalLevel          *string            `json:"signalLevel,omitempty"`
	ScoreDelta           *float64           `json:"scoreDelta,omitempty"`
	SparklineScores      []float64          `json:"sparklineScores"`
	LatestCardID         *int64             `json:"latestCardId,omitempty"`
	ActionLevel          *int               `json:"actionLevel,omitempty"`
	ConcentrationLevel   concentrationLevel `json:"concentrationLevel"`
	ConcentrationMessage string             `json:"concentrationMessage"`
	AnalyzedAt           *time.Time         `json:"analyzedAt,omitempty"`

	// AssetAnalysisID is the primary key of the rs_asset_analyses row backing
	// OverallScore/SignalLevel/ScoreDelta above. The frontend includes this
	// value in POST /api/v2/feedback so the feedback row can reference the
	// exact analysis that was rated. NULL when no analysis exists yet.
	AssetAnalysisID *int64 `json:"assetAnalysisId,omitempty"`
	// ChangeAttribution is a one-sentence attribution for today's score
	// change (shown in the UI when |scoreDelta| >= 5). Sourced from the
	// analysis change_summary column.
	ChangeAttribution *string `json:"changeAttribution,omitempty"`
	// ConflictWarning is a human-readable warning when dimensional
	// conflicts are detected (analysis.conflict_message).
	ConflictWarning *string `json:"conflictWarning,omitempty"`
	// Direction is the derived trend label based on SignalLevel/OverallScore:
	// "bullish" | "bearish" | "neutral".
	Direction string `json:"direction"`
}

// BriefingDTO is the full response for GET /v2/briefing.
type BriefingDTO struct {
	Cards     []BriefingCardDTO `json:"cards"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// sparklineWindow is the number of recent scores used for the sparkline chart.
const sparklineWindow = 90

// Service provides the daily briefing aggregation logic.
type Service struct {
	holdingRepo  *repo.HoldingRepo
	analysisRepo *repo.AssetAnalysisReadRepo
	cardRepo     *repo.DecisionCardRepo
	logger       *zap.Logger
}

// NewService constructs a briefing Service.
func NewService(
	holdingRepo *repo.HoldingRepo,
	analysisRepo *repo.AssetAnalysisReadRepo,
	cardRepo *repo.DecisionCardRepo,
	logger *zap.Logger,
) *Service {
	return &Service{
		holdingRepo:  holdingRepo,
		analysisRepo: analysisRepo,
		cardRepo:     cardRepo,
		logger:       logger,
	}
}

// GetBriefing assembles the daily briefing for a user. Steps 1-4 run in
// parallel via errgroup; steps 5-7 run sequentially using the gathered data.
func (s *Service) GetBriefing(ctx context.Context, userID int64) (*BriefingDTO, error) {
	var (
		holdings    []model.Holding
		analyses    map[string]*model.AssetAnalysis
		latestCards map[int64]*model.DecisionCard
		sparklines  map[string][]float64
	)

	eg, egCtx := errgroup.WithContext(ctx)

	// Step 1: query active holdings.
	eg.Go(func() error {
		var err error
		holdings, err = s.holdingRepo.ListHoldingsByUser(egCtx, userID)
		if err != nil {
			return fmt.Errorf("list holdings: %w", err)
		}
		return nil
	})

	// Steps 2-4 are launched after step 1 completes because they depend on
	// the holding data. We wait for holdings first, then launch the rest.
	// To achieve parallelism after holdings load, we use a two-phase approach:
	// the errgroup above only covers step 1; after Wait() we launch a second
	// group for steps 2-4.
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if len(holdings) == 0 {
		return &BriefingDTO{Cards: []BriefingCardDTO{}, UpdatedAt: time.Now().UTC()}, nil
	}

	// Collect asset codes and holding IDs for batch queries.
	assetCodes := make([]string, 0, len(holdings))
	holdingIDs := make([]int64, 0, len(holdings))
	for _, h := range holdings {
		assetCodes = append(assetCodes, h.AssetCode)
		holdingIDs = append(holdingIDs, h.HoldingID)
	}

	// Phase 2: parallel batch queries (steps 2-4).
	eg2, egCtx2 := errgroup.WithContext(ctx)

	// Step 2: batch query latest analyses.
	eg2.Go(func() error {
		var err error
		analyses, err = s.analysisRepo.GetLatestByAssetCodes(egCtx2, assetCodes)
		if err != nil {
			return fmt.Errorf("batch query analyses: %w", err)
		}
		return nil
	})

	// Step 3: batch query latest decision cards.
	eg2.Go(func() error {
		var err error
		latestCards, err = s.cardRepo.GetLatestByHoldings(egCtx2, holdingIDs)
		if err != nil {
			return fmt.Errorf("batch query decision cards: %w", err)
		}
		return nil
	})

	// Step 4: query sparkline scores for each unique asset code.
	eg2.Go(func() error {
		m := make(map[string][]float64, len(assetCodes))
		for _, code := range assetCodes {
			scores, err := s.analysisRepo.GetSparklineScores(egCtx2, code, sparklineWindow)
			if err != nil {
				s.logger.Warn("failed to fetch sparkline scores",
					zap.String("asset_code", code),
					zap.Error(err),
				)
				scores = []float64{}
			}
			m[code] = scores
		}
		sparklines = m
		return nil
	})

	if err := eg2.Wait(); err != nil {
		return nil, err
	}

	// Step 5-7: calculate P&L, concentration, and assemble cards.
	cards := make([]BriefingCardDTO, 0, len(holdings))
	for _, h := range holdings {
		card := s.buildCard(h, analyses, latestCards, sparklines)
		cards = append(cards, card)
	}

	return &BriefingDTO{
		Cards:     cards,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// buildCard assembles a BriefingCardDTO from a holding and the pre-fetched maps.
func (s *Service) buildCard(
	h model.Holding,
	analyses map[string]*model.AssetAnalysis,
	cards map[int64]*model.DecisionCard,
	sparklines map[string][]float64,
) BriefingCardDTO {
	card := BriefingCardDTO{
		HoldingID:       h.HoldingID,
		AssetCode:       h.AssetCode,
		AssetName:       h.AssetName,
		AssetType:       h.AssetType,
		CostPrice:       h.CostPrice.String(),
		PositionRatio:   h.PositionRatio.String(),
		Quantity:        h.Quantity.String(),
		SparklineScores: []float64{},
	}

	// Attach sparkline.
	if sc, ok := sparklines[h.AssetCode]; ok && len(sc) > 0 {
		card.SparklineScores = sc
	}

	// Attach analysis data.
	if a, ok := analyses[h.AssetCode]; ok {
		card.OverallScore = &a.OverallScore
		card.SignalLevel = &a.SignalLevel
		card.ScoreDelta = a.ScoreDelta
		card.AnalyzedAt = &a.AnalyzedAt
		id := a.AssetAnalysisID
		card.AssetAnalysisID = &id
		card.ChangeAttribution = a.ChangeSummary
		card.ConflictWarning = a.ConflictMessage
		card.Direction = deriveDirection(a.SignalLevel, a.OverallScore)

		// Step 5: calculate P&L using price_at_analysis vs cost_price.
		if a.PriceAtAnalysis != nil && !h.CostPrice.IsZero() {
			current := *a.PriceAtAnalysis
			cost, _ := h.CostPrice.Float64()
			card.CurrentPrice = &current
			if cost > 0 {
				pnl := (current - cost) / cost * 100
				card.PnLPercent = &pnl
			}
		}
	} else {
		// No analysis yet: default direction to neutral so the frontend has a
		// deterministic string to render.
		card.Direction = "neutral"
	}

	// Attach latest decision card data.
	if dc, ok := cards[h.HoldingID]; ok {
		card.LatestCardID = &dc.CardID
		card.ActionLevel = &dc.ActionLevel
	}

	// Step 6: compute concentration level for this holding's asset type.
	// Sum of position_ratio for all holdings of the same asset_type is not
	// available here without an extra query; we approximate using this
	// holding's own position_ratio and the ComputeConcentration thresholds.
	// The handler layer may enrich this with the full exposure query.
	posRatio, _ := h.PositionRatio.Float64()
	card.ConcentrationLevel, card.ConcentrationMessage = ComputeConcentration(posRatio)

	return card
}

// deriveDirection returns a simple trend label based on the richson signal
// level first, falling back to the overall score when the signal is missing
// or unrecognised. The frontend uses this value to render the sparkline trend
// badge on each briefing card.
//
// Signal levels produced by richson (see core.scoring.signal_level_from_score):
// strong_bullish / moderate_bullish -> bullish
// strong_bearish / moderate_bearish -> bearish
// neutral / unknown -> score-based fallback.
func deriveDirection(signalLevel string, overallScore float64) string {
	switch signalLevel {
	case "strong_bullish", "moderate_bullish":
		return "bullish"
	case "strong_bearish", "moderate_bearish":
		return "bearish"
	}
	switch {
	case overallScore >= 60:
		return "bullish"
	case overallScore <= 40:
		return "bearish"
	default:
		return "neutral"
	}
}

// ComputeConcentration returns a concentration level and a descriptive message
// based on the sum of position_ratio for an asset type.
// Thresholds: >30% red, >20% orange, >10% blue, else green.
func ComputeConcentration(totalExposure float64) (concentrationLevel, string) {
	switch {
	case totalExposure > 30:
		return concentrationRed, fmt.Sprintf("exposure %.1f%% exceeds 30%% threshold, high concentration risk", totalExposure)
	case totalExposure > 20:
		return concentrationOrange, fmt.Sprintf("exposure %.1f%% exceeds 20%% threshold, moderate concentration risk", totalExposure)
	case totalExposure > 10:
		return concentrationBlue, fmt.Sprintf("exposure %.1f%% exceeds 10%% threshold, low concentration risk", totalExposure)
	default:
		return concentrationGreen, fmt.Sprintf("exposure %.1f%% within acceptable range", totalExposure)
	}
}
