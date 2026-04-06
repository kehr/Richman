export function formatCurrency(value: number, currency = "CNY"): string {
	return new Intl.NumberFormat("zh-CN", {
		style: "currency",
		currency,
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
	}).format(value);
}

export function formatPercent(value: number, decimals = 2): string {
	return `${(value * 100).toFixed(decimals)}%`;
}

export function formatDate(date: string | Date, format?: string): string {
	const d = typeof date === "string" ? new Date(date) : date;
	if (format === "date") {
		return d.toLocaleDateString("zh-CN");
	}
	if (format === "datetime") {
		return d.toLocaleString("zh-CN");
	}
	return d.toLocaleDateString("zh-CN");
}

export function formatConfidence(value: number): string {
	return `${(value * 100).toFixed(1)}%`;
}
