package market

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"github.com/richman/backend/internal/richson"
	"go.uber.org/zap"
)

// percentileCacheEntry stores a cached percentile label with an expiry time.
type percentileCacheEntry struct {
	label     *string
	expiresAt time.Time
}

// ohlcvCacheEntry stores a derived OHLCV snapshot with an expiry time. Only
// successful fetches are cached; failures are intentionally not memoised so a
// transient richson hiccup does not lock out fresh data for a full TTL.
type ohlcvCacheEntry struct {
	snapshot  *ohlcvSnapshot
	expiresAt time.Time
}

// ohlcvSnapshot carries the OHLCV-derived fields the asset detail DTO needs.
// Fields are pointers so the JSON layer can omit them when richson is
// unavailable.
type ohlcvSnapshot struct {
	Currency           string
	CurrentPrice       *float64
	PriceChangePercent *float64
	SMA200             *float64
	Supports           []float64
	Resistances        []float64
}

const (
	ohlcvCacheTTL = 60 * time.Second
	// dimensionDisclaimerEN is the English fallback disclaimer injected into
	// every execution plan. Frontend translates via i18n key when available.
	executionPlanDisclaimerEN = "This plan is for research only and is not investment advice."
)

// AssetCardDTO represents a single asset card in the market overview.
type AssetCardDTO struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	NameEn       string   `json:"nameEn"`
	AssetType    string   `json:"assetType"`
	Exchange     string   `json:"exchange"`
	OverallScore *float64 `json:"overallScore,omitempty"`
	SignalLevel  *string  `json:"signalLevel,omitempty"`
	ScoreDelta   *float64 `json:"scoreDelta,omitempty"`
}

// AssetGroupDTO groups asset cards by asset_type.
type AssetGroupDTO struct {
	AssetType string         `json:"assetType"`
	Assets    []AssetCardDTO `json:"assets"`
}

// MarketOverviewDTO is the response payload for GET /v2/market/overview.
type MarketOverviewDTO struct {
	Groups    []AssetGroupDTO `json:"groups"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

// DimensionSubIndicatorDTO mirrors frontend DimensionSubIndicator. RawValue is
// emitted as a number; sub_indicator strings are not exposed here because the
// frontend already knows the indicator vocabulary by name.
type DimensionSubIndicatorDTO struct {
	Name            string   `json:"name"`
	RawValue        float64  `json:"rawValue"`
	Percentile      *float64 `json:"percentile"`
	NormalizedScore float64  `json:"normalizedScore"`
	Weight          float64  `json:"weight"`
}

// DimensionDTO mirrors frontend DimensionDetailDto. Summary/llmReason are not
// populated yet (richson does not write per-dimension narratives today); the
// fields exist so the contract stays stable when richson backfills them.
type DimensionDTO struct {
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	Score         float64                    `json:"score"`
	QuantScore    *float64                   `json:"quantScore"`
	LLMAdjustment *float64                   `json:"llmAdjustment"`
	Signal        string                     `json:"signal"`
	Weight        float64                    `json:"weight"`
	Summary       string                     `json:"summary"`
	LLMReason     *string                    `json:"llmReason"`
	SubIndicators []DimensionSubIndicatorDTO `json:"subIndicators"`
}

// RiskFactorDTO mirrors frontend RiskFactorDto. richson today writes only the
// description string; severity is hardcoded to "medium" until richson upgrades
// the schema (tracked in richson enhancement backlog).
type RiskFactorDTO struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// KeyPriceLevelDTO mirrors frontend KeyPriceLevelDto. distancePct is computed
// relative to currentPrice; when currentPrice is unavailable the value is 0
// and the frontend should hide the percentage label.
type KeyPriceLevelDTO struct {
	Type        string  `json:"type"`
	Price       float64 `json:"price"`
	DistancePct float64 `json:"distancePct"`
	Currency    string  `json:"currency"`
}

// DrawdownReferenceDTO mirrors frontend DrawdownReferenceDto. Field names use
// the camelCase produced by richson/src/richson/core/drawdown.py; we re-tag
// here to match the slimmer frontend interface.
type DrawdownReferenceDTO struct {
	CurrentBullMaxDrawdown     float64 `json:"currentBullMaxDrawdown"`
	CurrentBullMaxDrawdownDate string  `json:"currentBullMaxDrawdownDate"`
	HistoricalAvgDrawdown      float64 `json:"historicalAvgDrawdown"`
}

// MajorChangeRecapDTO mirrors frontend MajorChangeRecapDto. previousScore is
// reached either via prev_analysis_id lookup or, when that lookup fails, by
// subtracting score_delta from the current score.
type MajorChangeRecapDTO struct {
	Text          string  `json:"text"`
	ScoreDelta    float64 `json:"scoreDelta"`
	PreviousScore float64 `json:"previousScore"`
	CurrentScore  float64 `json:"currentScore"`
}

// ExecutionScenarioDTO mirrors frontend ExecutionScenarioDto.
type ExecutionScenarioDTO struct {
	ID        string `json:"id"`
	Priority  int    `json:"priority"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
}

