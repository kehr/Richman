import { useUserSettings } from "@/features/user-settings";
import { formatAmountOrNull, formatPercentWithAmount } from "./format";

// useMoney is the single hook every decision card / portfolio / dashboard
// view should call to render a "percentage + optional amount" pair. It pulls
// the user's total capital preference from the user-settings feature and
// exposes:
//   - hasCapital:       whether the user has configured totalCapitalCny
//   - format:           render "X% · ¥Y" when possible, otherwise just "X%"
//   - formatAmountOnly: render "¥Y" when possible, otherwise null
//
// Keeping all formatting paths behind a single hook means individual screens
// do not have to know whether the user opted into capital tracking.
export function useMoney() {
	const { data: settings } = useUserSettings();
	const hasCapital = settings?.totalCapitalCny != null;

	return {
		hasCapital,
		format: (pct: number, amount?: number | null) =>
			formatPercentWithAmount(pct, amount, hasCapital),
		formatAmountOnly: (amount?: number | null) => formatAmountOrNull(amount, hasCapital),
	};
}
