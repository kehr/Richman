import type { DisplayCurrency } from "@/features/user-settings";
import { useUserSettings } from "@/features/user-settings";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { formatAmountOrNull, formatPercentWithAmount } from "./format";
import { useExchangeRates } from "./useExchangeRates";

// convertCny converts a CNY amount to the target display currency using the
// provided rates (expressed as "1 CNY = X foreign"). Returns amountCny
// unchanged when: currency is "CNY", the rate is missing, or the rate is 0.
// Returns null when amountCny is null/undefined (preserves null semantics).
function convertCny(
	amountCny: number | null | undefined,
	currency: DisplayCurrency,
	rates: Partial<Record<DisplayCurrency, number>>,
): number | null {
	if (amountCny == null) return null;
	if (currency === "CNY") return amountCny;
	const rate = rates[currency];
	if (!rate) return amountCny; // degraded: show CNY value when rate unavailable
	return amountCny * rate;
}

// useMoney is the single hook every decision card / portfolio / dashboard
// view should call to render a "percentage + optional amount" pair. It reads
// the user's capital preference, display currency, and live exchange rates,
// then exposes:
//   - hasCapital:       whether the user has configured totalCapitalCny
//   - currency:         the current display currency (CNY | USD | HKD)
//   - format:           render "X% · ¥Y" (or $Y / HK$Y) when possible
//   - formatAmountOnly: render the converted amount, or null
export function useMoney() {
	const { data: settings } = useUserSettings();
	const { rates } = useExchangeRates();
	const { i18n } = useTranslation();

	const hasCapital = settings?.totalCapitalCny != null;
	const currency: DisplayCurrency = settings?.displayCurrency ?? "CNY";
	const locale = i18n.language;

	// When the rate for the selected currency is unavailable, format amounts in
	// CNY so numbers and symbols stay consistent (¥15,000 not $15,000 with a
	// CNY value). Falls back to CNY both when rates are loading and on error.
	const effectiveCurrency: DisplayCurrency =
		currency === "CNY" || rates[currency] != null ? currency : "CNY";

	return useMemo(
		() => ({
			hasCapital,
			currency,
			format: (pct: number, amountCny?: number | null) =>
				formatPercentWithAmount(
					pct,
					convertCny(amountCny, currency, rates),
					hasCapital,
					locale,
					effectiveCurrency,
				),
			formatAmountOnly: (amountCny?: number | null) =>
				formatAmountOrNull(
					convertCny(amountCny, currency, rates),
					hasCapital,
					locale,
					effectiveCurrency,
				),
		}),
		[hasCapital, currency, rates, locale, effectiveCurrency],
	);
}
