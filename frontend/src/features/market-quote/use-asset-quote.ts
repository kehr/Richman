import { useQuery } from "@tanstack/react-query";
import { fetchAssetQuote } from "./api";
import type { AssetQuoteDTO } from "./types";

// assetQuoteQueryKey builds a stable query key for asset quotes.
export function assetQuoteQueryKey(assetType: string, assetCode: string) {
	return ["asset-quote", assetType, assetCode] as const;
}

// useAssetQuote loads the latest quote and recent history for an asset.
// The query is disabled when assetType or assetCode is empty.
// staleTime is 120s to align with the backend in-memory cache TTL.
export function useAssetQuote(assetType: string, assetCode: string) {
	return useQuery<AssetQuoteDTO>({
		queryKey: assetQuoteQueryKey(assetType, assetCode),
		queryFn: async () => {
			const res = await fetchAssetQuote(assetType, assetCode);
			return res.data;
		},
		enabled: !!assetType && !!assetCode,
		staleTime: 120_000,
	});
}
