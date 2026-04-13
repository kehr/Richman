// api.ts — HTTP calls for the asset detail feature.
// Public endpoints use requestPublic (no JWT).
// Auth-required endpoints use requestV2 (JWT attached).

import { requestPublic, requestV2 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type {
	AnalysisJobDto,
	AssetDetailDto,
	DemoPlanDto,
	OhlcvDto,
	ScoreHistoryDto,
	TriggerAnalysisResponseDto,
} from "./types";

export function fetchAssetDetail(code: string): Promise<ApiResponse<AssetDetailDto>> {
	return requestPublic<ApiResponse<AssetDetailDto>>(`/market/${code}`);
}

export function fetchAssetOhlcv(code: string, period: string): Promise<ApiResponse<OhlcvDto>> {
	return requestPublic<ApiResponse<OhlcvDto>>(`/market/${code}/ohlcv?period=${period}`);
}

export function fetchScoreHistory(
	code: string,
	days: number,
): Promise<ApiResponse<ScoreHistoryDto>> {
	return requestPublic<ApiResponse<ScoreHistoryDto>>(`/market/${code}/scores?days=${days}`);
}

export function fetchDemoPlan(code: string): Promise<ApiResponse<DemoPlanDto>> {
	return requestPublic<ApiResponse<DemoPlanDto>>(`/market/${code}/demo-plan`);
}

export function triggerHoldingAnalysis(
	holdingId: number,
): Promise<ApiResponse<TriggerAnalysisResponseDto>> {
	return requestV2<ApiResponse<TriggerAnalysisResponseDto>>(`/analysis/holding/${holdingId}`, {
		method: "POST",
	});
}

export function fetchAnalysisJob(jobId: string): Promise<ApiResponse<AnalysisJobDto>> {
	return requestV2<ApiResponse<AnalysisJobDto>>(`/analysis/jobs/${jobId}`);
}
