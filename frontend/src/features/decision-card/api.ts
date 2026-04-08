import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { DecisionCardDTO, RerunAnalysisResponse } from "./types";

// getDecisionCards loads the latest decision card for every holding owned by
// the authenticated user. The `latest=true` query parameter is accepted by
// the backend today but the handler already defaults to the "latest per
// holding" behaviour, so the query string is only added for forward
// compatibility with future filter modes.
export function getDecisionCards() {
	return request<ApiResponse<DecisionCardDTO[]>>("/decision-cards?latest=true");
}

// getDecisionCardById loads a single decision card by its primary key. The
// backend enforces that the card belongs to the authenticated user.
export function getDecisionCardById(cardId: number) {
	return request<ApiResponse<DecisionCardDTO>>(`/decision-cards/${cardId}`);
}

// postRerunAnalysis triggers a re-analysis for the current user. The backend
// returns 202 Accepted with a task id the UI can poll (or just wait for the
// cache invalidation kicked off by useRerunAnalysis).
export function postRerunAnalysis() {
	return request<ApiResponse<RerunAnalysisResponse>>("/analysis/trigger", {
		method: "POST",
	});
}
