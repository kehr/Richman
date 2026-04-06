package weight

import (
	"fmt"
	"math"

	"github.com/richman/backend/internal/analysis"
)

const maxAdjustment = 0.10 // +/-10%

// baseWeights maps asset type to default weight configuration.
var baseWeights = map[string]analysis.WeightConfig{
	"gold_etf":         {Trend: 0.25, Position: 0.30, Catalyst: 0.45},
	"a_share_broad":    {Trend: 0.30, Position: 0.40, Catalyst: 0.30},
	"a_share_industry": {Trend: 0.35, Position: 0.30, Catalyst: 0.35},
	"us_stock":         {Trend: 0.30, Position: 0.35, Catalyst: 0.35},
}

// Manager handles weight configuration and adjustment for different asset types.
type Manager struct{}

// NewManager creates a new weight manager.
func NewManager() *Manager {
	return &Manager{}
}

// GetBaseWeights returns the default weight configuration for the given asset type.
func (m *Manager) GetBaseWeights(assetType string) (analysis.WeightConfig, error) {
	w, ok := baseWeights[assetType]
	if !ok {
		return analysis.WeightConfig{}, fmt.Errorf("no base weights defined for asset type: %s", assetType)
	}
	return w, nil
}

// Adjustment specifies a delta change for each dimension weight.
type Adjustment struct {
	Trend    float64 // delta, e.g. +0.05 or -0.05
	Position float64
	Catalyst float64
}

// AdjustWeights applies adjustments to base weights with clamping and normalization.
// Each dimension adjustment is clamped to +/-10% of the base weight.
// The result is normalized so all weights sum to 1.0.
func (m *Manager) AdjustWeights(base analysis.WeightConfig, adj Adjustment) analysis.WeightConfig {
	// Clamp each adjustment
	trendAdj := clamp(adj.Trend, -maxAdjustment, maxAdjustment)
	positionAdj := clamp(adj.Position, -maxAdjustment, maxAdjustment)
	catalystAdj := clamp(adj.Catalyst, -maxAdjustment, maxAdjustment)

	// Apply adjustments
	result := analysis.WeightConfig{
		Trend:    base.Trend + trendAdj,
		Position: base.Position + positionAdj,
		Catalyst: base.Catalyst + catalystAdj,
	}

	// Ensure no negative weights
	result.Trend = math.Max(result.Trend, 0.01)
	result.Position = math.Max(result.Position, 0.01)
	result.Catalyst = math.Max(result.Catalyst, 0.01)

	// Normalize to sum to 1.0
	total := result.Trend + result.Position + result.Catalyst
	result.Trend /= total
	result.Position /= total
	result.Catalyst /= total

	return result
}

// Risk preference bias deltas applied on top of the current weights. These
// values come from PRD §6 / TRD §5.4 and must stay within the ±10% allowed
// range for every asset type.
const riskBiasDelta = 0.05

// ApplyRiskBias layers the user's risk_preference bias on top of the provided
// weights. The bias is added to whatever adjustment the caller already made
// (for example LLM-driven adjustments) and is clamped to the asset type's
// allowed range (base ± 10%) before being normalized so the three dimensions
// still sum to 1.0.
//
// Rules:
//   - conservative -> position +5%, catalyst -5%
//   - neutral      -> no change (returns a normalized copy)
//   - aggressive   -> catalyst +5%, position -5%
//
// Unknown or empty preference values are treated as neutral. Unknown asset
// types silently fall back to normalizing the input without any allowed-range
// clamp, which keeps the function total for callers that operate on custom
// weight sets in tests.
func (m *Manager) ApplyRiskBias(
	current analysis.WeightConfig, assetType, pref string,
) analysis.WeightConfig {
	// Neutral / unknown preference: return the input verbatim after
	// normalizing to defend against minor floating-point drift in the caller.
	var trendDelta, posDelta, catDelta float64
	switch pref {
	case "conservative":
		posDelta = riskBiasDelta
		catDelta = -riskBiasDelta
	case "aggressive":
		catDelta = riskBiasDelta
		posDelta = -riskBiasDelta
	default:
		// neutral, empty, or unknown -> no bias.
	}

	result := analysis.WeightConfig{
		Trend:    current.Trend + trendDelta,
		Position: current.Position + posDelta,
		Catalyst: current.Catalyst + catDelta,
	}

	// Clamp each dimension to the allowed range defined by the asset type's
	// base weights ± maxAdjustment. If the asset type is unknown we skip the
	// clamp to avoid surprising callers that pass ad-hoc weight configs.
	if base, ok := baseWeights[assetType]; ok {
		result.Trend = clamp(result.Trend,
			base.Trend-maxAdjustment, base.Trend+maxAdjustment)
		result.Position = clamp(result.Position,
			base.Position-maxAdjustment, base.Position+maxAdjustment)
		result.Catalyst = clamp(result.Catalyst,
			base.Catalyst-maxAdjustment, base.Catalyst+maxAdjustment)
	}

	// Guard against negatives from pathological inputs before normalization.
	result.Trend = math.Max(result.Trend, 0.01)
	result.Position = math.Max(result.Position, 0.01)
	result.Catalyst = math.Max(result.Catalyst, 0.01)

	total := result.Trend + result.Position + result.Catalyst
	if total > 0 {
		result.Trend /= total
		result.Position /= total
		result.Catalyst /= total
	}
	return result
}

func clamp(value, lower, upper float64) float64 {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}
