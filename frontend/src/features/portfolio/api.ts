import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

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
