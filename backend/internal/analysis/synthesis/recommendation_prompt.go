package synthesis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/prompts"
	"github.com/richman/backend/internal/analysis/recommendation"
)

// buildRecommendationSection renders the recommendation sub-prompt template
// with constraints derived from the synthesis input. The action and current
// position percentage are locked to the matrix output so the LLM cannot
// override the fundamental recommendation direction.
func buildRecommendationSection(input *SynthesisInput) (string, error) {
	currentPct := input.PositionRatio * 100
	action := string(input.Recommendation)

	deltaConstraint, execTypeHint := recommendationConstraints(input.Recommendation)
	stopGuidance := stopLossGuidance(input.CostPrice)

	return prompts.SynthesisRecommendation(prompts.SynthesisRecommendationData{
		Action:          action,
		CurrentPct:      currentPct,
		ExecTypeHint:    execTypeHint,
		DeltaConstraint: deltaConstraint,
		StopGuidance:    stopGuidance,
	})
}

// recommendationConstraints returns the delta-sizing constraint text and the
// recommended execution type for a given action. These strings are injected
// into the recommendation template as hard requirements for the LLM.
func recommendationConstraints(rec analysis.Recommendation) (deltaConstraint, execTypeHint string) {
	switch rec {
	case analysis.RecommendAggressiveAdd:
		return "Total delta across all steps: 8-15%. Single-step max: 10%.", "one-shot"
	case analysis.RecommendSmallAdd:
		return "Total delta across all steps: 3-8%. Single-step max: 5%.", "one-shot"
	case analysis.RecommendHold:
		return "Use type=monitor. Steps represent WATCH CONDITIONS, not trades. " +
			"DeltaPct: 0 (pure watch) or -5 (reduce if triggered).", "monitor"
	case analysis.RecommendGradualReduce:
		return "Total delta: -8 to -15% (negative). Use staged (2 steps) to reduce market impact.", "staged"
	case analysis.RecommendControlPosition:
		return "Total delta: -15 to -20% (negative). Can be one-shot if signal urgency is high.", "one-shot"
	default:
		return "Keep delta proportional to signal confidence. Respect risk limits.", "one-shot"
	}
}

// stopLossGuidance produces the stop-loss / take-profit guidance string for
// the recommendation template. When cost price is available, concrete default
// levels (7% stop, 12% take-profit) are embedded so the LLM has a grounded
// starting point rather than fabricating arbitrary levels.
func stopLossGuidance(costPrice float64) string {
	if costPrice <= 0 {
		return "Cost price not available. Set stopLoss and takeProfit to null."
	}
	defaultStop := costPrice * 0.93
	defaultTake := costPrice * 1.12
	return fmt.Sprintf(
		"stopLoss: derive from cost price (%.4f). A 5-8%% trailing stop is standard risk management. "+
			"Default suggestion: %.4f (7%% below cost). "+
			"takeProfit: set at a meaningful resistance or upside target. "+
			"Default suggestion: %.4f (12%% above cost). "+
			"Both must be numeric price values. Set null only if fixed levels are inapplicable for this asset class.",
		costPrice, defaultStop, defaultTake,
	)
}

// recommendationEnvelope is the intermediate shape used to decode the
// recommendation sub-object from the LLM response. It exists so parsing
// failures in the sub-object do not fail the whole synthesis response.
type recommendationEnvelope struct {
	Recommendation *recommendation.Recommendation `json:"recommendation"`
}

// parseRecommendation attempts to decode the recommendation sub-object from a
// raw LLM JSON payload. Returns nil if the field is missing or malformed.
func parseRecommendation(jsonStr string) *recommendation.Recommendation {
	var env recommendationEnvelope
	if err := json.Unmarshal([]byte(jsonStr), &env); err != nil {
		return nil
	}
	if env.Recommendation == nil {
		return nil
	}
	// Normalize: ensure ActionLevel is derived from Action (LLM may omit it).
	env.Recommendation.ActionLevel = env.Recommendation.Action.Level()
	if env.Recommendation.Execution.ValidDays <= 0 {
		env.Recommendation.Execution.ValidDays = recommendation.ValidityDefaultDays
	}
	return env.Recommendation
}

// legacyToAction maps the legacy analysis.Recommendation string enum to the
// structured recommendation.Action type. The string values of the two enums
// are deliberately aligned, so a direct cast is sufficient.
func legacyToAction(r analysis.Recommendation) recommendation.Action {
	return recommendation.Action(string(r))
}

// executeImmediatelyTrigger is the canonical trigger value for time-based
// one-shot steps. The frontend format-trigger.ts detects this string and
// translates it via i18n, so it is intentionally kept in English regardless
// of the user's language preference.
const executeImmediatelyTrigger = "execute immediately"

// monitorTriggerFallback returns a language-specific trigger description used
// only when CostPrice is zero (no structured TriggerPayload available). When
// CostPrice > 0, the caller produces a structured price trigger instead.
func monitorTriggerFallback(lang string) string {
	if lang == "zh" {
		return "\u4ef7\u683c\u8dcc\u7834\u6b62\u635f\u7ebf" // 价格跌破止损线
	}
	return "price breaks below stop-loss"
}

