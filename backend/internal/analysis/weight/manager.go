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

func clamp(value, lower, upper float64) float64 {
	if value < lower {
		return lower
	}
	if value > upper {
		return upper
	}
	return value
}
