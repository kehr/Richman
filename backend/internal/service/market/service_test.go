package market

import (
	"encoding/json"
	"testing"

	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

func newTestService() *Service {
	return &Service{
		logger: zap.NewNop(),
	}
}

func float64Ptr(v float64) *float64 { return &v }
func intPtr(v int) *int             { return &v }
func stringPtr(v string) *string    { return &v }

func TestDeriveDimensionSignal_Buckets(t *testing.T) {
	s := newTestService()
	cases := []struct {
		name  string
		score *float64
		want  string
	}{
		{"nil", nil, "neutral"},
		{"strong-bullish-edge", float64Ptr(75), "bullish"},
		{"bullish-low-edge", float64Ptr(60), "bullish"},
		{"neutral-mid", float64Ptr(50), "neutral"},
		{"neutral-high-edge", float64Ptr(59.99), "neutral"},
		{"neutral-low-edge", float64Ptr(40), "neutral"},
		{"bearish-edge", float64Ptr(39.99), "bearish"},
		{"bearish-deep", float64Ptr(10), "bearish"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := s.deriveDimensionSignal(c.score); got != c.want {
				t.Fatalf("score=%v want=%s got=%s", c.score, c.want, got)
			}
		})
	}
}

func TestBuildExecutionPlan_Complete(t *testing.T) {
	s := newTestService()
	raw := json.RawMessage(`{
		"action": "buy",
		"action_label": "Increase position",
		"default_action": "Hold and observe",
		"stop_loss": 100.5,
		"take_profit": 150.25,
		"valid_days": 7,
		"concentration_message": "Position exceeds 30% of portfolio",
		"scenarios": [
			{"condition": "Price > 120", "action": "Trim 1/3", "rationale": "Take profit", "priority": 1},
			{"condition": "Price < 100", "action": "Stop out", "rationale": "Capital preservation", "priority": 2}
		],
		"is_demo_plan": true,
		"current_position": 0
	}`)

	plan := s.buildExecutionPlan(raw)
	if plan == nil {
		t.Fatal("expected plan, got nil")
	}
	if plan.Recommendation != "Increase position" {
		t.Errorf("recommendation: want %q got %q", "Increase position", plan.Recommendation)
	}
	if plan.DefaultAdvice != "Hold and observe" {
		t.Errorf("defaultAdvice: want %q got %q", "Hold and observe", plan.DefaultAdvice)
	}
	if plan.StopLoss == nil || *plan.StopLoss != 100.5 {
		t.Errorf("stopLoss: want 100.5 got %v", plan.StopLoss)
	}
	if plan.ValidDays != 7 {
		t.Errorf("validDays: want 7 got %d", plan.ValidDays)
	}
	if plan.ConcentrationWarning == nil || *plan.ConcentrationWarning != "Position exceeds 30% of portfolio" {
		t.Errorf("concentrationWarning unexpected: %v", plan.ConcentrationWarning)
	}
	if len(plan.Scenarios) != 2 {
		t.Fatalf("scenarios len: want 2 got %d", len(plan.Scenarios))
	}
	if plan.Scenarios[0].ID != "scenario-1" || plan.Scenarios[1].ID != "scenario-2" {
		t.Errorf("scenario ids: %v / %v", plan.Scenarios[0].ID, plan.Scenarios[1].ID)
	}
	if plan.Disclaimer == "" {
		t.Error("disclaimer should be populated")
	}
}

func TestBuildExecutionPlan_NilOrInvalid(t *testing.T) {
	s := newTestService()
	cases := []struct {
		name string
		raw  json.RawMessage
	}{
		{"nil", nil},
		{"empty", json.RawMessage("")},
		{"null", json.RawMessage("null")},
		{"invalid-json", json.RawMessage("{not valid")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := s.buildExecutionPlan(c.raw); got != nil {
				t.Fatalf("want nil got %+v", got)
			}
		})
	}
}

func TestBuildRiskFactors(t *testing.T) {
	s := newTestService()

	if got := s.buildRiskFactors(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}
	if got := s.buildRiskFactors(json.RawMessage("[]")); got != nil {
		t.Errorf("empty array should return nil, got %v", got)
	}

	out := s.buildRiskFactors(json.RawMessage(`["Liquidity drying up", "", "Macro headwinds"]`))
	if len(out) != 2 {
		t.Fatalf("want 2 factors (empty filtered), got %d", len(out))
	}
	if out[0].ID != "rf-1" || out[1].ID != "rf-3" {
		t.Errorf("ids should preserve original index: got %s / %s", out[0].ID, out[1].ID)
	}
	for _, f := range out {
		if f.Severity != "medium" {
			t.Errorf("severity should default to medium, got %q", f.Severity)
		}
	}
}

