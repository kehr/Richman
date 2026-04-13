import { useQuery } from "@tanstack/react-query";
import { fetchMarketRegime } from "./api";
import type { MarketRegimeDto } from "./types";

// MARKET_REGIME_KEY is the stable query key for market regime data.
const MARKET_REGIME_KEY = ["market", "regime"] as const;

// useMarketRegime fetches the current market regime signal and index snapshots.
// staleTime is 5 minutes — the regime signal is updated at most hourly.
// On richson 503, TanStack Query surfaces an error which the component uses
// to hide the regime bar (G3.9).
export function useMarketRegime() {
	return useQuery<MarketRegimeDto>({
		queryKey: MARKET_REGIME_KEY,
		queryFn: fetchMarketRegime,
		staleTime: 5 * 60 * 1000,
		retry: 1,
	});
}
