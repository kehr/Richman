package model

import (
	"encoding/json"
	"time"

	"github.com/richman/backend/internal/analysis/recommendation"
)

// DecisionCard represents a generated investment decision card for a holding.
type DecisionCard struct {
	CardID            int64     `json:"cardId"`
	UserID            int64     `json:"userId"`
	HoldingID         int64     `json:"holdingId"`
	AssetCode         string    `json:"assetCode"`
	AssetName         string    `json:"assetName"`
	AssetType         string    `json:"assetType"`
	CostPrice         float64   `json:"costPrice"`
	PositionRatio     float64   `json:"positionRatio"`
	TrendDirection    string    `json:"trendDirection"`
	TrendSummary      string    `json:"trendSummary"`
	PositionDirection string    `json:"positionDirection"`
	PositionSummary   string    `json:"positionSummary"`
	CatalystDirection string    `json:"catalystDirection"`
	CatalystSummary   string    `json:"catalystSummary"`
	Confidence        float64   `json:"confidence"`
	Recommendation    string    `json:"recommendation"`
	ActionAdvice      string    `json:"actionAdvice"`
	DetailedAdvice    string    `json:"detailedAdvice"`
	RiskWarnings      []string  `json:"riskWarnings"`
	TodayHighlights   string    `json:"todayHighlights"`
	WeightTrend       float64   `json:"weightTrend"`
	WeightPosition    float64   `json:"weightPosition"`
	WeightCatalyst    float64   `json:"weightCatalyst"`
	AnalyzedAt        time.Time `json:"analyzedAt"`
	CreatedAt         time.Time `json:"createdAt"`

	// Structured recommendation and badge diff fields (migration 006).
	// RecommendationJSON is serialized into the recommendation_json JSONB column.
	RecommendationJSON   recommendation.Recommendation `json:"recommendation_detail"`
	ActionLevel          int                           `json:"actionLevel"`
	TargetPositionRatio  float64                       `json:"targetPositionRatio"`
	BadgeState           string                        `json:"badgeState"`
	ConfidenceDelta      float64                       `json:"confidenceDelta"`
	PrevCardID           *int64                        `json:"prevCardId,omitempty"`
	ExecutionFingerprint string                        `json:"executionFingerprint"`
}

// RecommendationDetailJSON returns the structured recommendation as a JSON
// byte slice for DB storage in the recommendation_json JSONB column.
func (d *DecisionCard) RecommendationDetailJSON() ([]byte, error) {
	return json.Marshal(d.RecommendationJSON)
}

// RiskWarningsJSON returns the risk warnings as a JSON byte slice for DB storage.
func (d *DecisionCard) RiskWarningsJSON() ([]byte, error) {
	if d.RiskWarnings == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(d.RiskWarnings)
}
