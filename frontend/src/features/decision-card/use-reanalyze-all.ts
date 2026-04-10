import { useMutation } from "@tanstack/react-query";
import { postReanalyzeAll } from "./api";

// useReanalyzeAll triggers the bulk re-analysis endpoint. On success the
// taskId is passed to onTaskStarted so callers can open the progress drawer
// and begin polling. Cache invalidation is handled by useAnalysisTask when
// the task reaches completed status — this mutation no longer invalidates
// directly. The dashboard-summary invalidation is also deferred to that hook.
export function useReanalyzeAll(onTaskStarted?: (taskId: string) => void) {
	return useMutation({
		mutationFn: () => postReanalyzeAll(),
		onSuccess: (data) => {
			onTaskStarted?.(data.data.taskId);
		},
	});
}
