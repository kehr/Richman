import { useQuery } from "@tanstack/react-query";
import { fetchAssetDetail } from "./api";
import type { AssetDetailDto } from "./types";

export const ASSET_DETAIL_KEY = (code: string) => ["asset-detail", code] as const;

export function useAssetDetail(code: string) {
	return useQuery<AssetDetailDto>({
		queryKey: ASSET_DETAIL_KEY(code),
		queryFn: async () => {
			const res = await fetchAssetDetail(code);
			return res.data;
		},
		enabled: !!code,
		staleTime: 60_000,
		retry: 1,
	});
}
