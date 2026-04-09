import { useUserSettings } from "@/features/user-settings";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { formatAmountOrNull, formatPercentWithAmount } from "./format";

// useMoney is the single hook every decision card / portfolio / dashboard
// view should call to render a "percentage + optional amount" pair. It pulls
// the user's total capital preference from the user-settings feature and
// exposes:
//   - hasCapital:       whether the user has configured totalCapitalCny
//   - format:           render "X% · ¥Y" when possible, otherwise just "X%"
//   - formatAmountOnly: render "¥Y" when possible, otherwise null
//
// The returned object and its functions are memoized on hasCapital and locale
// so that consumers (Dashboard decision card list, Portfolio table, etc.) that
// pass them into React.memo children or useEffect dependency arrays do not bust
// memoization on every render.
//
// During initial load (useUserSettings still fetching), hasCapital is false
// and the hook renders percent-only output. The first paint after hydration
// will therefore replace any visible amount — this is accepted UX because
// the OnboardingGuard shell already prevents rendering the main app tree
// until settings are in cache, so the flash is not observable in practice.
export function useMoney() {
	const { data: settings } = useUserSettings();
	const hasCapital = settings?.totalCapitalCny != null;
	const { i18n } = useTranslation();
	const locale = i18n.language;

	return useMemo(
		() => ({
			hasCapital,
			format: (pct: number, amount?: number | null) =>
				formatPercentWithAmount(pct, amount, hasCapital, locale),
			formatAmountOnly: (amount?: number | null) => formatAmountOrNull(amount, hasCapital, locale),
		}),
		[hasCapital, locale],
	);
}
