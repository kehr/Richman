import { useQuery } from "@tanstack/react-query";
import { getHoldingHistory } from "./api";
import type { DecisionCardDTO } from "./types";

// useHoldingHistory loads the recent decision cards for a specific holding.
// Used by the MetaSidebar history strip on the card detail page. The query
// is disabled when holdingId is 0 (card not yet loaded) so no wasted request.
export function useHoldingHistory(holdingId: number, limit = 10) {
	return useQuery<DecisionCardDTO[]>({
		queryKey: ["decision-cards", "history", { holdingId, limit }] as const,
		queryFn: async () => {
			const res = await getHoldingHistory(holdingId, limit);
			return res.data;
		},
		enabled: holdingId > 0,
		staleTime: 30_000,
	});
}
