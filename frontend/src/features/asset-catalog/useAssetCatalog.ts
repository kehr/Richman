import { useQuery } from "@tanstack/react-query";
import { fetchAssetByCode, fetchAssets } from "./api";

export function useAssets(params?: { type?: string; keyword?: string }) {
	return useQuery({
		queryKey: ["assets", params],
		queryFn: () => fetchAssets(params),
		select: (res) => res.data,
	});
}

export function useAssetByCode(code: string) {
	return useQuery({
		queryKey: ["assets", code],
		queryFn: () => fetchAssetByCode(code),
		select: (res) => res.data,
		enabled: !!code,
	});
}
