import { useQuery } from "@tanstack/react-query";
import { getDecisionCardById } from "./api";
import type { DecisionCardDTO } from "./types";

// decisionCardDetailQueryKey builds a stable query key for a single card.
export function decisionCardDetailQueryKey(cardId: number) {
	return ["decision-card", cardId] as const;
}

// useDecisionCardDetail loads a single decision card by id. The query is
// disabled for non-positive ids so that accidental `NaN` from router params
// does not trigger a network request. staleTime is 60s because the detail
// view is navigated to from the list and the underlying row rarely changes
// within a single session.
export function useDecisionCardDetail(cardId: number) {
	return useQuery<DecisionCardDTO>({
		queryKey: decisionCardDetailQueryKey(cardId),
		queryFn: async () => {
			const res = await getDecisionCardById(cardId);
			return res.data;
		},
		enabled: Number.isFinite(cardId) && cardId > 0,
		staleTime: 60_000,
	});
}
