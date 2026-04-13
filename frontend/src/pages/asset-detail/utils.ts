// utils.ts — formatting helpers for the asset detail page.

// formatPrice formats a price according to the currency convention.
// USD assets use "$" prefix. CNY assets use "CN" prefix.
export function formatPrice(price: number, currency: "USD" | "CNY"): string {
	if (currency === "USD") {
		return `$${price.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
	}
	return `CN${price.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`;
}

// formatUsdEquiv formats a CNY price as its USD equivalent.
// Returns null when usdExchangeRate is not available.
export function formatUsdEquiv(priceCny: number, usdExchangeRate: number | null): string | null {
	if (!usdExchangeRate) return null;
	const usd = priceCny * usdExchangeRate;
	return `~$${usd.toLocaleString("en-US", { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`;
}

// getPriceChangeColor returns the color for a price change percentage.
// A-share assets use Chinese convention (red = up, green = down).
// All other assets use international convention (green = up, red = down).
export function getPriceChangeColor(assetCode: string, changePercent: number): string {
	const isAShare = /^\d{6}$/.test(assetCode);
	if (changePercent > 0) return isAShare ? "#f5222d" : "#52c41a";
	if (changePercent < 0) return isAShare ? "#52c41a" : "#f5222d";
	return "#8c8c8c";
}

// getSignalColor returns the color for a signal level.
// Direction labels always use international convention.
export function getSignalColor(signal: string): string {
	if (signal === "bullish" || signal === "strong_bullish") return "#52c41a";
	if (signal === "bearish" || signal === "strong_bearish") return "#f5222d";
	return "#8c8c8c";
}

// computePriceDriftPercent computes the percentage drift of current price
// relative to the price at analysis time.
export function computePriceDriftPercent(currentPrice: number, priceAtAnalysis: number): number {
	if (!priceAtAnalysis) return 0;
	return Math.abs((currentPrice - priceAtAnalysis) / priceAtAnalysis) * 100;
}
