// Public barrel for the decision-card feature. Pages must consume the
// feature exclusively through this entry point; importing the internal
// component files directly is rejected by the Pages+Features dependency
// rules.

export {
	getDecisionCards,
	getDecisionCardById,
	getHoldingHistory,
	postRerunAnalysis,
	postReanalyzeAll,
	postRerunSingle,
} from "./api";
export {
	useDecisionCards,
	DECISION_CARDS_QUERY_KEY,
} from "./use-decision-cards";
export { useHoldingHistory } from "./use-holding-history";
export {
	useDecisionCardDetail,
	decisionCardDetailQueryKey,
} from "./use-decision-card-detail";
export { useRerunAnalysis, useRerunSingle } from "./use-rerun-analysis";
export { useReanalyzeAll } from "./use-reanalyze-all";
export { useAnalysisTask } from "./use-analysis-task";
export { AnalysisProgressDrawer } from "./components/AnalysisProgressDrawer";

export { computeNextAnalysisTime, formatHm } from "./analysis-schedule";

export { ChangeBadge, BADGE_TEXT } from "./components/ChangeBadge";
export { DimensionBadges } from "./components/DimensionBadges";
export { ExecutionPlanStrip } from "./components/ExecutionPlanStrip";
export { DecisionCardSummary } from "./components/DecisionCardSummary";
export { SourcePill } from "./components/SourcePill";

export { useFormatTriggerValue } from "./format-trigger";
export { isStructuredRationale, isV2Card } from "./types";

export type {
	DecisionCardDTO,
	Recommendation,
	Execution,
	ExecutionType,
	Step,
	StructuredRationale,
	TriggerType,
	TriggerPayload,
	Action,
	BadgeState,
	SynthesisSource,
	ProviderUsed,
	RerunAnalysisResponse,
	ReanalyzeAllResponse,
	AnalysisTask,
	HoldingProgress,
	AnalysisTaskStep,
	AnalysisTaskLog,
	HoldingAnalysisStatus,
	AnalysisTaskStatus,
	TaskStepKey,
	TaskStepStatus,
} from "./types";
