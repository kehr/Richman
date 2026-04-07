// Public barrel for the decision-card feature. Pages must consume the
// feature exclusively through this entry point; importing the internal
// component files directly is rejected by the Pages+Features dependency
// rules.

export { getDecisionCards, getDecisionCardById, postRerunAnalysis } from "./api";
export {
	useDecisionCards,
	DECISION_CARDS_QUERY_KEY,
} from "./use-decision-cards";
export {
	useDecisionCardDetail,
	decisionCardDetailQueryKey,
} from "./use-decision-card-detail";
export { useRerunAnalysis } from "./use-rerun-analysis";

export { computeNextAnalysisTime, formatHm } from "./analysis-schedule";

export { ChangeBadge, BADGE_TEXT } from "./components/ChangeBadge";
export { DimensionBadges } from "./components/DimensionBadges";
export { ExecutionPlanStrip } from "./components/ExecutionPlanStrip";
export { DecisionCardSummary } from "./components/DecisionCardSummary";

export type {
	DecisionCardDTO,
	Recommendation,
	Execution,
	ExecutionType,
	Step,
	TriggerType,
	TriggerPayload,
	Action,
	BadgeState,
	RerunAnalysisResponse,
} from "./types";
