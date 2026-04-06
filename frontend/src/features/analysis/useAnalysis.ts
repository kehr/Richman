import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchTaskStatus, triggerAnalysis, type AnalysisTaskStatusDto } from "./api";

export function useTriggerAnalysis() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: triggerAnalysis,
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["dashboard"] });
		},
	});
}

export function useTaskStatus(taskId: string | null) {
	return useQuery({
		queryKey: ["analysis", "task", taskId],
		queryFn: () => fetchTaskStatus(taskId as string),
		select: (res) => res.data,
		enabled: !!taskId,
		refetchInterval: (query) => {
			const status = (query.state.data as AnalysisTaskStatusDto | undefined)?.status;
			if (status === "completed" || status === "failed") return false;
			return 3000;
		},
	});
}
