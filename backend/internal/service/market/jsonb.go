package market

import "encoding/json"

// rawDemoPlanScenario mirrors the per-scenario shape inside richson's
// demo_plan JSONB. richson writes snake_case keys; the service layer maps
// them to the camelCase frontend DTO.
type rawDemoPlanScenario struct {
	Condition string `json:"condition"`
	Action    string `json:"action"`
	Rationale string `json:"rationale"`
	Priority  int    `json:"priority"`
}

// rawDemoPlan is the snake_case shape richson writes into
// rs_asset_analyses.demo_plan. Only fields consumed by the frontend DTO are
// declared; unused fields like is_demo_plan / current_position /
// target_position / lot_count / exclusion_group / no_trigger_note are
// intentionally omitted because they never reach the frontend.
type rawDemoPlan struct {
	Action               string                `json:"action"`
	ActionLabel          string                `json:"action_label"`
	DefaultAction        string                `json:"default_action"`
	Scenarios            []rawDemoPlanScenario `json:"scenarios"`
	StopLoss             *float64              `json:"stop_loss"`
	TakeProfit           *float64              `json:"take_profit"`
	ValidDays            *int                  `json:"valid_days"`
	ConcentrationMessage *string               `json:"concentration_message"`
}

// rawDrawdownReference mirrors analysis_metadata.drawdown_reference. Per
// richson/src/richson/core/drawdown.py these keys are already camelCase, so
// json tags here intentionally use camelCase as well.
type rawDrawdownReference struct {
	CurrentBullRunStart   *string  `json:"currentBullRunStart"`
	MaxDrawdown           *float64 `json:"maxDrawdown"`
	MaxDrawdownDate       *string  `json:"maxDrawdownDate"`
	HistoricalAvgDrawdown *float64 `json:"historicalAvgDrawdown"`
}

// rawAnalysisMetadata mirrors rs_asset_analyses.analysis_metadata. The
// support/resistance keys are snake_case (richson canonicalised before write);
// drawdown_reference is the only nested object and uses camelCase internally.
type rawAnalysisMetadata struct {
	DrawdownReference *rawDrawdownReference `json:"drawdown_reference"`
	SupportLevels     []float64             `json:"support_levels"`
	ResistanceLevels  []float64             `json:"resistance_levels"`
}

// unmarshalDemoPlan decodes rs_asset_analyses.demo_plan. Returns nil when the
// input is nil/empty or decoding fails so callers can apply a fallback without
// panicking.
func unmarshalDemoPlan(raw json.RawMessage) *rawDemoPlan {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var dp rawDemoPlan
	if err := json.Unmarshal(raw, &dp); err != nil {
		return nil
	}
	return &dp
}

// unmarshalAnalysisMetadata decodes rs_asset_analyses.analysis_metadata.
// Returns nil for nil/empty input or decode failures.
func unmarshalAnalysisMetadata(raw json.RawMessage) *rawAnalysisMetadata {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var meta rawAnalysisMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}
	return &meta
}

// unmarshalRiskFactors decodes rs_asset_analyses.risk_factors. richson today
// writes a flat []string; future enhancements may add structured rows but the
// service layer wraps each entry into a RiskFactorDTO so callers stay stable.
// Returns nil for nil/empty input or decode failures.
func unmarshalRiskFactors(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var factors []string
	if err := json.Unmarshal(raw, &factors); err != nil {
		return nil
	}
	return factors
}
