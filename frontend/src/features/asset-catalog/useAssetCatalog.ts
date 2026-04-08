import { useQuery } from "@tanstack/react-query";
import { fetchAssetByCode, fetchAssets } from "./api";

// Namespaced query keys: the second element distinguishes list vs detail
// so devtools show a clean hierarchy and `queryClient.invalidateQueries`
// can target either shape via a single prefix. The normalized tuple keys
// also avoid object-identity instability when the caller constructs
// `params` inline on every render.
export function useAssets(params?: { type?: string; keyword?: string }) {
	return useQuery({
		queryKey: ["assets", "list", params?.type ?? null, params?.keyword ?? null],
		queryFn: () => fetchAssets(params),
		select: (res) => res.data,
	});
}

export function useAssetByCode(code: string) {
	return useQuery({
		queryKey: ["assets", "detail", code],
		queryFn: () => fetchAssetByCode(code),
		select: (res) => res.data,
		enabled: !!code,
	});
}
