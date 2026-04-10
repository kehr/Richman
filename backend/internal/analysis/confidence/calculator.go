package confidence

import (
	"math"

	"github.com/richman/backend/internal/analysis"
)

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
	HasLLMCatalyst bool                  // whether LLM enhancement was available
	Weights        analysis.WeightConfig // per-asset-type dimension weights
}

// Calculate returns a confidence score from 0 to 100.
//
// Formula: C = round(Coherence×70 + Quality×20 + Strength×10)
//
// The three components are:
//
// Coherence (0-1): weighted agreement of each dimension with the final
// recommendation direction, derived from multi-factor signal aggregation
// theory. For each present dimension i with signal sᵢ ∈ {-1, 0, +1}:
//   - Weighted direction D = sign(Σ wᵢ·sᵢ)
//   - agreementᵢ = (1 + sᵢ·D) / 2  → 1.0 (aligned), 0.5 (neutral), 0.0 (opposing)
//   - When D=0 (pure deadlock): agreementᵢ = 0.5 for all
//   - Coherence = Σᵢ (wᵢ·agreementᵢ) / Σᵢ wᵢ  (normalised over present dims)
//
// Quality (0-1): data completeness × LLM availability.
//   - 3 dimensions: 1.0,  2: 0.6,  1: 0.2
//   - LLM catalyst available: ×1.0,  missing: ×0.9
//
// Strength (0-1): fraction of present dimensions with a non-neutral signal.
// Neutral signals carry less information (Shannon); rewarding decisiveness
// with a modest 10-point weight prevents all-neutral deadlocks from scoring
// the same as decisive but split signals.
//
// The 70:20:10 split ensures coherence dominates — no combination of quality
// and strength bonuses can compensate for fundamentally disagreeing signals.
func (c *Calculator) Calculate(input Input) float64 {
	// Collect present dimensions: direction + weight.
	type dim struct {
		signal float64 // -1, 0, +1
		weight float64
	}
	dims := make([]dim, 0, 3)
	if input.Trend != nil {
		dims = append(dims, dim{normalizeDirection(input.Trend.Direction), input.Weights.Trend})
	}
	if input.Position != nil {
		dims = append(dims, dim{normalizeDirection(input.Position.Assessment), input.Weights.Position})
	}
	if input.Catalyst != nil {
		dims = append(dims, dim{normalizeDirection(input.Catalyst.Direction), input.Weights.Catalyst})
	}

	n := len(dims)
	if n == 0 {
		return 0
	}

	// Compute weighted recommendation direction.
	var weightedSum, totalWeight float64
	for _, d := range dims {
		weightedSum += d.weight * d.signal
		totalWeight += d.weight
	}
	D := math.Copysign(1, weightedSum) // +1 or -1
	if weightedSum == 0 {
		D = 0 // pure deadlock
	}

	// Coherence: weighted agreement with D.
	var coherenceSum float64
	for _, d := range dims {
		var agreement float64
		if D == 0 {
			agreement = 0.5
		} else {
			agreement = (1 + d.signal*D) / 2 // 1.0 aligned, 0.5 neutral, 0.0 opposing
		}
		coherenceSum += d.weight * agreement
	}
	coherence := coherenceSum / totalWeight // normalised

	// Quality: data completeness × LLM factor.
	completeness := map[int]float64{1: 0.2, 2: 0.6, 3: 1.0}[n]
	llmFactor := 1.0
	if !input.HasLLMCatalyst {
		llmFactor = 0.9
	}
	quality := completeness * llmFactor

	// Strength: fraction of non-neutral signals.
	var nonNeutral float64
	for _, d := range dims {
		if d.signal != 0 {
			nonNeutral++
		}
	}
	strength := nonNeutral / float64(n)

	score := coherence*70 + quality*20 + strength*10
	return math.Round(math.Max(0, math.Min(100, score)))
}

// normalizeDirection maps a Direction value to a numeric signal.
//
//	upward / bullish  → +1
//	downward / bearish → -1
//	sideways / neutral / anything else → 0
func normalizeDirection(d analysis.Direction) float64 {
	switch d {
	case analysis.DirectionUpward, analysis.DirectionBullish:
		return 1
	case analysis.DirectionDownward, analysis.DirectionBearish:
		return -1
	default:
		return 0
	}
}
