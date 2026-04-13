import { requestPublic } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { MarketOverviewDto, MarketRegimeDto } from "./types";

// fetchMarketRegime loads the current market regime signal and index snapshots.
// Uses requestPublic — no JWT required (public page).
export async function fetchMarketRegime(): Promise<MarketRegimeDto> {
	const res = await requestPublic<ApiResponse<MarketRegimeDto>>("/market/regime");
	return res.data;
}

// fetchMarketOverview loads the grouped asset card wall data.
// Uses requestPublic — no JWT required (public page).
export async function fetchMarketOverview(): Promise<MarketOverviewDto> {
	const res = await requestPublic<ApiResponse<MarketOverviewDto>>("/market/overview");
	return res.data;
}
