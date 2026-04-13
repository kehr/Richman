import { useQuery } from "@tanstack/react-query";
import { fetchScoreHistory } from "./api";
import type { ScoreHistoryDto } from "./types";

export type ScoreHistoryDays = 30 | 90 | 180 | 240;

export function useScoreHistory(code: string, days: ScoreHistoryDays, enabled = true) {
	return useQuery<ScoreHistoryDto>({
		queryKey: ["asset-score-history", code, days] as const,
		queryFn: async () => {
			const res = await fetchScoreHistory(code, days);
			return res.data;
		},
		enabled: enabled && !!code,
		staleTime: 5 * 60_000,
	});
}
