package analysis

// Matrix maps three-dimension analysis scores to a final recommendation.
type Matrix struct{}

// NewMatrix creates a new decision matrix.
func NewMatrix() *Matrix {
	return &Matrix{}
}

// Decide computes the final recommendation from the three dimension results and weights.
//
// Each dimension is scored on a -1 to +1 scale:
//   - Trend: upward=+1, sideways=0, downward=-1
//   - Position: bullish=+1, neutral=0, bearish=-1
//   - Catalyst: bullish=+1, neutral=0, bearish=-1
//
// The weighted sum maps to a recommendation:
//
//	score > 0.5  -> aggressive_add
//	score > 0.2  -> small_add
//	score > -0.2 -> hold
//	score > -0.5 -> gradual_reduce
//	score <= -0.5 -> control_position
func (m *Matrix) Decide(
	trend TrendResult,
	position PositionResult,
	catalyst CatalystResult,
	weights WeightConfig,
) Recommendation {
	trendScore := directionScore(trend.Direction) * trend.Strength
	positionScore := directionScore(position.Assessment)
	catalystScore := catalyst.Score // already in -1 to 1 range

	// Position score is binary from direction, scale by percentile distance from 0.5
	positionMagnitude := 1.0
	if position.Percentile >= 0.3 && position.Percentile <= 0.7 {
		positionMagnitude = 0.5 // fair value, moderate signal
	}
	positionScore *= positionMagnitude

	weightedScore := trendScore*weights.Trend + positionScore*weights.Position + catalystScore*weights.Catalyst

	return scoreToRecommendation(weightedScore)
}

// directionScore converts a Direction to a numeric score.
func directionScore(d Direction) float64 {
	switch d {
	case DirectionUpward, DirectionBullish:
		return 1.0
	case DirectionDownward, DirectionBearish:
		return -1.0
	default:
		return 0.0
	}
}

// scoreToRecommendation maps a weighted score to a Recommendation.
func scoreToRecommendation(score float64) Recommendation {
	switch {
	case score > 0.5:
		return RecommendAggressiveAdd
	case score > 0.2:
		return RecommendSmallAdd
	case score > -0.2:
		return RecommendHold
	case score > -0.5:
		return RecommendGradualReduce
	default:
		return RecommendControlPosition
	}
}
