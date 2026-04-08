import { useQuery } from "@tanstack/react-query";
import { getDashboardSummary } from "./api";
import type { DashboardSummaryDTO } from "./types";

// DASHBOARD_SUMMARY_QUERY_KEY is the stable query key used by every
// dashboard-adjacent consumer of the summary payload. The same array
// literal is spelled inline by sibling features (e.g. decision-card's
// useReanalyzeAll) because dependency-cruiser forbids cross-feature
// imports; TanStack Query matches invalidations by array structure so the
// two definitions stay equivalent as long as their literals match.
export const DASHBOARD_SUMMARY_QUERY_KEY = ["dashboard-summary"] as const;

// useDashboardSummary reads the dashboard summary payload. The staleTime is
// short (10s) so the banner/needsReanalysis signal updates within one tick
// after a reanalyze-all mutation settles.
export function useDashboardSummary() {
	return useQuery<DashboardSummaryDTO>({
		queryKey: DASHBOARD_SUMMARY_QUERY_KEY,
		queryFn: async () => {
			const res = await getDashboardSummary();
			return res.data;
		},
		staleTime: 10_000,
	});
}
