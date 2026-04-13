import { useQuery } from "@tanstack/react-query";
import { fetchBriefing } from "./api";
import type { BriefingDto } from "./types";

export const BRIEFING_QUERY_KEY = ["briefing"] as const;

// useBriefing fetches the user's research briefing data.
// staleTime of 30s balances freshness with avoiding excessive API calls
// since briefing data changes at analysis cadence (not real-time).
export function useBriefing() {
	return useQuery<BriefingDto>({
		queryKey: BRIEFING_QUERY_KEY,
		queryFn: async () => {
			const res = await fetchBriefing();
			return res.data;
		},
		staleTime: 30_000,
	});
}
