// Formats a date as a localized relative time string (e.g., "2 minutes ago", "2 分钟前").
// Returns an em dash when the input date is null or undefined.

const THRESHOLDS: Array<{ limit: number; unit: Intl.RelativeTimeFormatUnit; divisor: number }> = [
	{ limit: 60, unit: "second", divisor: 1 },
	{ limit: 3600, unit: "minute", divisor: 60 },
	{ limit: 86400, unit: "hour", divisor: 3600 },
	{ limit: 604800, unit: "day", divisor: 86400 },
	{ limit: 2592000, unit: "week", divisor: 604800 },
	{ limit: 31536000, unit: "month", divisor: 2592000 },
];

const YEAR_DIVISOR = 31536000;

export function formatRelativeTime(date: string | Date | null | undefined, lang: string): string {
	if (date == null) {
		return "\u2014";
	}

	const diff = (Date.now() - new Date(date).getTime()) / 1000;

	let unit: Intl.RelativeTimeFormatUnit = "year";
	let divisor = YEAR_DIVISOR;

	for (const threshold of THRESHOLDS) {
		if (Math.abs(diff) < threshold.limit) {
			unit = threshold.unit;
			divisor = threshold.divisor;
			break;
		}
	}

	return new Intl.RelativeTimeFormat(lang, { style: "long" }).format(
		-Math.round(diff / divisor),
		unit,
	);
}