func TestBuildKeyPriceLevels_DistanceAndOrder(t *testing.T) {
	s := newTestService()
	current := float64Ptr(100)
	supports := []float64{80, 95, 60}
	resistances := []float64{105, 130, 110}

	levels := s.buildKeyPriceLevels(supports, resistances, current, "USD")
	if len(levels) != 6 {
		t.Fatalf("want 6 levels, got %d", len(levels))
	}

	// First three are supports sorted by absolute distance: 95 (-5), 80 (-20), 60 (-40)
	if levels[0].Price != 95 || levels[1].Price != 80 || levels[2].Price != 60 {
		t.Errorf("supports order wrong: %v %v %v", levels[0].Price, levels[1].Price, levels[2].Price)
	}
	// Then resistances sorted by distance: 105 (+5), 110 (+10), 130 (+30)
	if levels[3].Price != 105 || levels[4].Price != 110 || levels[5].Price != 130 {
		t.Errorf("resistances order wrong: %v %v %v", levels[3].Price, levels[4].Price, levels[5].Price)
	}
	if levels[0].Type != "support" || levels[3].Type != "resistance" {
		t.Errorf("types wrong: %s %s", levels[0].Type, levels[3].Type)
	}
	if levels[0].Currency != "USD" {
		t.Errorf("currency not propagated: %q", levels[0].Currency)
	}
	// distancePct sanity
	if levels[0].DistancePct != -5 {
		t.Errorf("distance for 95 vs 100: want -5 got %v", levels[0].DistancePct)
	}
	if levels[3].DistancePct != 5 {
		t.Errorf("distance for 105 vs 100: want 5 got %v", levels[3].DistancePct)
	}
}

func TestBuildKeyPriceLevels_NoCurrentPrice(t *testing.T) {
	s := newTestService()
	levels := s.buildKeyPriceLevels([]float64{80}, []float64{120}, nil, "USD")
	if len(levels) != 2 {
		t.Fatalf("want 2 got %d", len(levels))
	}
	for _, l := range levels {
		if l.DistancePct != 0 {
			t.Errorf("distancePct should be 0 when current is nil, got %v", l.DistancePct)
		}
	}
}

func TestBuildKeyPriceLevels_Empty(t *testing.T) {
	s := newTestService()
	if got := s.buildKeyPriceLevels(nil, nil, float64Ptr(100), "USD"); got != nil {
		t.Errorf("want nil got %v", got)
	}
}

func TestBuildDimensions_Aggregation(t *testing.T) {
	s := newTestService()
	analysis := &model.AssetAnalysis{
		D1Score: float64Ptr(70), D1BaseScore: float64Ptr(65), D1LLMAdjustment: float64Ptr(5), D1Weight: 0.3,
		D2Score: float64Ptr(50), D2BaseScore: float64Ptr(50), D2LLMAdjustment: nil, D2Weight: 0.25,
		D3Score: float64Ptr(35), D3BaseScore: float64Ptr(40), D3LLMAdjustment: float64Ptr(-5), D3Weight: 0.2,
		D4Score: float64Ptr(80), D4BaseScore: float64Ptr(80), D4Weight: 0.25,
	}
	dims := []model.AnalysisDimension{
		{Dimension: "d1", SubIndicator: "vix", RawValue: float64Ptr(15.2), BlendedPercentile: float64Ptr(72), NormalizedScore: float64Ptr(60), WeightInDimension: float64Ptr(0.5)},
		{Dimension: "d1", SubIndicator: "credit", RawValue: float64Ptr(0.45), Percentile1Y: float64Ptr(80), NormalizedScore: float64Ptr(65), WeightInDimension: float64Ptr(0.5)},
		{Dimension: "d4", SubIndicator: "rsi", RawValue: float64Ptr(55), BlendedPercentile: float64Ptr(50), NormalizedScore: float64Ptr(80), WeightInDimension: float64Ptr(1)},
	}

	out := s.buildDimensions(analysis, dims)
	if len(out) != 4 {
		t.Fatalf("want 4 dimensions, got %d", len(out))
	}

	// d1 should have 2 sub-indicators
	if len(out[0].SubIndicators) != 2 {
		t.Errorf("d1 sub count: want 2 got %d", len(out[0].SubIndicators))
	}
	// d2 has none
	if len(out[1].SubIndicators) != 0 {
		t.Errorf("d2 sub should be empty, got %d", len(out[1].SubIndicators))
	}
	// d4 has 1 and no LLMAdjustment field
	if len(out[3].SubIndicators) != 1 {
		t.Errorf("d4 sub count: want 1 got %d", len(out[3].SubIndicators))
	}
	if out[3].LLMAdjustment != nil {
		t.Errorf("d4 LLMAdjustment must be nil, got %v", out[3].LLMAdjustment)
	}

	// Signal bucketing
	if out[0].Signal != "bullish" || out[1].Signal != "neutral" || out[2].Signal != "bearish" || out[3].Signal != "bullish" {
		t.Errorf("signals: %s %s %s %s", out[0].Signal, out[1].Signal, out[2].Signal, out[3].Signal)
	}

	// Names + IDs
	wantIDs := []string{"d1", "d2", "d3", "d4"}
	wantNames := []string{"Macro", "Liquidity", "Sentiment", "Technical"}
	for i := range out {
		if out[i].ID != wantIDs[i] {
			t.Errorf("id[%d]: want %s got %s", i, wantIDs[i], out[i].ID)
		}
		if out[i].Name != wantNames[i] {
			t.Errorf("name[%d]: want %s got %s", i, wantNames[i], out[i].Name)
		}
	}

	// Percentile preference: blended over 1y
	d1Subs := out[0].SubIndicators
	// Find vix entry
	var vix *DimensionSubIndicatorDTO
	for i := range d1Subs {
		if d1Subs[i].Name == "vix" {
			vix = &d1Subs[i]
		}
	}
	if vix == nil || vix.Percentile == nil || *vix.Percentile != 72 {
		t.Errorf("vix percentile: want 72 got %v", vix)
	}
}

