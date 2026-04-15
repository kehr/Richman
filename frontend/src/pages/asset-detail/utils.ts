// utils.ts — formatting helpers for the asset detail page.

// PLACEHOLDER is the UI fallback when a numeric field is absent. Backend MVP
// returns a subset of the TRD payload (see richman-v2-plan-execution-report.md
// Round 3-4 observation), so every consumer must tolerate undefined values.
export const PLACEHOLDER = "—";

// formatPrice formats a price according to the currency convention.
// USD assets use "$" prefix. CNY assets use "CN" prefix.
// Returns PLACEHOLDER when price or currency is unavailable.
export function formatPrice(
	price: number | undefined | null,
	currency: "USD" | "CNY" | undefined,
): string {
	if (price === undefined || price === null || Number.isNaN(price)) return PLACEHOLDER;
	const c = currency ?? "USD";
	if (c === "USD") {
		return `$${price.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
	}
	return `CN${price.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`;
}

// formatUsdEquiv formats a CNY price as its USD equivalent.
// Returns null when usdExchangeRate is not available.
export function formatUsdEquiv(
	priceCny: number,
	usdExchangeRate: number | null | undefined,
): string | null {
	if (!usdExchangeRate) return null;
	const usd = priceCny * usdExchangeRate;
	return `~$${usd.toLocaleString("en-US", { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`;
}

// getPriceChangeColor returns the color for a price change percentage.
// A-share assets use Chinese convention (red = up, green = down).
// All other assets use international convention (green = up, red = down).
export function getPriceChangeColor(
	assetCode: string,
	changePercent: number | undefined | null,
): string {
	if (changePercent === undefined || changePercent === null) return "#8c8c8c";
	const isAShare = /^\d{6}$/.test(assetCode);
	if (changePercent > 0) return isAShare ? "#f5222d" : "#52c41a";
	if (changePercent < 0) return isAShare ? "#52c41a" : "#f5222d";
	return "#8c8c8c";
}

// getSignalColor returns the color for a signal level.
// Direction labels always use international convention.
// Accepts both richson `SignalLevel` (strong_/moderate_) and legacy plain
// bullish/bearish tokens used elsewhere in the app (e.g. dimension direction).
export function getSignalColor(signal: string | undefined): string {
	if (signal === "bullish" || signal === "strong_bullish" || signal === "moderate_bullish") {
		return "#52c41a";
	}
	if (signal === "bearish" || signal === "strong_bearish" || signal === "moderate_bearish") {
		return "#f5222d";
	}
	return "#8c8c8c";
}

// computePriceDriftPercent computes the percentage drift of current price
// relative to the price at analysis time. Returns 0 when either is missing
// so consumers can skip rendering the freshness indicator.
export function computePriceDriftPercent(
	currentPrice: number | undefined,
	priceAtAnalysis: number | undefined,
): number {
	if (!currentPrice || !priceAtAnalysis) return 0;
	return Math.abs((currentPrice - priceAtAnalysis) / priceAtAnalysis) * 100;
}
