// Pure formatting utilities for percentage + optional CNY amount pairs used
// across decision cards, portfolio, and dashboard views. Kept free of React
// hooks so they can be unit tested in isolation. Locale-aware thousand
// separators come from Intl.NumberFormat with the "zh-CN" locale, which
// matches the product copy used everywhere in the MVP.

const cnyFormatter = new Intl.NumberFormat("zh-CN", {
	maximumFractionDigits: 0,
});

// formatPercent renders a percentage value with a single decimal when the
// value is non-integer, otherwise as an integer. We intentionally avoid
// toLocaleString here because percentages should not carry locale-specific
// grouping (they are small numbers by definition).
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
// minus sign before the symbol so the output looks like "-¥1,234".
export function formatAmount(amount: number): string {
	if (Number.isNaN(amount)) {
		return "¥0";
	}
	if (amount < 0) {
		return `-¥${cnyFormatter.format(Math.abs(amount))}`;
	}
	return `¥${cnyFormatter.format(amount)}`;
}

// formatPercentWithAmount composes "X% · ¥Y" when a capital is configured and
// an amount is known, falling back to just the percentage otherwise. The
// middle dot (·) is the separator used in the PRD mockups.
export function formatPercentWithAmount(
	pct: number,
	amount: number | null | undefined,
	hasCapital: boolean,
): string {
	if (!hasCapital || amount == null) {
		return formatPercent(pct);
	}
	return `${formatPercent(pct)} · ${formatAmount(amount)}`;
}

// formatAmountOrNull returns the formatted amount when a capital is configured
// and the amount is known; otherwise it returns null so callers can render a
// placeholder (e.g. hide the row, show a dash).
export function formatAmountOrNull(
	amount: number | null | undefined,
	hasCapital: boolean,
): string | null {
	if (!hasCapital || amount == null) {
		return null;
	}
	return formatAmount(amount);
}
