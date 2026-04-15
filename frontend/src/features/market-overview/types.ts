// Types for the Market Overview feature (SS3.1 in frontend-v2-trd.md).
// Field names must match the backend `AssetCardDTO` / `AssetGroupDTO` /
// `MarketOverviewDTO` exactly — see docs/standards/contract-drift.md.

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

// AssetCardDto mirrors backend `internal/service/market/service.go AssetCardDTO`.
// Fields with `*T` on the Go side must be `T | null` on the TS side.
// Optional (omitempty) fields come in as undefined when absent.
export interface AssetCardDto {
	code: string;
	name: string;
	nameEn: string;
	assetType: string;
	exchange: string;
	overallScore?: number | null;
	signalLevel?:
		| "strong_bullish"
		| "moderate_bullish"
		| "neutral"
		| "moderate_bearish"
		| "strong_bearish"
		| null;
	scoreDelta?: number | null;
}

// AssetGroupDto mirrors backend `AssetGroupDTO`. The `assetType` serves as both
// grouping key and i18n lookup for the section header label.
export interface AssetGroupDto {
	assetType: string;
	assets: AssetCardDto[];
}

export interface MarketOverviewDto {
	groups: AssetGroupDto[];
	updatedAt: string;
}
