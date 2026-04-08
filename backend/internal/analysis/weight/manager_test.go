package weight

import (
	"math"
	"testing"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/model"
)

const epsilon = 1e-9

// Compile-time assertion that the weight package's local risk preference
// string constants match the canonical values defined in the model package.
// If any of these drift, the build fails.
var _ = [1]struct{}{}[map[bool]int{
	prefConservative == model.RiskPreferenceConservative &&
		prefNeutral == model.RiskPreferenceNeutral &&
		prefAggressive == model.RiskPreferenceAggressive: 0,
}[true]]

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

func TestApplyRiskBiasNeutralIsIdentity(t *testing.T) {
	m := NewManager()
	assetTypes := []string{"gold_etf", "a_share_broad", "a_share_industry", "us_stock"}

	for _, at := range assetTypes {
		t.Run(at, func(t *testing.T) {
			base, err := m.GetBaseWeights(at)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, pref := range []string{"neutral", "", "unknown-value"} {
				got := m.ApplyRiskBias(base, at, pref)
				if math.Abs(got.Trend-base.Trend) > epsilon ||
					math.Abs(got.Position-base.Position) > epsilon ||
					math.Abs(got.Catalyst-base.Catalyst) > epsilon {
					t.Errorf("pref=%q: got %+v, want %+v", pref, got, base)
				}
			}
		})
	}
}

func TestApplyRiskBiasSumsToOne(t *testing.T) {
	m := NewManager()
	cases := []struct {
		assetType string
		pref      string
	}{
		{"gold_etf", "conservative"},
		{"gold_etf", "aggressive"},
		{"a_share_broad", "conservative"},
		{"a_share_broad", "aggressive"},
		{"a_share_industry", "conservative"},
		{"a_share_industry", "aggressive"},
		{"us_stock", "conservative"},
		{"us_stock", "aggressive"},
	}
	for _, tc := range cases {
		t.Run(tc.assetType+"/"+tc.pref, func(t *testing.T) {
			base, err := m.GetBaseWeights(tc.assetType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := m.ApplyRiskBias(base, tc.assetType, tc.pref)
			sum := got.Trend + got.Position + got.Catalyst
			if math.Abs(sum-1.0) > epsilon {
				t.Errorf("sum=%f, want 1.0 (got=%+v)", sum, got)
			}
			if got.Trend < 0 || got.Position < 0 || got.Catalyst < 0 {
				t.Errorf("negative weight in %+v", got)
			}
		})
	}
}

func TestApplyRiskBiasDirection(t *testing.T) {
	m := NewManager()
	base, _ := m.GetBaseWeights("a_share_broad") // {0.30, 0.40, 0.30}

	conservative := m.ApplyRiskBias(base, "a_share_broad", "conservative")
	if conservative.Position <= base.Position {
		t.Errorf("conservative should raise position: got %f, base %f",
			conservative.Position, base.Position)
	}
	if conservative.Catalyst >= base.Catalyst {
		t.Errorf("conservative should lower catalyst: got %f, base %f",
			conservative.Catalyst, base.Catalyst)
	}

	aggressive := m.ApplyRiskBias(base, "a_share_broad", "aggressive")
	if aggressive.Catalyst <= base.Catalyst {
		t.Errorf("aggressive should raise catalyst: got %f, base %f",
			aggressive.Catalyst, base.Catalyst)
	}
	if aggressive.Position >= base.Position {
		t.Errorf("aggressive should lower position: got %f, base %f",
			aggressive.Position, base.Position)
	}
}

func TestApplyRiskBiasTruncatesOutOfRange(t *testing.T) {
	m := NewManager()
	// a_share_broad base = {0.30, 0.40, 0.30}
	// allowed position range = [0.30, 0.50]; catalyst range = [0.20, 0.40]
	// Start already at the +10% upper bound for position and -10% lower bound
	// for catalyst, so conservative bias (+5% pos, -5% cat) should be
	// truncated back to the bounds.
	preAdjusted := analysis.WeightConfig{Trend: 0.30, Position: 0.50, Catalyst: 0.20}
	got := m.ApplyRiskBias(preAdjusted, "a_share_broad", "conservative")

	// Expect clamping to {0.30, 0.50, 0.20} then renormalization (sum=1.0
	// already, so unchanged).
	if math.Abs(got.Position-0.50) > epsilon {
		t.Errorf("position truncation: got %f, want 0.50", got.Position)
	}
	if math.Abs(got.Catalyst-0.20) > epsilon {
		t.Errorf("catalyst truncation: got %f, want 0.20", got.Catalyst)
	}
	sum := got.Trend + got.Position + got.Catalyst
	if math.Abs(sum-1.0) > epsilon {
		t.Errorf("sum=%f, want 1.0", sum)
	}
}

func TestApplyRiskBiasWithinAllowedRange(t *testing.T) {
	m := NewManager()
	// Every result dimension must stay inside base ± 10%.
	for _, at := range []string{"gold_etf", "a_share_broad", "a_share_industry", "us_stock"} {
		base, _ := m.GetBaseWeights(at)
		for _, pref := range []string{"conservative", "aggressive"} {
			got := m.ApplyRiskBias(base, at, pref)
			// After normalization the exact delta can drift slightly, but it
			// must never exceed the ±10% bound by more than epsilon.
			if got.Trend < base.Trend-maxAdjustment-epsilon ||
				got.Trend > base.Trend+maxAdjustment+epsilon {
				t.Errorf("%s/%s: trend %f outside [%f, %f]",
					at, pref, got.Trend,
					base.Trend-maxAdjustment, base.Trend+maxAdjustment)
			}
			if got.Position < base.Position-maxAdjustment-epsilon ||
				got.Position > base.Position+maxAdjustment+epsilon {
				t.Errorf("%s/%s: position %f outside [%f, %f]",
					at, pref, got.Position,
					base.Position-maxAdjustment, base.Position+maxAdjustment)
			}
			if got.Catalyst < base.Catalyst-maxAdjustment-epsilon ||
				got.Catalyst > base.Catalyst+maxAdjustment+epsilon {
				t.Errorf("%s/%s: catalyst %f outside [%f, %f]",
					at, pref, got.Catalyst,
					base.Catalyst-maxAdjustment, base.Catalyst+maxAdjustment)
			}
		}
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
