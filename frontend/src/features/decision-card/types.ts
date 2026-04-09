// Decision card TypeScript types.
//
// These types mirror the backend DTO defined in
// backend/internal/api/v1/decision_card.go (DecisionCardDTO) and the shared
// recommendation structures in backend/internal/analysis/recommendation.
// They are intentionally a direct structural copy so the UI can consume the
// JSON payload as-is without a runtime mapper. Keep the field order / names
// in sync with the Go source when changing either side.

// TriggerType is the discriminator for a step's trigger condition.
export type TriggerType = "price" | "time" | "event";

// ExecutionType describes the shape of an execution plan.
export type ExecutionType = "one-shot" | "staged" | "monitor";

// Action enumerates the five base recommendation actions. String values are
// the canonical wire format used in JSON payloads.
export type Action =
	| "aggressive_add"
	| "small_add"
	| "hold"
	| "gradual_reduce"
	| "control_position";

// BadgeState is the 8-state badge enum from PRD §3.4. "none" means no badge
// should be displayed.
export type BadgeState =
	| "none"
	| "data_degraded"
	| "first_analysis"
	| "action_upgrade"
	| "action_downgrade"
	| "signal_flip"
	| "plan_adjust"
	| "confidence_shift";

// SynthesisSource records whether the card text came from an LLM, a
// deterministic rules engine fallback, or a mix of the two. Backend nullable
// values are mapped to "unknown" at the API boundary so the UI can branch on
// a closed union.
export type SynthesisSource = "llm" | "template" | "mixed" | "unknown";

// ProviderUsed records which layer of the fallback chain actually produced
// the card. "user" means the user-configured provider answered, "system_default"
// means Richman's shared provider answered, "none" means neither LLM layer
// answered (pure template), and "unknown" is the null-at-rest sentinel.
export type ProviderUsed = "user" | "system_default" | "none" | "unknown";

// TriggerPayload is the optional structured representation of a trigger
// condition. Only fields relevant to the TriggerType are populated.
export interface TriggerPayload {
	priceOp?: string;
	priceValue?: number;
	deadlineIso?: string;
	eventKey?: string;
}

// Step is a single ordered action inside an execution plan.
export interface Step {
	order: number;
	triggerType: TriggerType;
	triggerValue: string;
	triggerPayload?: TriggerPayload;
	deltaPct: number;
	rationale: string;
}

// Execution is a complete execution plan attached to a Recommendation.
export interface Execution {
	type: ExecutionType;
	steps?: Step[];
	stopLoss?: number | null;
	takeProfit?: number | null;
	validDays: number;
}

// Recommendation is the full structured recommendation surfaced on a card.
export interface Recommendation {
	action: Action;
	actionLevel: number;
	label: string;
	currentPositionPct: number;
	targetPositionPct: number;
	execution: Execution;
}

// DecisionCardDTO mirrors backend v1.DecisionCardDTO.
export interface DecisionCardDTO {
	cardId: number;
	userId: number;
	holdingId: number;
	assetCode: string;
	assetName: string;
	assetType: string;
	costPrice: number;
	positionRatio: number;
	positionAmount?: number | null;
	trendDirection: string;
	trendSummary: string;
	positionDirection: string;
	positionSummary: string;
	catalystDirection: string;
	catalystSummary: string;
	confidence: number;
	actionAdvice: string;
	detailedAdvice: string;
	riskWarnings: string[];
	todayHighlights: string;
	weightTrend: number;
	weightPosition: number;
	weightCatalyst: number;
	analyzedAt: string;
	createdAt: string;

	// Structured recommendation + badge diff fields (migration 006).
	recommendation: Recommendation;
	actionLevel: number;
	targetPositionRatio: number;
	targetPositionAmount?: number | null;
	badgeState: BadgeState;
	confidenceDelta: number;
	prevCardId?: number | null;
	executionFingerprint: string;

	// Provenance fields added by the LLM degraded contract work. Backend may
	// emit null for historical rows that have not yet been reanalyzed; the
	// API client normalizes null to "unknown" so downstream components can
	// rely on the closed union.
	synthesisSource: SynthesisSource;
	providerUsed: ProviderUsed;
}

// RerunAnalysisResponse is the accepted response for POST /analysis/trigger.
export interface RerunAnalysisResponse {
	taskId: string;
	message?: string;
}

// ReanalyzeAllResponse is the accepted response for
// POST /analysis/reanalyze-all. It mirrors RerunAnalysisResponse shape on
// purpose so UI callers can reuse the same progress polling surface.
export interface ReanalyzeAllResponse {
	taskId: string;
	message?: string;
}
