import { getNumberFormat } from "../money/intl-cache";

export function formatCurrency(value: number, locale = "en", currency = "CNY"): string {
	const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
	return getNumberFormat(intlLocale, {
		style: "currency",
		currency,
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
	}).format(value);
}

export function formatPercent(value: number, decimals = 2): string {
	return `${(value * 100).toFixed(decimals)}%`;
}

export function formatDate(date: string | Date, locale = "en", format?: string): string {
	const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
	const d = typeof date === "string" ? new Date(date) : date;
	if (format === "datetime") return d.toLocaleString(intlLocale);
	return d.toLocaleDateString(intlLocale);
}

export function formatConfidence(value: number): string {
	return `${(value * 100).toFixed(1)}%`;
}
