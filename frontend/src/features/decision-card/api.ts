import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface DecisionCardDto {
	cardId: number;
	assetCode: string;
	assetName: string;
	assetType: string;
	costPrice: number;
	positionRatio: number;
	trendDirection: string;
	trendSummary: string;
	positionDirection: string;
	positionSummary: string;
	catalystDirection: string;
	catalystSummary: string;
	confidence: number;
	recommendation: string;
	actionAdvice: string;
	detailedAdvice: string;
	riskWarnings: string[];
	todayHighlights: string;
	weightTrend: number;
	weightPosition: number;
	weightCatalyst: number;
	analyzedAt: string;
}

export function fetchLatestCards(): Promise<ApiResponse<DecisionCardDto[]>> {
	return request<ApiResponse<DecisionCardDto[]>>("/decision-cards/latest");
}

export function fetchCardById(id: number): Promise<ApiResponse<DecisionCardDto>> {
	return request<ApiResponse<DecisionCardDto>>(`/decision-cards/${id}`);
}

export function fetchCardHistory(): Promise<ApiResponse<DecisionCardDto[]>> {
	return request<ApiResponse<DecisionCardDto[]>>("/decision-cards/history");
}
