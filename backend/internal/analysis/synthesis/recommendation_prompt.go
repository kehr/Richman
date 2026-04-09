package synthesis

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/richman/backend/internal/analysis"
	"github.com/richman/backend/internal/analysis/recommendation"
)

// recommendationPromptSection returns the prompt fragment that instructs the
// LLM to emit a structured "recommendation" sub-object alongside the existing
// natural-language fields. It is appended to the main synthesis prompt.
func recommendationPromptSection() string {
	var sb strings.Builder
	sb.WriteString("\nAlso include a top-level \"recommendation\" sub-object with this shape:\n")
	sb.WriteString(`{
  "action": "aggressive_add|small_add|hold|gradual_reduce|control_position",
  "label": "short human-readable label",
  "currentPositionPct": 0.0,
  "targetPositionPct": 0.0,
  "execution": {
    "type": "one-shot|staged|monitor",
    "steps": [
      {
        "order": 1,
        "triggerType": "price|time|event",
        "triggerValue": "short condition text",
        "deltaPct": 5.0,
        "rationale": {
          "triggerReason": "why this trigger condition (1 sentence)",
          "positionReason": "why this delta size (1 sentence)",
          "precondition": "what must be true before acting (1 sentence)",
          "fallback": "what to do if trigger missed (1 sentence)",
          "timeWindow": "expected timeframe (1 sentence)"
        }
      }
    ],
    "stopLoss": null,
    "takeProfit": null,
    "validDays": 7
  }
}`)
	sb.WriteString("\nUse stopLoss / takeProfit as numeric price levels when relevant, otherwise omit or set null.\n")
	sb.WriteString("For hold recommendations: use type=\"monitor\" with 1-2 conditional watch steps ")
	sb.WriteString("(triggerType=\"price\" or \"event\"). These steps represent conditions to watch, ")
	sb.WriteString("not immediate actions. Monitor steps should have negative or zero deltaPct.\n")
	return sb.String()
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

// fallbackRecommendation builds a deterministic default recommendation when
// the LLM response is missing or malformed. It uses the matrix-derived action
// together with the user's current position and cost price to produce a
// single-step one-shot plan (or a monitor plan for hold).
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
			Order:        1,
			TriggerType:  recommendation.TriggerTime,
			TriggerValue: "execute immediately",
			DeltaPct:     10,
			Rationale: recommendation.StructuredRationale{
				TriggerReason: "Aggressive add per matrix decision.",
			},
		}}
	case recommendation.ActionSmallAdd:
		rec.TargetPositionPct = currentPct + 5
		rec.Execution.Steps = []recommendation.Step{{
			Order:        1,
			TriggerType:  recommendation.TriggerTime,
			TriggerValue: "execute immediately",
			DeltaPct:     5,
			Rationale: recommendation.StructuredRationale{
				TriggerReason: "Small add per matrix decision.",
			},
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
			Order:        1,
			TriggerType:  recommendation.TriggerTime,
			TriggerValue: "execute immediately",
			DeltaPct:     -10,
			Rationale: recommendation.StructuredRationale{
				TriggerReason: "Gradual reduce per matrix decision.",
			},
		}}
	case recommendation.ActionControl:
		rec.TargetPositionPct = clampNonNegative(currentPct - 15)
		rec.Execution.Steps = []recommendation.Step{{
			Order:        1,
			TriggerType:  recommendation.TriggerTime,
			TriggerValue: "execute immediately",
			DeltaPct:     -15,
			Rationale: recommendation.StructuredRationale{
				TriggerReason: "Control position per matrix decision.",
			},
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
func fallbackMonitorSteps(input *SynthesisInput) []recommendation.Step {
	triggerValue := "price breaks below stop-loss"
	if input.CostPrice > 0 {
		triggerValue = fmt.Sprintf("%.4f below", input.CostPrice*0.95)
	}
	return []recommendation.Step{
		{
			Order:        1,
			TriggerType:  recommendation.TriggerPrice,
			TriggerValue: triggerValue,
			DeltaPct:     -5,
			Rationale: recommendation.StructuredRationale{
				TriggerReason:  "Reduce if price breaks below stop-loss to limit downside.",
				PositionReason: "Moderate trim to observe before further action.",
				Precondition:   "Price closes below stop-loss level on consecutive days.",
				Fallback:       "If price recovers above cost, continue holding.",
				TimeWindow:     "Continuous monitoring.",
			},
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
