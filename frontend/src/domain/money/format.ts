// Pure formatting utilities for percentage + optional CNY amount pairs used
// across decision cards, portfolio, and dashboard views. Kept free of React
// hooks so they can be unit tested in isolation.
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

// formatAmount renders a CNY amount using locale thousand separators and
// prefixes the Chinese yuan symbol. Negative amounts are prefixed with the
// minus sign before the symbol so the output looks like "-¥1,234". Negative
// zero is normalized to "¥0" — Intl.NumberFormat preserves the sign on -0,
// but the UI should always display a zero balance without a minus sign.
export function formatAmount(amount: number, locale = "en"): string {
	if (Number.isNaN(amount)) {
		return "¥0";
	}
	// Normalize -0 and any tiny negative-but-rounds-to-zero value.
	if (amount === 0) {
		return "¥0";
	}
	const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
	const fmt = getNumberFormat(intlLocale, { maximumFractionDigits: 0 });
	if (amount < 0) {
		return `-¥${fmt.format(Math.abs(amount))}`;
	}
	return `¥${fmt.format(amount)}`;
}

// formatPercentWithAmount composes "X% · ¥Y" when a capital is configured and
// an amount is known, falling back to just the percentage otherwise. The
// middle dot (·) is the separator used in the PRD mockups.
export function formatPercentWithAmount(
	pct: number,
	amount: number | null | undefined,
	hasCapital: boolean,
	locale = "en",
): string {
	if (!hasCapital || amount == null) {
		return formatPercent(pct);
	}
	return `${formatPercent(pct)} · ${formatAmount(amount, locale)}`;
}

// formatAmountOrNull returns the formatted amount when a capital is configured
// and the amount is known; otherwise it returns null so callers can render a
// placeholder (e.g. hide the row, show a dash).
export function formatAmountOrNull(
	amount: number | null | undefined,
	hasCapital: boolean,
	locale = "en",
): string | null {
	if (!hasCapital || amount == null) {
		return null;
	}
	return formatAmount(amount, locale);
}
