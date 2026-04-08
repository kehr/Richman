package synthesis

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/recommendation"
	"github.com/richman/backend/internal/llm"
)

// stubProvider is an in-memory llm.Provider used to exercise the success /
// failure / malformed-response code paths without touching the network.
type stubProvider struct {
	resp *llm.ChatResponse
	err  error
}

func (s *stubProvider) ChatCompletion(_ context.Context, _ llm.ChatRequest) (*llm.ChatResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func (s *stubProvider) Name() string { return "stub" }

func sampleInput(rec analysis.Recommendation) *SynthesisInput {
	return &SynthesisInput{
		AssetCode: "AAPL",
		AssetType: "us_stock",
		AssetName: "Apple Inc",
		Trend: analysis.TrendResult{
			Direction: analysis.DirectionUpward,
			Strength:  0.7,
			Summary:   "uptrend",
		},
		Position: analysis.PositionResult{
			Assessment: analysis.DirectionBullish,
			Percentile: 0.3,
			Summary:    "undervalued",
		},
		Catalyst: analysis.CatalystResult{
			Direction: analysis.DirectionBullish,
			Score:     0.5,
			Summary:   "earnings beat",
		},
		Weights:        analysis.WeightConfig{Trend: 0.4, Position: 0.3, Catalyst: 0.3},
		Confidence:     72,
		Recommendation: rec,
		CostPrice:      150.0,
		PositionRatio:  0.2,
	}
}

func TestSynthesize_LLMSuccessWithRecommendation(t *testing.T) {
	body := `{
        "trendSummary": "Uptrend intact.",
        "positionSummary": "Discount to fair value.",
        "catalystSummary": "Earnings beat consensus.",
        "actionAdvice": "Add on pullback.",
        "detailedAdvice": "Stage buys near $145.",
        "riskWarnings": ["market volatility"],
        "todayHighlights": "earnings release",
        "weightAdjustment": "",
        "recommendation": {
            "action": "small_add",
            "label": "Small add",
            "currentPositionPct": 20.0,
            "targetPositionPct": 25.0,
            "execution": {
                "type": "one-shot",
                "steps": [
                    {
                        "order": 1,
                        "triggerType": "price",
                        "triggerValue": "price <= 145",
                        "deltaPct": 5.0,
                        "rationale": "buy the dip"
                    }
                ],
                "validDays": 7
            }
        }
    }`
	provider := &stubProvider{resp: &llm.ChatResponse{Content: body, Latency: time.Millisecond}}
	s := NewSynthesizer(provider, zap.NewNop())

	out, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendSmallAdd))
	if err != nil {
		t.Fatalf("Synthesize returned error: %v", err)
	}
	if out.TrendSummary != "Uptrend intact." {
		t.Errorf("unexpected TrendSummary: %q", out.TrendSummary)
	}
	if out.Recommendation.Action != recommendation.ActionSmallAdd {
		t.Errorf("expected action=small_add, got %q", out.Recommendation.Action)
	}
	if out.Recommendation.ActionLevel != 1 {
		t.Errorf("expected level=1, got %d", out.Recommendation.ActionLevel)
	}
	if got := out.Recommendation.TargetPositionPct; got != 25 {
		t.Errorf("expected target=25, got %v", got)
	}
	if len(out.Recommendation.Execution.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(out.Recommendation.Execution.Steps))
	}
}

func TestSynthesize_NilProvider_UsesTemplateFallback(t *testing.T) {
	// When the LLM provider is unavailable at startup (no API key, dial
	// failure, etc.) main.go constructs a Synthesizer with a nil provider
	// and relies on Synthesize short-circuiting to the template fallback.
	// Regression guard for the nil-pointer panic observed on dev start when
	// this contract was silently broken.
	s := NewSynthesizer(nil, zap.NewNop())

	out, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendHold))
	if err != nil {
		t.Fatalf("expected nil error on nil-provider fallback, got %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output from template fallback")
	}
	if out.Recommendation.Action != recommendation.ActionHold {
		t.Errorf("expected hold fallback, got %q", out.Recommendation.Action)
	}
	if out.TrendSummary == "" || out.PositionSummary == "" {
		t.Error("expected template fallback to populate summary fields")
	}
}

