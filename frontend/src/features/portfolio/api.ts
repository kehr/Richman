import { getToken } from "@/domain/auth/storage";
import { API_V1_BASE, ApiError, requestV1 as request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { RecognizeResponse } from "./screenshot-types";
import type { CreateTradeInput } from "./trade-types";

// HoldingDto mirrors backend/internal/api/v1/portfolio.go HoldingDTO.
// PositionAmount is populated server-side by user_settings.AttachAmounts
// when the user has a total_capital_cny configured; when the user has not
// set a total capital it is omitted entirely so the frontend can fall
// through to percent-only rendering via the shared useMoney hook.
export interface HoldingDto {
	holdingId: number;
	userId: number;
	assetCode: string;
	assetName: string;
	assetType: string;
	category?: string | null;
	costPrice: number;
	positionRatio: number;
	positionAmount?: number | null;
	quantity: number;
	entryMode?: "tag" | "quick" | "detail" | null;
	createdAt: string;
	updatedAt: string;
}

export interface CreateHoldingInput {
	assetCode: string;
	assetName: string;
	assetType: string;
	costPrice: number;
	positionRatio: number;
	quantity: number;
}

// TradeDto mirrors backend/internal/api/v1/portfolio.go TradeDTO. Price and
// quantity are projected from the backend decimal.Decimal into float64 so
// the frontend receives plain numbers (without the projection decimal would
// marshal to quoted strings and `toFixed()` calls would blow up at runtime).
export interface TradeDto {
	tradeId: number;
	holdingId: number;
	userId: number;
	direction: string;
	price: number;
	quantity: number;
	tradedAt: string;
	createdAt: string;
	updatedAt: string;
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
	const response = await fetch(`${API_V1_BASE}/portfolio/import-screenshot`, {
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
