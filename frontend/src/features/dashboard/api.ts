import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface DashboardStatsDto {
	holdingCount: number;
	totalPositions: number;
	latestAnalysisTime: string | null;
}

export function fetchDashboardStats(): Promise<ApiResponse<DashboardStatsDto>> {
	return request<ApiResponse<DashboardStatsDto>>("/dashboard/stats");
}
