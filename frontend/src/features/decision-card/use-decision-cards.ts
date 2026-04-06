import { useQuery } from "@tanstack/react-query";
import { getDecisionCards } from "./api";
import type { DecisionCardDTO } from "./types";

// DECISION_CARDS_QUERY_KEY is the stable list query key used by the Dashboard
// and invalidated by the rerun-analysis mutation on success.
export const DECISION_CARDS_QUERY_KEY = ["decision-cards", { latest: true }] as const;

// useDecisionCards loads the user's latest decision card per holding.
// staleTime is 30s because analysis runs on a cron schedule, so aggressive
// refetching only wastes requests between cron ticks.
export function useDecisionCards() {
	return useQuery<DecisionCardDTO[]>({
		queryKey: DECISION_CARDS_QUERY_KEY,
		queryFn: async () => {
			const res = await getDecisionCards();
			return res.data;
		},
		staleTime: 30_000,
	});
}
