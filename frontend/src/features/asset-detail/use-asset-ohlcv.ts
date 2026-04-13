import { useQuery } from "@tanstack/react-query";
import { fetchAssetOhlcv } from "./api";
import type { OhlcvDto } from "./types";

export type OhlcvPeriod = "1D" | "1W" | "1M" | "3M" | "1Y";

export function useAssetOhlcv(code: string, period: OhlcvPeriod, enabled = true) {
	return useQuery<OhlcvDto>({
		queryKey: ["asset-ohlcv", code, period] as const,
		queryFn: async () => {
			const res = await fetchAssetOhlcv(code, period);
			return res.data;
		},
		enabled: enabled && !!code,
		staleTime: 5 * 60_000,
	});
}
