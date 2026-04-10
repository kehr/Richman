// Pure formatting utilities for percentage + optional CNY amount pairs used
// across decision cards, portfolio, and dashboard views. Kept free of React
// hooks so they can be unit tested in isolation.
import type { DisplayCurrency } from "@/features/user-settings";
import { getNumberFormat } from "./intl-cache";

// formatPercent renders a percentage value with a single decimal when the
// value is non-integer, otherwise as an integer. toFixed rounds to the
// nearest tenth using half-away-from-zero semantics (e.g. 3.25 -> "3.3%",
// 3.24 -> "3.2%"). We intentionally avoid toLocaleString here because
// percentages should not carry locale-specific grouping (they are small
// numbers by definition). NaN is coerced to "0%" to protect UI layers from
// divide-by-zero or missing upstream data.
export function formatPercent(pct: number): string {
	if (Number.isNaN(pct)) {
		return "0%";
	}
	if (Number.isInteger(pct)) {
		return `${pct}%`;
	}
	return `${pct.toFixed(1)}%`;
}

// getCurrencyLocale maps a DisplayCurrency to the Intl locale that produces
// the correct symbol. USD always uses en-US ($); HKD uses en-US (HK$ via
// symbol display); CNY follows the UI locale (zh-CN or en-US, both render ¥).
function getCurrencyLocale(currency: DisplayCurrency, uiLocale: string): string {
	if (currency === "USD") return "en-US";
	if (currency === "HKD") return "en-US";
	return uiLocale === "zh" ? "zh-CN" : "en-US";
}

// getCurrencyDisplay returns the Intl currencyDisplay option for each currency.
// HKD must use "symbol" to render "HK$"; narrowSymbol renders bare "$" for HKD
// and is indistinguishable from USD. USD and CNY use "narrowSymbol" which
// strips country prefixes (e.g. US$) for a cleaner display.
function getCurrencyDisplay(
	currency: DisplayCurrency,
): Intl.NumberFormatOptions["currencyDisplay"] {
	if (currency === "HKD") return "symbol";
	return "narrowSymbol";
}

// formatAmount renders an amount using the currency's standard Intl format.
// currency defaults to CNY (¥). Negative amounts are formatted by the Intl
// formatter (e.g. -$1,234). NaN and -0 are rendered as the zero value.
export function formatAmount(
	amount: number,
	locale = "en",
	currency: DisplayCurrency = "CNY",
): string {
	const intlLocale = getCurrencyLocale(currency, locale);
	const currencyDisplay = getCurrencyDisplay(currency);
	if (Number.isNaN(amount) || amount === 0) {
		return getNumberFormat(intlLocale, {
			style: "currency",
			currency,
			maximumFractionDigits: 0,
			currencyDisplay,
		}).format(0);
	}
	return getNumberFormat(intlLocale, {
		style: "currency",
		currency,
		maximumFractionDigits: 0,
		currencyDisplay,
	}).format(amount);
}

// formatPercentWithAmount composes "X% · ¥Y" (or $Y / HK$Y) when a capital is
// configured and an amount is known, falling back to just the percentage
// otherwise. The middle dot (·) is the separator used in the PRD mockups.
export function formatPercentWithAmount(
	pct: number,
	amount: number | null | undefined,
	hasCapital: boolean,
	locale = "en",
	currency: DisplayCurrency = "CNY",
): string {
	if (!hasCapital || amount == null) {
		return formatPercent(pct);
	}
	return `${formatPercent(pct)} · ${formatAmount(amount, locale, currency)}`;
}

// formatAmountOrNull returns the formatted amount when a capital is configured
// and the amount is known; otherwise it returns null so callers can render a
// placeholder (e.g. hide the row, show a dash).
export function formatAmountOrNull(
	amount: number | null | undefined,
	hasCapital: boolean,
	locale = "en",
	currency: DisplayCurrency = "CNY",
): string | null {
	if (!hasCapital || amount == null) {
		return null;
	}
	return formatAmount(amount, locale, currency);
}
