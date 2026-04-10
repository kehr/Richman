import type { DisplayCurrency } from "@/features/user-settings";
import { useQuery } from "@tanstack/react-query";
import { getExchangeRates } from "./api";

export const EXCHANGE_RATES_QUERY_KEY = ["exchange-rates"] as const;

// useExchangeRates fetches and caches exchange rates from the backend.
// staleTime is 30 minutes; the backend refreshes its source hourly.
// Returns { rates: {} } when loading or on error — consumers degrade to CNY.
export function useExchangeRates(): { rates: Partial<Record<DisplayCurrency, number>> } {
	const { data } = useQuery({
		queryKey: EXCHANGE_RATES_QUERY_KEY,
		queryFn: getExchangeRates,
		staleTime: 30 * 60 * 1000,
		retry: 2,
		select: (d) => d.data,
	});
	return { rates: data?.rates ?? {} };
}