// fallbackRecommendation builds a deterministic default recommendation when
// the LLM response is missing or malformed. It uses the matrix-derived action
// together with the user's current position and cost price to produce a
// single-step one-shot plan (or a monitor plan for hold).
//
// Rationale fields are intentionally left empty. RationaleTemplate is set
// instead so the frontend can resolve localized text from its i18n bundle.
func fallbackRecommendation(input *SynthesisInput) recommendation.Recommendation {
	action := legacyToAction(input.Recommendation)
	currentPct := input.PositionRatio * 100

	rec := recommendation.Recommendation{
		Action:             action,
		ActionLevel:        action.Level(),
		Label:              recommendationText(input.Recommendation),
		CurrentPositionPct: currentPct,
		TargetPositionPct:  currentPct,
		Execution: recommendation.Execution{
			Type:      recommendation.ExecutionOneShot,
			Steps:     nil,
			ValidDays: recommendation.ValidityDefaultDays,
		},
	}

	switch action {
	case recommendation.ActionAggressiveAdd:
		rec.TargetPositionPct = currentPct + 10
		rec.Execution.Steps = []recommendation.Step{{
			Order:             1,
			TriggerType:       recommendation.TriggerTime,
			TriggerValue:      executeImmediatelyTrigger,
			DeltaPct:          10,
			RationaleTemplate: "aggressiveAdd",
		}}
	case recommendation.ActionSmallAdd:
		rec.TargetPositionPct = currentPct + 5
		rec.Execution.Steps = []recommendation.Step{{
			Order:             1,
			TriggerType:       recommendation.TriggerTime,
			TriggerValue:      executeImmediatelyTrigger,
			DeltaPct:          5,
			RationaleTemplate: "smallAdd",
		}}
	case recommendation.ActionHold:
		rec.Execution.Type = recommendation.ExecutionMonitor
		rec.Execution.Steps = fallbackMonitorSteps(input)
		if input.CostPrice > 0 {
			stop := input.CostPrice * 0.95
			take := input.CostPrice * 1.10
			rec.Execution.StopLoss = &stop
			rec.Execution.TakeProfit = &take
		}
	case recommendation.ActionGradualReduce:
		rec.TargetPositionPct = clampNonNegative(currentPct - 10)
		rec.Execution.Steps = []recommendation.Step{{
			Order:             1,
			TriggerType:       recommendation.TriggerTime,
			TriggerValue:      executeImmediatelyTrigger,
			DeltaPct:          -10,
			RationaleTemplate: "gradualReduce",
		}}
	case recommendation.ActionControl:
		rec.TargetPositionPct = clampNonNegative(currentPct - 15)
		rec.Execution.Steps = []recommendation.Step{{
			Order:             1,
			TriggerType:       recommendation.TriggerTime,
			TriggerValue:      executeImmediatelyTrigger,
			DeltaPct:          -15,
			RationaleTemplate: "control",
		}}
	default:
		// Unknown action: treat as monitor with fallback steps.
		rec.Execution.Type = recommendation.ExecutionMonitor
		rec.Execution.Steps = fallbackMonitorSteps(input)
	}

	return rec
}

// fallbackMonitorSteps generates a single conditional watch step for
// monitor-type plans when the LLM did not provide steps. The step
// instructs the user to trim if price drops below the stop-loss level.
//
// Rationale fields are left empty; RationaleTemplate is set so the frontend
// resolves localized text from its i18n bundle.
func fallbackMonitorSteps(input *SynthesisInput) []recommendation.Step {
	triggerValue := monitorTriggerFallback(input.Language)
	var payload recommendation.TriggerPayload
	if input.CostPrice > 0 {
		stopPrice := input.CostPrice * 0.95
		triggerValue = fmt.Sprintf("%.4f below", stopPrice)
		payload = recommendation.TriggerPayload{
			PriceOp:    "below",
			PriceValue: stopPrice,
		}
	}
	return []recommendation.Step{
		{
			Order:             1,
			TriggerType:       recommendation.TriggerPrice,
			TriggerValue:      triggerValue,
			TriggerPayload:    payload,
			DeltaPct:          -5,
			RationaleTemplate: "monitor",
		},
	}
}

func clampNonNegative(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// ensureRecommendation normalizes a parsed recommendation by back-filling
// missing fields from the synthesis input. It is intended for the LLM success
// path where the LLM may have omitted currentPositionPct or label.
func ensureRecommendation(rec *recommendation.Recommendation, input *SynthesisInput) {
	if rec == nil {
		return
	}
	if rec.CurrentPositionPct == 0 {
		rec.CurrentPositionPct = input.PositionRatio * 100
	}
	if strings.TrimSpace(rec.Label) == "" {
		rec.Label = recommendationText(input.Recommendation)
	}
	if rec.Action == "" {
		rec.Action = legacyToAction(input.Recommendation)
	}
	rec.ActionLevel = rec.Action.Level()
	if rec.Execution.Type == "" {
		rec.Execution.Type = recommendation.ExecutionOneShot
	}
	if rec.Execution.ValidDays <= 0 {
		rec.Execution.ValidDays = recommendation.ValidityDefaultDays
	}
	// Monitor plans must have at least one watch step; if the LLM returned
	// an empty steps slice, inject fallback monitor steps.
	if rec.Execution.Type == recommendation.ExecutionMonitor && len(rec.Execution.Steps) == 0 {
		rec.Execution.Steps = fallbackMonitorSteps(input)
	}
}
