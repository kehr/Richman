import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface AnalysisTaskDto {
	taskId: string;
	status: string;
	progress: number;
	message: string;
}

export function triggerAnalysis(): Promise<ApiResponse<AnalysisTaskDto>> {
	return request<ApiResponse<AnalysisTaskDto>>("/analysis/trigger", {
		method: "POST",
	});
}

export function fetchTaskStatus(taskId: string): Promise<ApiResponse<AnalysisTaskDto>> {
	return request<ApiResponse<AnalysisTaskDto>>(`/analysis/tasks/${taskId}`);
}
