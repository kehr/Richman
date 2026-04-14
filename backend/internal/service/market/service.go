package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/richman/backend/internal/model"
	"github.com/richman/backend/internal/repo"
	"go.uber.org/zap"
)

// percentileCacheEntry stores a cached percentile label with an expiry time.
type percentileCacheEntry struct {
	label     *string
	expiresAt time.Time
}

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

// AssetDetailDTO is the response payload for GET /v2/market/assets/{code}.
type AssetDetailDTO struct {
	Code            string                    `json:"code"`
	Name            string                    `json:"name"`
	NameEn          string                    `json:"nameEn"`
	AssetType       string                    `json:"assetType"`
	Exchange        string                    `json:"exchange"`
	OverallScore    *float64                  `json:"overallScore,omitempty"`
	SignalLevel     *string                   `json:"signalLevel,omitempty"`
	ScoreDelta      *float64                  `json:"scoreDelta,omitempty"`
	PercentileLabel *string                   `json:"percentileLabel,omitempty"`
	Dimensions      []model.AnalysisDimension `json:"dimensions"`
	AnalyzedAt      *time.Time                `json:"analyzedAt,omitempty"`
	AnalysisID      *int64                    `json:"analysisId,omitempty"`
}

// Service provides market overview and asset detail queries.
type Service struct {
	assetRepo     *repo.AssetRepo
	analysisRepo  *repo.AssetAnalysisReadRepo
	dimensionRepo *repo.AnalysisDimensionReadRepo
	logger        *zap.Logger

	// percentile label cache keyed by asset_code, TTL 1hr.
	cacheMu sync.Mutex
	cache   map[string]percentileCacheEntry
}

// NewService constructs a market Service.
func NewService(
	assetRepo *repo.AssetRepo,
	analysisRepo *repo.AssetAnalysisReadRepo,
	dimensionRepo *repo.AnalysisDimensionReadRepo,
	logger *zap.Logger,
) *Service {
	return &Service{
		assetRepo:     assetRepo,
		analysisRepo:  analysisRepo,
		dimensionRepo: dimensionRepo,
		logger:        logger,
		cache:         make(map[string]percentileCacheEntry),
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
// dimension records, and a percentile label derived from 1-year score history.
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
		Dimensions: []model.AnalysisDimension{},
	}

	analysis, err := s.analysisRepo.GetLatestByAssetCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("query latest analysis: %w", err)
	}
	if analysis == nil {
		return detail, nil
	}

	detail.OverallScore = &analysis.OverallScore
	detail.SignalLevel = &analysis.SignalLevel
	detail.ScoreDelta = analysis.ScoreDelta
	detail.AnalyzedAt = &analysis.AnalyzedAt
	detail.AnalysisID = &analysis.AssetAnalysisID

	dims, err := s.dimensionRepo.GetByAnalysisID(ctx, analysis.AssetAnalysisID)
	if err != nil {
		return nil, fmt.Errorf("query dimensions: %w", err)
	}
	if dims != nil {
		detail.Dimensions = dims
	}

	detail.PercentileLabel = s.getPercentileLabel(ctx, code)

	return detail, nil
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