func TestBuildDimensions_NilAnalysis(t *testing.T) {
	s := newTestService()
	out := s.buildDimensions(nil, nil)
	if len(out) != 0 {
		t.Errorf("want empty, got %d", len(out))
	}
}

func TestBuildDrawdownReference(t *testing.T) {
	if got := buildDrawdownReference(nil); got != nil {
		t.Errorf("nil input should return nil")
	}
	// Missing required fields
	got := buildDrawdownReference(&rawDrawdownReference{
		MaxDrawdown: float64Ptr(0.32),
		// HistoricalAvgDrawdown intentionally missing
	})
	if got != nil {
		t.Errorf("missing avg should return nil, got %+v", got)
	}

	full := buildDrawdownReference(&rawDrawdownReference{
		CurrentBullRunStart:   stringPtr("2023-10-01"),
		MaxDrawdown:           float64Ptr(0.32),
		MaxDrawdownDate:       stringPtr("2024-08-05"),
		HistoricalAvgDrawdown: float64Ptr(0.25),
	})
	if full == nil {
		t.Fatal("expected dto, got nil")
	}
	if full.CurrentBullMaxDrawdown != 0.32 || full.CurrentBullMaxDrawdownDate != "2024-08-05" || full.HistoricalAvgDrawdown != 0.25 {
		t.Errorf("dto wrong: %+v", full)
	}
}

func TestInferCurrency(t *testing.T) {
	cases := []struct {
		name      string
		snap      *ohlcvSnapshot
		assetType string
		want      string
	}{
		{"ohlcv-cny", &ohlcvSnapshot{Currency: "CNY"}, "stock-cn", "CNY"},
		{"ohlcv-usd-overrides-type", &ohlcvSnapshot{Currency: "USD"}, "stock-cn", "USD"},
		{"no-ohlcv-stock-cn", nil, "stock-cn", "CNY"},
		{"no-ohlcv-other", nil, "etf", "USD"},
		{"ohlcv-empty-currency-stock-cn", &ohlcvSnapshot{Currency: ""}, "stock-cn", "CNY"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := inferCurrency(c.snap, c.assetType); got != c.want {
				t.Fatalf("want %s got %s", c.want, got)
			}
		})
	}
}

func TestUnmarshalAnalysisMetadata(t *testing.T) {
	if got := unmarshalAnalysisMetadata(nil); got != nil {
		t.Error("nil input should return nil")
	}
	if got := unmarshalAnalysisMetadata(json.RawMessage("null")); got != nil {
		t.Error("null input should return nil")
	}
	got := unmarshalAnalysisMetadata(json.RawMessage(`{
		"support_levels": [80, 90],
		"resistance_levels": [110, 120],
		"drawdown_reference": {
			"currentBullRunStart": "2023-01-01",
			"maxDrawdown": 0.18,
			"maxDrawdownDate": "2024-04-19",
			"historicalAvgDrawdown": 0.22
		}
	}`))
	if got == nil {
		t.Fatal("expected metadata, got nil")
	}
	if len(got.SupportLevels) != 2 || got.SupportLevels[0] != 80 {
		t.Errorf("supports decode wrong: %v", got.SupportLevels)
	}
	if got.DrawdownReference == nil || got.DrawdownReference.MaxDrawdown == nil || *got.DrawdownReference.MaxDrawdown != 0.18 {
		t.Errorf("drawdown decode wrong: %+v", got.DrawdownReference)
	}
}

// silence unused-helper warnings if test set later trims callers.
var _ = intPtr
