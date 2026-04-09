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

// stubResolver is an in-memory llm.Resolver used to exercise the success /
// failure / malformed-response code paths without touching the network. The
// fields are read once per call so each test can install its own canned
// response or error.
type stubResolver struct {
	resp *llm.ResolvedResponse
	err  error
}

func (s *stubResolver) ResolvedChatCompletion(
	_ context.Context, _ int64, _ llm.ChatRequest,
) (*llm.ResolvedResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

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

// resolvedUser wraps content in a ResolvedResponse that reports the user
// layer as the serving provider. Used by the happy-path tests.
func resolvedUser(content string) *llm.ResolvedResponse {
	return &llm.ResolvedResponse{
		Response: &llm.ChatResponse{Content: content, Latency: time.Millisecond},
		Layer:    llm.LayerUser,
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
                        "rationale": {
                            "triggerReason": "buy the dip",
                            "positionReason": "target position not reached",
                            "precondition": "price at support",
                            "fallback": "wait for lower entry",
                            "timeWindow": "1-3 days"
                        }
                    }
                ],
                "validDays": 7
            }
        }
    }`
	resolver := &stubResolver{resp: resolvedUser(body)}
	s := NewSynthesizer(resolver, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendSmallAdd), 42)
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
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta.Source != "llm" {
		t.Errorf("expected meta.Source=llm, got %q", meta.Source)
	}
	if meta.ProviderUsed != string(llm.LayerUser) {
		t.Errorf("expected meta.ProviderUsed=user, got %q", meta.ProviderUsed)
	}
}

func TestSynthesize_NilResolver_UsesTemplateFallback(t *testing.T) {
	// When the Resolver is unavailable at startup (no master key, no system
	// default, etc.) main.go constructs a Synthesizer with a nil Resolver
	// and relies on Synthesize short-circuiting to the template fallback.
	// Regression guard for the nil-pointer panic observed on dev start when
	// this contract was silently broken.
	s := NewSynthesizer(nil, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendHold), 1)
	if err != nil {
		t.Fatalf("expected nil error on nil-resolver fallback, got %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output from template fallback")
	}
	if out.Recommendation.Action != recommendation.ActionHold {
		t.Errorf("expected hold fallback, got %q", out.Recommendation.Action)
	}
	// Template fallback intentionally leaves text summary fields empty;
	// the frontend renders i18n placeholders when summaries are absent.
	if out.TrendSummary != "" || out.PositionSummary != "" {
		t.Error("expected template fallback to leave summary fields empty")
	}
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta.Source != "template" {
		t.Errorf("expected meta.Source=template, got %q", meta.Source)
	}
	if meta.ProviderUsed != string(llm.LayerNone) {
		t.Errorf("expected meta.ProviderUsed=none, got %q", meta.ProviderUsed)
	}
}

func TestSynthesize_LLMFailure_UsesTemplateFallback(t *testing.T) {
	resolver := &stubResolver{err: errors.New("network down")}
	s := NewSynthesizer(resolver, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendAggressiveAdd), 42)
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
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}
	if meta.Source != "template" {
		t.Errorf("expected meta.Source=template, got %q", meta.Source)
	}
	if meta.ProviderUsed != string(llm.LayerNone) {
		t.Errorf("expected meta.ProviderUsed=none, got %q", meta.ProviderUsed)
	}
}

func TestSynthesize_AllLayersFailed_UsesTemplateFallback(t *testing.T) {
	// Resolver surfaces ErrAllLayersFailed when every fallback layer is
	// unusable. The Synthesizer must treat this like any other failure and
	// emit a template card, not bubble the error.
	resolver := &stubResolver{err: llm.ErrAllLayersFailed}
	s := NewSynthesizer(resolver, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendHold), 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if out == nil {
		t.Fatal("expected non-nil output")
	}
	if meta.Source != "template" || meta.ProviderUsed != string(llm.LayerNone) {
		t.Errorf("expected template/none meta, got source=%q provider=%q",
			meta.Source, meta.ProviderUsed)
	}
}

func TestSynthesize_LLMMalformedJSON_UsesTemplateFallback(t *testing.T) {
	// Layer is preserved on the meta because the LLM was reachable: the
	// response just could not be parsed. Operators need to see "user layer
	// answered with garbage", not "no layer answered".
	resolver := &stubResolver{resp: resolvedUser("not json at all")}
	s := NewSynthesizer(resolver, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendHold), 42)
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
	if meta.Source != "template" {
		t.Errorf("expected meta.Source=template, got %q", meta.Source)
	}
	if meta.ProviderUsed != string(llm.LayerUser) {
		t.Errorf("expected meta.ProviderUsed=user (layer preserved), got %q", meta.ProviderUsed)
	}
}

func TestSynthesize_LLMMissingRecommendation_UsesRecommendationFallback(t *testing.T) {
	// Valid text fields but no recommendation sub-object — the output meta
	// should reflect the "mixed" ternary value.
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
	resolver := &stubResolver{resp: resolvedUser(body)}
	s := NewSynthesizer(resolver, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendGradualReduce), 42)
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
	if meta.Source != "mixed" {
		t.Errorf("expected meta.Source=mixed, got %q", meta.Source)
	}
	if meta.ProviderUsed != string(llm.LayerUser) {
		t.Errorf("expected meta.ProviderUsed=user, got %q", meta.ProviderUsed)
	}
}

func TestSynthesize_SystemDefaultFallback_RecordsLayer(t *testing.T) {
	// When the Resolver walks past the user layer and the system default
	// answers instead, the meta must report the system_default layer so the
	// dashboard banner can surface "using shared provider".
	body := `{
        "trendSummary": "ok",
        "positionSummary": "ok",
        "catalystSummary": "ok",
        "actionAdvice": "ok",
        "detailedAdvice": "ok",
        "riskWarnings": [],
        "todayHighlights": "",
        "weightAdjustment": "",
        "recommendation": {
            "action": "hold",
            "label": "Hold",
            "currentPositionPct": 20.0,
            "targetPositionPct": 20.0,
            "execution": {
                "type": "monitor",
                "steps": [],
                "stopLoss": 140.0,
                "takeProfit": 160.0,
                "validDays": 7
            }
        }
    }`
	resolver := &stubResolver{
		resp: &llm.ResolvedResponse{
			Response: &llm.ChatResponse{Content: body, Latency: time.Millisecond},
			Layer:    llm.LayerSystemDefault,
		},
	}
	s := NewSynthesizer(resolver, zap.NewNop())

	out, meta, err := s.Synthesize(context.Background(), sampleInput(analysis.RecommendHold), 42)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if meta.Source != "llm" {
		t.Errorf("expected meta.Source=llm, got %q", meta.Source)
	}
	if meta.ProviderUsed != string(llm.LayerSystemDefault) {
		t.Errorf("expected meta.ProviderUsed=system_default, got %q", meta.ProviderUsed)
	}
	// ensureRecommendation injects fallback steps when LLM returns monitor
	// with empty steps.
	if len(out.Recommendation.Execution.Steps) == 0 {
		t.Error("expected fallback monitor steps to be injected")
	}
}

func TestFallbackRecommendation_AllActions(t *testing.T) {
	cases := []struct {
		rec       analysis.Recommendation
		wantLevel int
		wantType  recommendation.ExecutionType
		wantSteps bool
	}{
		{analysis.RecommendAggressiveAdd, 2, recommendation.ExecutionOneShot, true},
		{analysis.RecommendSmallAdd, 1, recommendation.ExecutionOneShot, true},
		{analysis.RecommendHold, 0, recommendation.ExecutionMonitor, true},
		{analysis.RecommendGradualReduce, -1, recommendation.ExecutionOneShot, true},
		{analysis.RecommendControlPosition, -2, recommendation.ExecutionOneShot, true},
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
			if c.wantSteps && len(got.Execution.Steps) == 0 {
				t.Error("expected at least one step")
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
