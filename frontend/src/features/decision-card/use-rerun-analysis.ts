import { useMutation, useQueryClient } from "@tanstack/react-query";
import { postRerunAnalysis } from "./api";
import { DECISION_CARDS_QUERY_KEY } from "./use-decision-cards";

// useRerunAnalysis triggers a backend re-analysis. On success we invalidate
// both the list and detail caches so the UI picks up the freshly written
// card on the next render. The detail cache is invalidated via the shared
// `decision-card` prefix; individual ids do not need to be enumerated.
export function useRerunAnalysis() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => postRerunAnalysis(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: DECISION_CARDS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["decision-card"] });
		},
	});
}
