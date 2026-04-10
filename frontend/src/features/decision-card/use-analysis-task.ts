import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { getAnalysisTask } from "./api";
import type { AnalysisTask } from "./types";
import { DECISION_CARDS_QUERY_KEY } from "./use-decision-cards";

export function useAnalysisTask(taskId: string | null): {
	task: AnalysisTask | undefined;
	isPolling: boolean;
} {
	const queryClient = useQueryClient();

	const { data } = useQuery({
		queryKey: ["analysis-task", taskId],
		queryFn: () => getAnalysisTask(taskId as string),
		enabled: taskId !== null,
		staleTime: 0,
		refetchInterval: (query) => {
			const status = query.state.data?.data?.status;
			return status === "running" || status === "pending" ? 1500 : false;
		},
	});

	const task = data?.data;

	useEffect(() => {
		if (task?.status === "completed") {
			queryClient.invalidateQueries({ queryKey: DECISION_CARDS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["decision-card"] });
			queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
		}
	}, [task?.status, queryClient]);

	return {
		task,
		isPolling: taskId !== null && task?.status === "running",
	};
}
