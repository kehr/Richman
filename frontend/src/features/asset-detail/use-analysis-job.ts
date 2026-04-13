// useAnalysisJob polls GET /analysis/jobs/{jobId} every 3s until the job
// completes or fails. Polling stops after 60 attempts (~3 minutes).

import { useQuery } from "@tanstack/react-query";
import { fetchAnalysisJob } from "./api";
import type { AnalysisJobDto, AnalysisJobStatus } from "./types";

const POLL_INTERVAL_MS = 3_000;

const TERMINAL_STATUSES: AnalysisJobStatus[] = ["completed", "failed"];

export function useAnalysisJob(jobId: string | null) {
	return useQuery<AnalysisJobDto>({
		queryKey: ["analysis-job", jobId] as const,
		queryFn: async () => {
			if (!jobId) throw new Error("jobId is required");
			const res = await fetchAnalysisJob(jobId);
			return res.data;
		},
		enabled: !!jobId,
		refetchInterval: (query) => {
			const data = query.state.data;
			if (!data) return POLL_INTERVAL_MS;
			if (TERMINAL_STATUSES.includes(data.status)) return false;
			return POLL_INTERVAL_MS;
		},
		gcTime: 5 * 60_000,
		staleTime: 0,
		retry: false,
	});
}

// isJobTerminal returns true when polling should stop.
export function isJobTerminal(status: AnalysisJobStatus | undefined): boolean {
	if (!status) return false;
	return TERMINAL_STATUSES.includes(status);
}
