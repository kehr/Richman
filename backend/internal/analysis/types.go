package analysis

import "time"

// Direction represents the directional assessment of a dimension.
type Direction string

const (
	DirectionUpward   Direction = "upward"
	DirectionSideways Direction = "sideways"
	DirectionDownward Direction = "downward"
	DirectionBullish  Direction = "bullish"
	DirectionNeutral  Direction = "neutral"
	DirectionBearish  Direction = "bearish"
)

// Recommendation represents the final action recommendation.
type Recommendation string

const (
	RecommendAggressiveAdd   Recommendation = "aggressive_add"
	RecommendSmallAdd        Recommendation = "small_add"
	RecommendHold            Recommendation = "hold"
	RecommendGradualReduce   Recommendation = "gradual_reduce"
	RecommendControlPosition Recommendation = "control_position"
)

// TrendResult holds the output of the trend dimension analysis.
type TrendResult struct {
	Direction Direction          // upward, sideways, downward
	Strength  float64            // 0.0-1.0
	Summary   string             // one sentence
	Signals   map[string]float64 // MA, RSI, MACD values
}

// PositionResult holds the output of the position (valuation) dimension analysis.
type PositionResult struct {
	Assessment Direction // bullish (undervalued), neutral (fair), bearish (overvalued)
	Percentile float64   // 0.0-1.0, current valuation percentile in history
	Summary    string
	Metrics    map[string]float64
}

// CatalystResult holds the output of the catalyst dimension analysis.
type CatalystResult struct {
	Direction Direction // bullish, neutral, bearish
	Score     float64   // -1.0 to 1.0
	Summary   string
	Events    []EventSummary
}

// EventSummary describes a single catalyst event.
type EventSummary struct {
	Title       string
	Probability float64
	Impact      string // positive, negative, neutral
}

// WeightConfig holds the percentage weights for each dimension.
type WeightConfig struct {
	Trend    float64 // 0.0-1.0
	Position float64 // 0.0-1.0
	Catalyst float64 // 0.0-1.0
}

// AnalysisResult is the full three-dimension analysis output for an asset.
type AnalysisResult struct {
	AssetCode      string
	AssetType      string
	Trend          TrendResult
	Position       PositionResult
	Catalyst       CatalystResult
	Weights        WeightConfig
	Confidence     float64 // 0-100
	Recommendation Recommendation
	AnalyzedAt     time.Time
}
