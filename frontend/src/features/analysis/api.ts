import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface TriggerAnalysisDto {
	taskId: string;
	message: string;
}

export interface AnalysisTaskStatusDto {
	taskId: string;
	status: string;
	progress: number;
	error?: string;
	startedAt: string;
	doneAt?: string;
}

export function triggerAnalysis(): Promise<ApiResponse<TriggerAnalysisDto>> {
	return request<ApiResponse<TriggerAnalysisDto>>("/analysis/trigger", {
		method: "POST",
	});
}

export function fetchTaskStatus(taskId: string): Promise<ApiResponse<AnalysisTaskStatusDto>> {
	return request<ApiResponse<AnalysisTaskStatusDto>>(`/analysis/tasks/${taskId}`);
}
