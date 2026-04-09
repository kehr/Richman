import { useMutation, useQueryClient } from "@tanstack/react-query";
import { postReanalyzeAll } from "./api";
import { DECISION_CARDS_QUERY_KEY } from "./use-decision-cards";

// useReanalyzeAll triggers the bulk re-analysis endpoint and invalidates the
// caches that the dashboard banner reads from. On success the banner should
// disappear automatically because dashboard-summary recomputes
// needsReanalysis with the freshly upgraded cards.
//
// The dashboard-summary key is spelled inline as a literal here rather than
// imported from its owning feature because the dependency-cruiser forbids
// cross-feature imports. TanStack Query matches invalidations by array
// structure so the literal ["dashboard-summary"] is equivalent to whatever
// the dashboard-summary feature exports as its canonical key. Any change to
// the key shape must be made in both places — see
// features/dashboard-summary/use-dashboard-summary.ts.
export function useReanalyzeAll() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => postReanalyzeAll(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: DECISION_CARDS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["decision-card"] });
			queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
		},
	});
}
