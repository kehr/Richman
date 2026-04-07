import { getToken } from "@/domain/auth/storage";
import { ApiError, request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { RecognizeResponse } from "./screenshot-types";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

export interface HoldingDto {
	holdingId: number;
	assetCode: string;
	assetName: string;
	assetType: string;
	costPrice: number;
	positionRatio: number;
	quantity: number;
}

export interface CreateHoldingInput {
	assetCode: string;
	assetName: string;
	assetType: string;
	costPrice: number;
	positionRatio: number;
}

export interface TradeDto {
	tradeId: number;
	holdingId: number;
	direction: string;
	price: number;
	quantity: number;
	tradedAt: string;
}

export interface CreateTradeInput {
	direction: string;
	price: number;
	quantity: number;
	tradedAt: string;
}

export function fetchHoldings(): Promise<ApiResponse<HoldingDto[]>> {
	return request<ApiResponse<HoldingDto[]>>("/holdings");
}

export function createHolding(data: CreateHoldingInput): Promise<ApiResponse<HoldingDto>> {
	return request<ApiResponse<HoldingDto>>("/holdings", {
		method: "POST",
		body: JSON.stringify(data),
	});
}

export function updateHolding(
	id: number,
	data: Partial<CreateHoldingInput>,
): Promise<ApiResponse<HoldingDto>> {
	return request<ApiResponse<HoldingDto>>(`/holdings/${id}`, {
		method: "PATCH",
		body: JSON.stringify(data),
	});
}

export function deleteHolding(id: number): Promise<ApiResponse<null>> {
	return request<ApiResponse<null>>(`/holdings/${id}`, {
		method: "DELETE",
	});
}

export function fetchTrades(holdingId: number): Promise<ApiResponse<TradeDto[]>> {
	return request<ApiResponse<TradeDto[]>>(`/holdings/${holdingId}/trades`);
}

export function createTrade(
	holdingId: number,
	data: CreateTradeInput,
): Promise<ApiResponse<TradeDto>> {
	return request<ApiResponse<TradeDto>>(`/holdings/${holdingId}/trades`, {
		method: "POST",
		body: JSON.stringify(data),
	});
}

// importPortfolioScreenshot uploads a single image as multipart/form-data to
// the screenshot recognition endpoint. The standard request() helper always
// sets a JSON Content-Type, so we go to fetch directly here and reuse the
// same auth header strategy.
export async function importPortfolioScreenshot(
	file: File,
): Promise<ApiResponse<RecognizeResponse>> {
	const token = getToken();
	const form = new FormData();
	form.append("file", file);
	const response = await fetch(`${API_BASE}/portfolio/import-screenshot`, {
		method: "POST",
		headers: token ? { Authorization: `Bearer ${token}` } : undefined,
		body: form,
	});
	if (!response.ok) {
		const body = await response.json().catch(() => ({}));
		throw new ApiError(
			response.status,
			body?.error?.code || "UNKNOWN",
			body?.error?.message || response.statusText,
		);
	}
	return (await response.json()) as ApiResponse<RecognizeResponse>;
}