func TestSynthesize_LLMFailure_UsesTemplateFallback(t *testing.T) {
	provider := &stubProvider{err: errors.New("network down")}
	s := NewSynthesizer(provider, zap.NewNop())

	out, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendAggressiveAdd))
	if err != nil {
		t.Fatalf("expected nil error on fallback, got %v", err)
	}
	if out.Recommendation.Action != recommendation.ActionAggressiveAdd {
		t.Errorf("expected fallback action=aggressive_add, got %q", out.Recommendation.Action)
	}
	if out.Recommendation.ActionLevel != 2 {
		t.Errorf("expected level=2, got %d", out.Recommendation.ActionLevel)
	}
	// Fallback for aggressive_add should bump target by 10 percentage points.
	if got := out.Recommendation.TargetPositionPct; got != 30.0 {
		t.Errorf("expected target=30, got %v", got)
	}
	if len(out.Recommendation.Execution.Steps) != 1 {
		t.Fatalf("expected 1 fallback step")
	}
}

func TestSynthesize_LLMMalformedJSON_UsesTemplateFallback(t *testing.T) {
	provider := &stubProvider{resp: &llm.ChatResponse{Content: "not json at all"}}
	s := NewSynthesizer(provider, zap.NewNop())

	out, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendHold))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out.Recommendation.Action != recommendation.ActionHold {
		t.Errorf("expected hold fallback, got %q", out.Recommendation.Action)
	}
	if out.Recommendation.Execution.Type != recommendation.ExecutionMonitor {
		t.Errorf("expected monitor execution, got %q", out.Recommendation.Execution.Type)
	}
	if out.Recommendation.Execution.StopLoss == nil || out.Recommendation.Execution.TakeProfit == nil {
		t.Errorf("expected stop/take guards on hold fallback")
	}
}

func TestSynthesize_LLMMissingRecommendation_UsesRecommendationFallback(t *testing.T) {
	// Valid text fields but no recommendation sub-object.
	body := `{
        "trendSummary": "ok",
        "positionSummary": "ok",
        "catalystSummary": "ok",
        "actionAdvice": "ok",
        "detailedAdvice": "ok",
        "riskWarnings": [],
        "todayHighlights": "",
        "weightAdjustment": ""
    }`
	provider := &stubProvider{resp: &llm.ChatResponse{Content: body}}
	s := NewSynthesizer(provider, zap.NewNop())

	out, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendGradualReduce))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.TrendSummary != "ok" {
		t.Errorf("text field should remain LLM-sourced")
	}
	if out.Recommendation.Action != recommendation.ActionGradualReduce {
		t.Errorf("expected gradual_reduce fallback, got %q", out.Recommendation.Action)
	}
	if out.Recommendation.ActionLevel != -1 {
		t.Errorf("expected level=-1, got %d", out.Recommendation.ActionLevel)
	}
}

func TestFallbackRecommendation_AllActions(t *testing.T) {
	cases := []struct {
		rec       analysis.Recommendation
		wantLevel int
		wantType  recommendation.ExecutionType
	}{
		{analysis.RecommendAggressiveAdd, 2, recommendation.ExecutionOneShot},
		{analysis.RecommendSmallAdd, 1, recommendation.ExecutionOneShot},
		{analysis.RecommendHold, 0, recommendation.ExecutionMonitor},
		{analysis.RecommendGradualReduce, -1, recommendation.ExecutionOneShot},
		{analysis.RecommendControlPosition, -2, recommendation.ExecutionOneShot},
	}
	for _, c := range cases {
		t.Run(string(c.rec), func(t *testing.T) {
			got := fallbackRecommendation(sampleInput(c.rec))
			if got.ActionLevel != c.wantLevel {
				t.Errorf("level: want %d got %d", c.wantLevel, got.ActionLevel)
			}
			if got.Execution.Type != c.wantType {
				t.Errorf("type: want %q got %q", c.wantType, got.Execution.Type)
			}
			if got.Execution.ValidDays != recommendation.ValidityDefaultDays {
				t.Errorf("validDays: want %d got %d",
					recommendation.ValidityDefaultDays, got.Execution.ValidDays)
			}
		})
	}
}

func TestLegacyToAction_IsDirectCast(t *testing.T) {
	if got := legacyToAction(analysis.RecommendSmallAdd); got != recommendation.ActionSmallAdd {
		t.Errorf("mismatch: %q", got)
	}
	if got := legacyToAction(analysis.RecommendControlPosition); got != recommendation.ActionControl {
		t.Errorf("mismatch: %q", got)
	}
}
