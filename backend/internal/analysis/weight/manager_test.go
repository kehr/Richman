package weight

import (
	"math"
	"testing"

	"github.com/richman/backend/internal/analysis"
)

const epsilon = 1e-9

func TestGetBaseWeightsAllTypes(t *testing.T) {
	m := NewManager()

	tests := []struct {
		assetType string
		want      analysis.WeightConfig
	}{
		{"gold_etf", analysis.WeightConfig{Trend: 0.25, Position: 0.30, Catalyst: 0.45}},
		{"a_share_broad", analysis.WeightConfig{Trend: 0.30, Position: 0.40, Catalyst: 0.30}},
		{"a_share_industry", analysis.WeightConfig{Trend: 0.35, Position: 0.30, Catalyst: 0.35}},
		{"us_stock", analysis.WeightConfig{Trend: 0.30, Position: 0.35, Catalyst: 0.35}},
	}

	for _, tt := range tests {
		t.Run(tt.assetType, func(t *testing.T) {
			got, err := m.GetBaseWeights(tt.assetType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetBaseWeightsUnknownType(t *testing.T) {
	m := NewManager()
	_, err := m.GetBaseWeights("crypto")
	if err == nil {
		t.Error("expected error for unknown asset type")
	}
}

func TestBaseWeightsSumToOne(t *testing.T) {
	m := NewManager()
	for _, assetType := range []string{"gold_etf", "a_share_broad", "a_share_industry", "us_stock"} {
		w, err := m.GetBaseWeights(assetType)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", assetType, err)
		}
		sum := w.Trend + w.Position + w.Catalyst
		if math.Abs(sum-1.0) > epsilon {
			t.Errorf("%s weights sum to %f, want 1.0", assetType, sum)
		}
	}
}

func TestAdjustWeightsNormalization(t *testing.T) {
	m := NewManager()
	base := analysis.WeightConfig{Trend: 0.30, Position: 0.40, Catalyst: 0.30}

	result := m.AdjustWeights(base, Adjustment{Trend: 0.05, Position: -0.05, Catalyst: 0.00})

	sum := result.Trend + result.Position + result.Catalyst
	if math.Abs(sum-1.0) > epsilon {
		t.Errorf("adjusted weights sum to %f, want 1.0", sum)
	}
}

func TestAdjustWeightsClamping(t *testing.T) {
	m := NewManager()
	base := analysis.WeightConfig{Trend: 0.30, Position: 0.40, Catalyst: 0.30}

	// Try to adjust beyond the +/-10% limit
	result := m.AdjustWeights(base, Adjustment{Trend: 0.25, Position: -0.25, Catalyst: 0.0})

	// Trend should be clamped to base + 0.10, then normalized
	// Position should be clamped to base - 0.10, then normalized
	expectedTrend := 0.40 // 0.30 + 0.10
	expectedPos := 0.30   // 0.40 - 0.10
	expectedCat := 0.30   // 0.30 + 0.00
	total := expectedTrend + expectedPos + expectedCat

	if math.Abs(result.Trend-expectedTrend/total) > epsilon {
		t.Errorf("trend: got %f, want %f", result.Trend, expectedTrend/total)
	}
	if math.Abs(result.Position-expectedPos/total) > epsilon {
		t.Errorf("position: got %f, want %f", result.Position, expectedPos/total)
	}
}

func TestAdjustWeightsZeroAdjustment(t *testing.T) {
	m := NewManager()
	base := analysis.WeightConfig{Trend: 0.30, Position: 0.40, Catalyst: 0.30}

	result := m.AdjustWeights(base, Adjustment{})

	if math.Abs(result.Trend-base.Trend) > epsilon {
		t.Errorf("trend changed with zero adjustment: got %f, want %f", result.Trend, base.Trend)
	}
	if math.Abs(result.Position-base.Position) > epsilon {
		t.Errorf("position changed with zero adjustment: got %f, want %f", result.Position, base.Position)
	}
	if math.Abs(result.Catalyst-base.Catalyst) > epsilon {
		t.Errorf("catalyst changed with zero adjustment: got %f, want %f", result.Catalyst, base.Catalyst)
	}
}
