package confidence

import "github.com/richman/backend/internal/analysis"

// Calculator computes the confidence score for an analysis result.
type Calculator struct{}

// NewCalculator creates a new confidence calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Input holds the parameters needed to compute confidence.
type Input struct {
	Trend          *analysis.TrendResult
	Position       *analysis.PositionResult
	Catalyst       *analysis.CatalystResult
	HasLLMCatalyst bool // whether LLM enhancement was available
}

// Calculate returns a confidence score from 0 to 100.
//
// Scoring rules (per PRD 3.2.5):
//   - 3 dimensions aligned: base 80-100
//   - 2 aligned, 1 conflict: base 50-70
//   - All different directions: base 20-40
//   - Missing dimension: -20 per missing
//   - No LLM catalyst enhancement: -10
func (c *Calculator) Calculate(input Input) float64 {
	missingCount := 0
	if input.Trend == nil {
		missingCount++
	}
	if input.Position == nil {
		missingCount++
	}
	if input.Catalyst == nil {
		missingCount++
	}

	// If all missing, return minimum
	if missingCount == 3 {
		return 0
	}

	// Normalize directions to a common scale: bullish/upward = +1, bearish/downward = -1, neutral/sideways = 0
	directions := make([]int, 0, 3)
	if input.Trend != nil {
		directions = append(directions, normalizeDirection(input.Trend.Direction))
	}
	if input.Position != nil {
		directions = append(directions, normalizeDirection(input.Position.Assessment))
	}
	if input.Catalyst != nil {
		directions = append(directions, normalizeDirection(input.Catalyst.Direction))
	}

	base := calcBaseConfidence(directions)

	// Deductions
	penalty := float64(missingCount) * 20
	if !input.HasLLMCatalyst {
		penalty += 10
	}

	result := base - penalty
	if result < 0 {
		result = 0
	}
	if result > 100 {
		result = 100
	}

	return result
}

// normalizeDirection maps various direction types to a common scale.
func normalizeDirection(d analysis.Direction) int {
	switch d {
	case analysis.DirectionUpward, analysis.DirectionBullish:
		return 1
	case analysis.DirectionDownward, analysis.DirectionBearish:
		return -1
	default:
		return 0
	}
}

// calcBaseConfidence determines the base confidence from direction alignment.
func calcBaseConfidence(directions []int) float64 {
	if len(directions) <= 1 {
		return 50 // single dimension, moderate confidence
	}

	if len(directions) == 2 {
		if directions[0] == directions[1] {
			return 70 // two aligned (missing third)
		}
		if directions[0] == -directions[1] && directions[0] != 0 {
			return 40 // two conflicting
		}
		return 55 // one neutral
	}

	// 3 directions
	a, b, d := directions[0], directions[1], directions[2]

	allSame := a == b && b == d
	if allSame && a != 0 {
		return 90 // all aligned non-neutral
	}
	if allSame && a == 0 {
		return 60 // all neutral
	}

	// Count alignment
	matches := 0
	if a == b {
		matches++
	}
	if b == d {
		matches++
	}
	if a == d {
		matches++
	}

	if matches >= 1 {
		// At least 2 aligned, 1 different
		return 60
	}

	// All different
	return 30
}
