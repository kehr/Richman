// getPriceChangeColor returns the correct color token name for a price change,
// taking into account the asset market convention (SS4.4 in frontend-v2-trd.md).
//
// A-share detection: code is exactly 6 decimal digits (e.g. "000001", "518880").
// A-shares use the Chinese convention: red = up, green = down.
// All other assets (US stocks, gold futures, crypto) use international convention:
// green = up, red = down.
export function getPriceChangeColor(
	assetCode: string,
	changePercent: number,
): "red" | "green" | "gray" {
	const isAShare = /^\d{6}$/.test(assetCode);
	if (changePercent > 0) return isAShare ? "red" : "green";
	if (changePercent < 0) return isAShare ? "green" : "red";
	return "gray";
}

// getDirectionColor returns the color for a signal/direction label.
// Direction labels always follow international convention: green = bullish, red = bearish.
// Handles two enum spaces: richson `SignalLevel` (strong_/moderate_) for asset
// analysis, and `goldDirection` three-level (bullish/bearish/neutral) for the
// event radar. Both map to the same green/red/gray palette.
export function getDirectionColor(signal: string): "green" | "red" | "gray" {
	if (signal === "bullish" || signal === "strong_bullish" || signal === "moderate_bullish") {
		return "green";
	}
	if (signal === "bearish" || signal === "strong_bearish" || signal === "moderate_bearish") {
		return "red";
	}
	return "gray";
}

// formatCurrencyPrice formats a price value according to its currency.
// USD: "$4,750.00" — CNY: "CN¥4.85"
export function formatCurrencyPrice(price: number, currency: string): string {
	if (currency === "USD") {
		return `$${price.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
	}
	if (currency === "CNY") {
		return `CN¥${price.toLocaleString("zh-CN", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
	}
	return `${price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} ${currency}`;
}