// ExecutionPlanDTO mirrors frontend ExecutionPlanDto. validDays is emitted as
// 0 when richson did not write it, matching the frontend's "treat absent as
// fallback default" handling.
type ExecutionPlanDTO struct {
	Recommendation       string                 `json:"recommendation"`
	DefaultAdvice        string                 `json:"defaultAdvice"`
	StopLoss             *float64               `json:"stopLoss"`
	TakeProfit           *float64               `json:"takeProfit"`
	ValidDays            int                    `json:"validDays"`
	ConcentrationWarning *string                `json:"concentrationWarning"`
	Scenarios            []ExecutionScenarioDTO `json:"scenarios"`
	Disclaimer           string                 `json:"disclaimer"`
}

// AssetDetailDTO is the response payload for GET /v2/market/:code. All
// optional fields use pointer types with omitempty so the frontend treats
// them as absent-when-undefined per docs/standards/contract-drift.md.
type AssetDetailDTO struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	NameEn    string `json:"nameEn"`
	AssetType string `json:"assetType"`
	Exchange  string `json:"exchange"`

	Currency           *string  `json:"currency,omitempty"`
	UsdExchangeRate    *float64 `json:"usdExchangeRate,omitempty"`
	CurrentPrice       *float64 `json:"currentPrice,omitempty"`
	PriceChangePercent *float64 `json:"priceChangePercent,omitempty"`
	PriceAtAnalysis    *float64 `json:"priceAtAnalysis,omitempty"`

	OverallScore         *float64 `json:"overallScore,omitempty"`
	ScoreBandLow         *float64 `json:"scoreBandLow,omitempty"`
	ScoreBandHigh        *float64 `json:"scoreBandHigh,omitempty"`
	SignalLevel          *string  `json:"signalLevel,omitempty"`
	PercentileLabel      *string  `json:"percentileLabel,omitempty"`
	MarketInterpretation *string  `json:"marketInterpretation,omitempty"`
	ScoreDelta           *float64 `json:"scoreDelta,omitempty"`
	ChangeSummary        *string  `json:"changeSummary,omitempty"`
	MajorChangeRecap     *MajorChangeRecapDTO `json:"majorChangeRecap,omitempty"`
	ConflictType         *string  `json:"conflictType,omitempty"`
	ConflictMessage      *string  `json:"conflictMessage,omitempty"`
	AnalyzedAt           *time.Time `json:"analyzedAt,omitempty"`
	ValidDays            *int       `json:"validDays,omitempty"`

	Dimensions        []DimensionDTO        `json:"dimensions"`
	RiskFactors       []RiskFactorDTO       `json:"riskFactors,omitempty"`
	KeyPriceLevels    []KeyPriceLevelDTO    `json:"keyPriceLevels,omitempty"`
	DrawdownReference *DrawdownReferenceDTO `json:"drawdownReference,omitempty"`
	ExecutionPlan     *ExecutionPlanDTO     `json:"executionPlan,omitempty"`

	Supports    []float64 `json:"supports,omitempty"`
	Resistances []float64 `json:"resistances,omitempty"`
	SMA200      *float64  `json:"sma200,omitempty"`

	// AnalysisID is retained for backward compatibility; the frontend DTO no
	// longer declares it but older clients still consume it.
	AnalysisID *int64 `json:"analysisId,omitempty"`
}

