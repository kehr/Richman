import { useMutation } from "@tanstack/react-query";
import { postRerunAnalysis, postRerunSingle } from "./api";

// useRerunAnalysis triggers a backend re-analysis. On success the taskId is
// passed to onTaskStarted so callers can open the progress drawer and begin
// polling. Cache invalidation is handled by useAnalysisTask when the task
// reaches completed status, so this mutation no longer invalidates directly.
export function useRerunAnalysis(onTaskStarted?: (taskId: string) => void) {
	return useMutation({
		mutationFn: () => postRerunAnalysis(),
		onSuccess: (data) => {
			onTaskStarted?.(data.data.taskId);
		},
	});
}

// useRerunSingle triggers re-analysis for a single holding. On success the
// taskId is passed to onTaskStarted so callers can open the progress drawer.
export function useRerunSingle(onTaskStarted?: (taskId: string) => void) {
	return useMutation({
		mutationFn: (holdingId: number) => postRerunSingle(holdingId),
		onSuccess: (data) => {
			onTaskStarted?.(data.data.taskId);
		},
	});
}
