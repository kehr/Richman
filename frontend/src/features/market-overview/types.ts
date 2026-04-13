// Types for the Market Overview feature (SS3.1 in frontend-v2-trd.md).

export interface IndexSnapshotDto {
	code: string;
	name: string;
	price: number;
	changePercent: number;
	currency: string;
}

export interface MarketRegimeDto {
	regime: "risk_on" | "neutral" | "risk_off";
	summary: string;
	indices: IndexSnapshotDto[];
	updatedAt: string;
}

export interface AssetCardDto {
	code: string;
	nameZh: string;
	nameEn: string;
	currency: string;
	price: number | null;
	changePercent: number | null;
	overallScore: number | null;
	signal: "strong_bullish" | "bullish" | "neutral" | "bearish" | "strong_bearish" | null;
	percentileLabel: string | null;
	isActive: boolean;
}

export interface AssetGroupDto {
	category: string;
	categoryLabel: string;
	assets: AssetCardDto[];
}

export interface MarketOverviewDto {
	groups: AssetGroupDto[];
	updatedAt: string;
}