// Service provides market overview and asset detail queries.
type Service struct {
	assetRepo      *repo.AssetRepo
	analysisRepo   *repo.AssetAnalysisReadRepo
	dimensionRepo  *repo.AnalysisDimensionReadRepo
	richsonClient  *richson.Client
	logger         *zap.Logger

	// percentile label cache keyed by asset_code, TTL 1hr.
	cacheMu sync.Mutex
	cache   map[string]percentileCacheEntry

	// OHLCV snapshot cache keyed by asset_code, TTL 60s. Failures are not
	// cached so a transient richson outage does not shadow fresh data.
	ohlcvMu    sync.Mutex
	ohlcvCache map[string]ohlcvCacheEntry
}

// NewService constructs a market Service. richsonClient may be nil in tests
// or when richson integration is intentionally disabled; OHLCV-derived fields
// silently degrade in that case.
func NewService(
	assetRepo *repo.AssetRepo,
	analysisRepo *repo.AssetAnalysisReadRepo,
	dimensionRepo *repo.AnalysisDimensionReadRepo,
	richsonClient *richson.Client,
	logger *zap.Logger,
) *Service {
	return &Service{
		assetRepo:     assetRepo,
		analysisRepo:  analysisRepo,
		dimensionRepo: dimensionRepo,
		richsonClient: richsonClient,
		logger:        logger,
		cache:         make(map[string]percentileCacheEntry),
		ohlcvCache:    make(map[string]ohlcvCacheEntry),
	}
}

