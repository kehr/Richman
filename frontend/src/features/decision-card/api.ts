import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type {
	AnalysisTask,
	DecisionCardDTO,
	ProviderUsed,
	ReanalyzeAllResponse,
	RerunAnalysisResponse,
	SynthesisSource,
} from "./types";

// WireDecisionCard is the raw wire shape the backend sends. The only
// difference from DecisionCardDTO is that `synthesisSource` and
// `providerUsed` may be null/undefined for historical rows that pre-date
// migration 007. The normalize step below coerces those to the "unknown"
// sentinel so downstream code always sees a closed union.
type WireDecisionCard = Omit<DecisionCardDTO, "synthesisSource" | "providerUsed"> & {
	synthesisSource?: SynthesisSource | null;
	providerUsed?: ProviderUsed | null;
};

function normalizeSource(value: SynthesisSource | null | undefined): SynthesisSource {
	if (value === "llm" || value === "template" || value === "mixed") return value;
	return "unknown";
}

function normalizeProvider(value: ProviderUsed | null | undefined): ProviderUsed {
	if (value === "user" || value === "system_default" || value === "none") return value;
	return "unknown";
}

function normalizeCard(wire: WireDecisionCard): DecisionCardDTO {
	return {
		...wire,
		synthesisSource: normalizeSource(wire.synthesisSource),
		providerUsed: normalizeProvider(wire.providerUsed),
	};
}

// getDecisionCards loads the latest decision card for every holding owned by
// the authenticated user. The `latest=true` query parameter is accepted by
// the backend today but the handler already defaults to the "latest per
// holding" behaviour, so the query string is only added for forward
// compatibility with future filter modes.
export async function getDecisionCards(): Promise<ApiResponse<DecisionCardDTO[]>> {
	const res = await request<ApiResponse<WireDecisionCard[]>>("/decision-cards?latest=true");
	return { data: res.data.map(normalizeCard) };
}

// getDecisionCardById loads a single decision card by its primary key. The
// backend enforces that the card belongs to the authenticated user.
export async function getDecisionCardById(cardId: number): Promise<ApiResponse<DecisionCardDTO>> {
	const res = await request<ApiResponse<WireDecisionCard>>(`/decision-cards/${cardId}`);
	return { data: normalizeCard(res.data) };
}

// postRerunAnalysis triggers a re-analysis for the current user. The backend
// returns 202 Accepted with a task id the UI can poll (or just wait for the
// cache invalidation kicked off by useRerunAnalysis).
export function postRerunAnalysis() {
	return request<ApiResponse<RerunAnalysisResponse>>("/analysis/trigger", {
		method: "POST",
	});
}

// postReanalyzeAll triggers a bulk re-analysis across every active holding
// owned by the user. The degraded-contract dashboard banner uses this to
// upgrade template/mixed cards to LLM cards after a provider becomes
// healthy. The backend throttles the endpoint at the gateway so the client
// does not need its own rate limit.
export function postReanalyzeAll() {
	return request<ApiResponse<ReanalyzeAllResponse>>("/analysis/reanalyze-all", {
		method: "POST",
	});
}

// getHoldingHistory loads the recent decision cards for a specific holding.
// Returns up to `limit` cards ordered newest-first. This endpoint is used
// by the MetaSidebar history strip on the card detail page.
export async function getHoldingHistory(
	holdingId: number,
	limit = 10,
): Promise<ApiResponse<DecisionCardDTO[]>> {
	const res = await request<ApiResponse<WireDecisionCard[]>>(
		`/decision-cards/history?holding_id=${holdingId}&limit=${limit}`,
	);
	return { data: res.data.map(normalizeCard) };
}

// getAnalysisTask fetches the current status of an analysis task by ID.
export function getAnalysisTask(taskId: string): Promise<ApiResponse<AnalysisTask>> {
	return request<ApiResponse<AnalysisTask>>(`/analysis/tasks/${taskId}`);
}
