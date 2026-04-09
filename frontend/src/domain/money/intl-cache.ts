const cache = new Map<string, Intl.NumberFormat>();

export function getNumberFormat(
	locale: string,
	options: Intl.NumberFormatOptions,
): Intl.NumberFormat {
	const key = `${locale}:${JSON.stringify(options)}`;
	let fmt = cache.get(key);
	if (!fmt) {
		fmt = new Intl.NumberFormat(locale, options);
		cache.set(key, fmt);
	}
	return fmt;
}
