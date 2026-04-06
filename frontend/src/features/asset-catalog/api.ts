import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface AssetDto {
	code: string;
	name: string;
	nameEn: string;
	assetType: string;
	exchange: string;
}

export function fetchAssets(params?: {
	type?: string;
	keyword?: string;
}): Promise<ApiResponse<AssetDto[]>> {
	const query = new URLSearchParams();
	if (params?.type) query.set("type", params.type);
	if (params?.keyword) query.set("keyword", params.keyword);
	const qs = query.toString();
	return request<ApiResponse<AssetDto[]>>(`/assets${qs ? `?${qs}` : ""}`);
}

export function fetchAssetByCode(code: string): Promise<ApiResponse<AssetDto>> {
	return request<ApiResponse<AssetDto>>(`/assets/${code}`);
}
