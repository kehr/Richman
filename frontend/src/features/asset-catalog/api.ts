import { requestV1 } from "@/domain/http/client";
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
	return requestV1<ApiResponse<AssetDto[]>>(`/assets${qs ? `?${qs}` : ""}`);
}

export function fetchAssetByCode(code: string): Promise<ApiResponse<AssetDto>> {
	return requestV1<ApiResponse<AssetDto>>(`/assets/${code}`);
}
