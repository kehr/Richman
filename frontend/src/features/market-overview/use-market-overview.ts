import { useQuery } from "@tanstack/react-query";
import { fetchMarketOverview } from "./api";
import type { MarketOverviewDto } from "./types";

// MARKET_OVERVIEW_KEY is the stable query key for the asset card wall.
const MARKET_OVERVIEW_KEY = ["market", "overview"] as const;

// useMarketOverview fetches the grouped asset card wall data.
// staleTime is 5 minutes to reduce redundant fetches while the user browses.
export function useMarketOverview() {
	return useQuery<MarketOverviewDto>({
		queryKey: MARKET_OVERVIEW_KEY,
		queryFn: fetchMarketOverview,
		staleTime: 5 * 60 * 1000,
		retry: 1,
	});
}