// GetOverview returns all active assets grouped by asset_type, each with the
// latest analysis scores. Assets with no analysis are included without scores.
func (s *Service) GetOverview(ctx context.Context) (*MarketOverviewDTO, error) {
	assets, err := s.assetRepo.ListActiveWithType(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("list active assets: %w", err)
	}

	codes := make([]string, 0, len(assets))
	for _, a := range assets {
		codes = append(codes, a.Code)
	}

	analyses, err := s.analysisRepo.GetLatestByAssetCodes(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("batch query analyses: %w", err)
	}

	// Group by asset_type while preserving the original sort order.
	groupMap := make(map[string]*AssetGroupDTO)
	groupOrder := make([]string, 0)

	for _, a := range assets {
		if _, exists := groupMap[a.AssetType]; !exists {
			groupMap[a.AssetType] = &AssetGroupDTO{AssetType: a.AssetType, Assets: []AssetCardDTO{}}
			groupOrder = append(groupOrder, a.AssetType)
		}
		card := AssetCardDTO{
			Code:      a.Code,
			Name:      a.Name,
			NameEn:    a.NameEn,
			AssetType: a.AssetType,
			Exchange:  a.Exchange,
		}
		if analysis, ok := analyses[a.Code]; ok {
			card.OverallScore = &analysis.OverallScore
			card.SignalLevel = &analysis.SignalLevel
			card.ScoreDelta = analysis.ScoreDelta
		}
		groupMap[a.AssetType].Assets = append(groupMap[a.AssetType].Assets, card)
	}

	groups := make([]AssetGroupDTO, 0, len(groupOrder))
	for _, t := range groupOrder {
		groups = append(groups, *groupMap[t])
	}

	return &MarketOverviewDTO{
		Groups:    groups,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// GetAssetDetail returns a single asset with the latest analysis, all
// dimension records, OHLCV-derived price metadata, and a percentile label
// derived from 1-year score history. The DTO is a superset of the frontend
// AssetDetailDto contract; missing data sources degrade individual fields
// rather than failing the whole request.
func (s *Service) GetAssetDetail(ctx context.Context, code string) (*AssetDetailDTO, error) {
	asset, err := s.assetRepo.GetAssetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("query asset: %w", err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset %q not found", code)
	}

	detail := &AssetDetailDTO{
		Code:       asset.Code,
		Name:       asset.Name,
		NameEn:     asset.NameEn,
		AssetType:  asset.AssetType,
		Exchange:   asset.Exchange,
		Dimensions: []DimensionDTO{},
	}

	analysis, err := s.analysisRepo.GetLatestByAssetCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("query latest analysis: %w", err)
	}

	// OHLCV is independent of analysis presence; fetch in parallel only when
	// useful (skip when richson client is absent).
	ohlcv := s.fetchOHLCVForDetail(ctx, code)

	// Determine currency: OHLCV is the source of truth; fall back to asset
	// type when richson is unavailable.
	currency := inferCurrency(ohlcv, asset.AssetType)
	detail.Currency = &currency

	if ohlcv != nil {
		if ohlcv.CurrentPrice != nil {
			cp := *ohlcv.CurrentPrice
			detail.CurrentPrice = &cp
		}
		if ohlcv.PriceChangePercent != nil {
			pc := *ohlcv.PriceChangePercent
			detail.PriceChangePercent = &pc
		}
		if ohlcv.SMA200 != nil {
			sm := *ohlcv.SMA200
			detail.SMA200 = &sm
		}
		if len(ohlcv.Supports) > 0 {
			detail.Supports = append([]float64(nil), ohlcv.Supports...)
		}
		if len(ohlcv.Resistances) > 0 {
			detail.Resistances = append([]float64(nil), ohlcv.Resistances...)
		}
	}

	if analysis == nil {
		// No analysis yet but asset exists: return basic metadata + whatever
		// OHLCV gave us so the frontend can still render the "awaiting
		// analysis" state with price context.
		return detail, nil
	}

	detail.OverallScore = &analysis.OverallScore
	detail.SignalLevel = &analysis.SignalLevel
	detail.ScoreDelta = analysis.ScoreDelta
	detail.ChangeSummary = analysis.ChangeSummary
	detail.ConflictType = analysis.ConflictType
	detail.ConflictMessage = analysis.ConflictMessage
	detail.AnalyzedAt = &analysis.AnalyzedAt
	detail.AnalysisID = &analysis.AssetAnalysisID
	detail.UsdExchangeRate = analysis.UsdExchangeRate
	detail.PriceAtAnalysis = analysis.PriceAtAnalysis

	scoreBandLow := analysis.ConfidenceBandLow
	scoreBandHigh := analysis.ConfidenceBandHigh
	detail.ScoreBandLow = &scoreBandLow
	detail.ScoreBandHigh = &scoreBandHigh

	if analysis.MarketInterpretation != "" {
		mi := analysis.MarketInterpretation
		detail.MarketInterpretation = &mi
	}

	dims, err := s.dimensionRepo.GetByAnalysisID(ctx, analysis.AssetAnalysisID)
	if err != nil {
		return nil, fmt.Errorf("query dimensions: %w", err)
	}
	detail.Dimensions = s.buildDimensions(analysis, dims)

	detail.PercentileLabel = s.getPercentileLabel(ctx, code)

	// JSONB derived fields — each helper handles nil/decode failure on its own.
	detail.RiskFactors = s.buildRiskFactors(analysis.RiskFactors)
	detail.ExecutionPlan = s.buildExecutionPlan(analysis.DemoPlan)
	if detail.ExecutionPlan != nil && detail.ExecutionPlan.ValidDays > 0 {
		vd := detail.ExecutionPlan.ValidDays
		detail.ValidDays = &vd
	}

	meta := unmarshalAnalysisMetadata(analysis.AnalysisMetadata)
	if meta != nil {
		detail.DrawdownReference = buildDrawdownReference(meta.DrawdownReference)
		// Fall back to analysis_metadata levels when OHLCV did not supply them.
		if len(detail.Supports) == 0 && len(meta.SupportLevels) > 0 {
			detail.Supports = append([]float64(nil), meta.SupportLevels...)
		}
		if len(detail.Resistances) == 0 && len(meta.ResistanceLevels) > 0 {
			detail.Resistances = append([]float64(nil), meta.ResistanceLevels...)
		}
	}

	detail.KeyPriceLevels = s.buildKeyPriceLevels(detail.Supports, detail.Resistances, detail.CurrentPrice, currency)
	detail.MajorChangeRecap = s.buildMajorChangeRecap(ctx, analysis)

	return detail, nil
}

// inferCurrency returns the OHLCV-supplied currency when present, otherwise
// derives it from asset_type so the frontend always has a non-empty value.
func inferCurrency(snap *ohlcvSnapshot, assetType string) string {
	if snap != nil && snap.Currency != "" {
		return snap.Currency
	}
	if assetType == "stock-cn" {
		return "CNY"
	}
	return "USD"
}

// buildDimensions emits exactly four DimensionDTO entries (d1..d4) regardless
// of whether richson populated them. Sub-indicator rows are bucketed by their
// `dimension` column ("d1".."d4") and projected into the frontend shape.
func (s *Service) buildDimensions(analysis *model.AssetAnalysis, dims []model.AnalysisDimension) []DimensionDTO {
	if analysis == nil {
		return []DimensionDTO{}
	}

	// Bucket sub-indicators by dimension key ("d1".."d4").
	byDim := make(map[string][]DimensionSubIndicatorDTO, 4)
	for _, d := range dims {
		sub := DimensionSubIndicatorDTO{
			Name:            d.SubIndicator,
			RawValue:        derefFloat(d.RawValue),
			Percentile:      pickPercentile(d.BlendedPercentile, d.Percentile1Y),
			NormalizedScore: derefFloat(d.NormalizedScore),
			Weight:          derefFloat(d.WeightInDimension),
		}
		byDim[d.Dimension] = append(byDim[d.Dimension], sub)
	}

	type dimSpec struct {
		id       string
		nameEN   string
		score    *float64
		base     *float64
		llm      *float64
		weight   float64
	}
	specs := []dimSpec{
		{id: "d1", nameEN: "Macro", score: analysis.D1Score, base: analysis.D1BaseScore, llm: analysis.D1LLMAdjustment, weight: analysis.D1Weight},
		{id: "d2", nameEN: "Liquidity", score: analysis.D2Score, base: analysis.D2BaseScore, llm: analysis.D2LLMAdjustment, weight: analysis.D2Weight},
		{id: "d3", nameEN: "Sentiment", score: analysis.D3Score, base: analysis.D3BaseScore, llm: analysis.D3LLMAdjustment, weight: analysis.D3Weight},
		{id: "d4", nameEN: "Technical", score: analysis.D4Score, base: analysis.D4BaseScore, llm: nil, weight: analysis.D4Weight},
	}

	out := make([]DimensionDTO, 0, len(specs))
	for _, sp := range specs {
		subs := byDim[sp.id]
		if subs == nil {
			subs = []DimensionSubIndicatorDTO{}
		}
		out = append(out, DimensionDTO{
			ID:            sp.id,
			Name:          sp.nameEN,
			Score:         derefFloat(sp.score),
			QuantScore:    sp.base,
			LLMAdjustment: sp.llm,
			Signal:        s.deriveDimensionSignal(sp.score),
			Weight:        sp.weight,
			Summary:       "",
			LLMReason:     nil,
			SubIndicators: subs,
		})
	}
	return out
}

// deriveDimensionSignal buckets a 0-100 score into bullish/neutral/bearish
// using the same thresholds richson applies in signal_level_from_score (>=60
// bullish / 40-60 neutral / <40 bearish). Nil scores fall back to neutral so
// the frontend never sees an empty signal string.
func (s *Service) deriveDimensionSignal(score *float64) string {
	if score == nil {
		return "neutral"
	}
	switch {
	case *score >= 60:
		return "bullish"
	case *score >= 40:
		return "neutral"
	default:
		return "bearish"
	}
}

// buildRiskFactors wraps the flat []string richson currently writes into the
// structured DTO the frontend consumes. Severity is hardcoded "medium" until
// richson upgrades the schema (TRD risk table item).
func (s *Service) buildRiskFactors(raw []byte) []RiskFactorDTO {
	factors := unmarshalRiskFactors(raw)
	if len(factors) == 0 {
		return nil
	}
	out := make([]RiskFactorDTO, 0, len(factors))
	for i, f := range factors {
		if f == "" {
			continue
		}
		out = append(out, RiskFactorDTO{
			ID:          fmt.Sprintf("rf-%d", i+1),
			Description: f,
			Severity:    "medium",
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildExecutionPlan decodes richson's snake_case demo_plan JSONB and projects
// it into the camelCase frontend ExecutionPlanDto. Returns nil when the JSONB
// is missing or cannot be decoded; callers omit the field in that case.
func (s *Service) buildExecutionPlan(raw []byte) *ExecutionPlanDTO {
	dp := unmarshalDemoPlan(raw)
	if dp == nil {
		return nil
	}
	scenarios := make([]ExecutionScenarioDTO, 0, len(dp.Scenarios))
	for i, sc := range dp.Scenarios {
		scenarios = append(scenarios, ExecutionScenarioDTO{
			ID:        fmt.Sprintf("scenario-%d", i+1),
			Priority:  sc.Priority,
			Condition: sc.Condition,
			Action:    sc.Action,
			Rationale: sc.Rationale,
		})
	}
	validDays := 0
	if dp.ValidDays != nil {
		validDays = *dp.ValidDays
	}
	return &ExecutionPlanDTO{
		Recommendation:       dp.ActionLabel,
		DefaultAdvice:        dp.DefaultAction,
		StopLoss:             dp.StopLoss,
		TakeProfit:           dp.TakeProfit,
		ValidDays:            validDays,
		ConcentrationWarning: dp.ConcentrationMessage,
		Scenarios:            scenarios,
		Disclaimer:           executionPlanDisclaimerEN,
	}
}

// buildKeyPriceLevels merges supports and resistances into a single sorted
// slice. Supports are sorted by ascending distance from the current price
// (closest first); resistances follow the same rule. When currentPrice is
// missing the distance is 0 and the original ordering is preserved.
func (s *Service) buildKeyPriceLevels(supports, resistances []float64, currentPrice *float64, currency string) []KeyPriceLevelDTO {
	if len(supports) == 0 && len(resistances) == 0 {
		return nil
	}
	out := make([]KeyPriceLevelDTO, 0, len(supports)+len(resistances))

	add := func(typ string, prices []float64) {
		bucket := make([]KeyPriceLevelDTO, 0, len(prices))
		for _, p := range prices {
			bucket = append(bucket, KeyPriceLevelDTO{
				Type:        typ,
				Price:       p,
				DistancePct: distancePercent(p, currentPrice),
				Currency:    currency,
			})
		}
		// Sort by absolute distance ascending so the closest level is first.
		sort.SliceStable(bucket, func(i, j int) bool {
			return absFloat(bucket[i].DistancePct) < absFloat(bucket[j].DistancePct)
		})
		out = append(out, bucket...)
	}
	add("support", supports)
	add("resistance", resistances)
	return out
}

// buildMajorChangeRecap reaches the previous overall_score either via
// prev_analysis_id lookup or, when the lookup misses, by subtracting
// score_delta from the current score. Returns nil when richson did not write
// a recap text — the frontend hides the section in that case.
func (s *Service) buildMajorChangeRecap(ctx context.Context, analysis *model.AssetAnalysis) *MajorChangeRecapDTO {
	if analysis == nil || analysis.MajorChangeRecap == nil || *analysis.MajorChangeRecap == "" {
		return nil
	}
	scoreDelta := derefFloat(analysis.ScoreDelta)
	currentScore := analysis.OverallScore
	previousScore := currentScore - scoreDelta
	if analysis.PrevAnalysisID != nil {
		prev, err := s.analysisRepo.GetByID(ctx, *analysis.PrevAnalysisID)
		if err != nil {
			s.logger.Warn("failed to fetch prev analysis for major change recap",
				zap.String("asset_code", analysis.AssetCode),
				zap.Int64("prev_analysis_id", *analysis.PrevAnalysisID),
				zap.Error(err),
			)
		} else if prev != nil {
			previousScore = prev.OverallScore
		}
	}
	return &MajorChangeRecapDTO{
		Text:          *analysis.MajorChangeRecap,
		ScoreDelta:    scoreDelta,
		PreviousScore: previousScore,
		CurrentScore:  currentScore,
	}
}

// buildDrawdownReference projects the snake_case-keyed structure (despite the
// json tags, the underlying source fields are camelCase per richson) into the
// slimmer frontend DTO. Returns nil when the source struct is absent or the
// required numeric fields are missing.
func buildDrawdownReference(raw *rawDrawdownReference) *DrawdownReferenceDTO {
	if raw == nil {
		return nil
	}
	if raw.MaxDrawdown == nil || raw.HistoricalAvgDrawdown == nil {
		return nil
	}
	date := ""
	if raw.MaxDrawdownDate != nil {
		date = *raw.MaxDrawdownDate
	}
	return &DrawdownReferenceDTO{
		CurrentBullMaxDrawdown:     *raw.MaxDrawdown,
		CurrentBullMaxDrawdownDate: date,
		HistoricalAvgDrawdown:      *raw.HistoricalAvgDrawdown,
	}
}

// fetchOHLCVForDetail resolves OHLCV-derived fields with a 60-second
// in-process cache. Cache misses or expired entries hit richson; failures are
// logged at warn level and return nil so the caller can fall back gracefully.
func (s *Service) fetchOHLCVForDetail(ctx context.Context, code string) *ohlcvSnapshot {
	if s.richsonClient == nil {
		return nil
	}

	s.ohlcvMu.Lock()
	entry, ok := s.ohlcvCache[code]
	s.ohlcvMu.Unlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.snapshot
	}

	resp, err := s.richsonClient.GetOHLCV(ctx, code)
	if err != nil {
		s.logger.Warn("failed to fetch ohlcv for asset detail",
			zap.String("asset_code", code),
			zap.Error(err),
		)
		return nil
	}
	if resp == nil {
		return nil
	}

	snap := &ohlcvSnapshot{
		Currency:    resp.Currency,
		SMA200:      resp.SMA200,
		Supports:    append([]float64(nil), resp.SupportLevels...),
		Resistances: append([]float64(nil), resp.ResistanceLevels...),
	}
	if n := len(resp.Candles); n >= 1 {
		latest := resp.Candles[n-1].Close
		snap.CurrentPrice = &latest
		if n >= 2 {
			prev := resp.Candles[n-2].Close
			if prev != 0 {
				pct := (latest - prev) / prev * 100
				snap.PriceChangePercent = &pct
			}
		}
	}

	s.ohlcvMu.Lock()
	s.ohlcvCache[code] = ohlcvCacheEntry{
		snapshot:  snap,
		expiresAt: time.Now().Add(ohlcvCacheTTL),
	}
	s.ohlcvMu.Unlock()
	return snap
}

// distancePercent returns (price - current) / current * 100, or 0 when current
// is nil or zero (the frontend hides the percentage label in that case).
func distancePercent(price float64, current *float64) float64 {
	if current == nil || *current == 0 {
		return 0
	}
	return (price - *current) / *current * 100
}

// pickPercentile prefers the blended percentile (1y+5y mix) over the 1y-only
// figure so the frontend sees the richest signal richson can emit.
func pickPercentile(blended, oneYear *float64) *float64 {
	if blended != nil {
		v := *blended
		return &v
	}
	if oneYear != nil {
		v := *oneYear
		return &v
	}
	return nil
}

// derefFloat returns *p or 0 when p is nil.
func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// absFloat returns the absolute value of f without depending on math import.
func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// getPercentileLabel returns a cached percentile label for an asset, refreshing
// it when the cache entry is absent or expired (1hr TTL). Returns nil when
// fewer than 30 days of history exist (cold start).
func (s *Service) getPercentileLabel(ctx context.Context, code string) *string {
	s.cacheMu.Lock()
	entry, ok := s.cache[code]
	s.cacheMu.Unlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.label
	}

	label := s.computePercentileLabel(ctx, code)

	s.cacheMu.Lock()
	s.cache[code] = percentileCacheEntry{
		label:     label,
		expiresAt: time.Now().Add(time.Hour),
	}
	s.cacheMu.Unlock()

	return label
}

// computePercentileLabel queries the last 365 days of scores and calculates
// where the latest score falls percentile-wise. Returns nil for cold start
// (<30 days). For 30-364 days available, uses a "近N月" label prefix instead
// of "近一年".
func (s *Service) computePercentileLabel(ctx context.Context, code string) *string {
	scores, err := s.analysisRepo.GetScoresForPercentile(ctx, code, 365)
	if err != nil {
		s.logger.Warn("failed to fetch scores for percentile",
			zap.String("asset_code", code),
			zap.Error(err),
		)
		return nil
	}

	n := len(scores)
	if n < 30 {
		// Fewer than 30 data points: insufficient history for a meaningful label.
		return nil
	}

	// Use only the latest score for the percentile calculation.
	latest := scores[n-1]

	// Count how many historical scores are strictly less than the latest.
	lessCount := 0
	// Include all scores except the latest to avoid counting self.
	for _, s := range scores[:n-1] {
		if s < latest {
			lessCount++
		}
	}
	total := n - 1
	if total <= 0 {
		return nil
	}

	// Percentile rank of latest score among historical scores.
	pct := float64(lessCount) / float64(total) * 100

	// Determine time prefix: full year or N months.
	var prefix string
	if n >= 365 {
		prefix = "近一年"
	} else {
		months := n / 30
		if months <= 0 {
			months = 1
		}
		prefix = fmt.Sprintf("近%d月", months)
	}

	var suffix string
	switch {
	case pct >= 90:
		suffix = "偏高"
	case pct >= 75:
		suffix = "中高"
	case pct >= 25:
		suffix = "中位"
	case pct >= 10:
		suffix = "中低"
	default:
		suffix = "偏低"
	}

	label := prefix + suffix
	return &label
}

// InvalidatePercentileCache evicts the cached label for a given asset code.
// Useful for forced refreshes after a new analysis is written.
func (s *Service) InvalidatePercentileCache(code string) {
	s.cacheMu.Lock()
	delete(s.cache, code)
	s.cacheMu.Unlock()
}

// GetLatestAnalysisForDemoPlan returns the latest asset analysis record for the
// given asset code. Used by the demo-plan endpoint to check whether the DB already
// holds a cached demo_plan JSON before falling back to richson.
// Returns nil when no analysis exists (not an error).
func (s *Service) GetLatestAnalysisForDemoPlan(ctx context.Context, code string) (*model.AssetAnalysis, error) {
	return s.analysisRepo.GetLatestByAssetCode(ctx, code)
}
