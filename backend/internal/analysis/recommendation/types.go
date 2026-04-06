// Package recommendation defines structured recommendation data types
// (Action, Execution, Step, Recommendation) and pure helpers for the
// product-flow card model. The package is intentionally dependency-free
// (stdlib only) so it can be safely imported by service, repo, and diff
// layers without introducing cycles.
package recommendation

import "time"

// ConfidenceShiftThreshold is the minimum absolute confidence delta (in
// percentage points) that triggers a confidence-shift badge state.
// See PRD §3.4.
const ConfidenceShiftThreshold = 10.0

// ValidityDefaultDays is the default lifespan of an execution plan in days
// when the upstream LLM does not provide an explicit valid_days value.
// See TRD §3.4.
const ValidityDefaultDays = 7

// Action enumerates the five base recommendation actions returned by the
// synthesis layer. The string values are the canonical wire format used in
// JSON payloads and database rows.
type Action string

const (
	ActionAggressiveAdd Action = "aggressive_add"
	ActionSmallAdd      Action = "small_add"
	ActionHold          Action = "hold"
	ActionGradualReduce Action = "gradual_reduce"
	ActionControl       Action = "control_position"
)

// Level returns the integer aggressiveness level for an action. The level
// scale is symmetric around hold and is the sole input used by the badge
// diff algorithm to detect upgrades / downgrades. See PRD §3.4
// "建议积极度等级".
//
//	aggressive_add  ->  2
//	small_add       ->  1
//	hold            ->  0
//	gradual_reduce  -> -1
//	control_position -> -2
//
// Unknown actions return 0 (treated as neutral) so that malformed upstream
// data does not falsely register as an upgrade or downgrade.
func (a Action) Level() int {
	switch a {
	case ActionAggressiveAdd:
		return 2
	case ActionSmallAdd:
		return 1
	case ActionHold:
		return 0
	case ActionGradualReduce:
		return -1
	case ActionControl:
		return -2
	default:
		return 0
	}
}

// ExecutionType describes the shape of an execution plan. one-shot plans
// run a single buy / sell, staged plans break the trade into ordered steps,
// and monitor plans hold the existing position with stop-loss / take-profit
// guardrails (no steps).
type ExecutionType string

const (
	ExecutionOneShot ExecutionType = "one-shot"
	ExecutionStaged  ExecutionType = "staged"
	ExecutionMonitor ExecutionType = "monitor"
)

// TriggerType is the discriminator for a step's trigger condition.
type TriggerType string

const (
	TriggerPrice TriggerType = "price"
	TriggerTime  TriggerType = "time"
	TriggerEvent TriggerType = "event"
)

// TriggerPayload is the optional structured representation of a trigger
// condition. Only the fields relevant to the TriggerType are populated.
// All fields are tagged omitempty so the JSON output stays compact.
type TriggerPayload struct {
	PriceOp     string     `json:"priceOp,omitempty"`
	PriceValue  float64    `json:"priceValue,omitempty"`
	DeadlineISO *time.Time `json:"deadlineIso,omitempty"`
	EventKey    string     `json:"eventKey,omitempty"`
}

// Step is a single ordered action inside an execution plan.
type Step struct {
	Order          int            `json:"order"`
	TriggerType    TriggerType    `json:"triggerType"`
	TriggerValue   string         `json:"triggerValue"`
	TriggerPayload TriggerPayload `json:"triggerPayload"`
	DeltaPct       float64        `json:"deltaPct"`
	Rationale      string         `json:"rationale"`
}

// Execution is a complete execution plan attached to a Recommendation.
// Steps is empty for monitor plans; StopLoss / TakeProfit are optional and
// represented as pointers so that "no guard" can be distinguished from
// "guard at price 0".
type Execution struct {
	Type       ExecutionType `json:"type"`
	Steps      []Step        `json:"steps,omitempty"`
	StopLoss   *float64      `json:"stopLoss,omitempty"`
	TakeProfit *float64      `json:"takeProfit,omitempty"`
	ValidDays  int           `json:"validDays"`
}

// Recommendation is the full structured recommendation surfaced on a card.
// ActionLevel is denormalized from Action so that diff consumers do not
// need to import this package just to call Action.Level.
type Recommendation struct {
	Action             Action    `json:"action"`
	ActionLevel        int       `json:"actionLevel"`
	Label              string    `json:"label"`
	CurrentPositionPct float64   `json:"currentPositionPct"`
	TargetPositionPct  float64   `json:"targetPositionPct"`
	Execution          Execution `json:"execution"`
}
