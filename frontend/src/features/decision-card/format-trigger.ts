import { useTranslation } from "react-i18next";
import type { Step } from "./types";

// Known price operation keys emitted by the backend. The LLM may produce
// additional values; unrecognized ops fall back to the raw triggerValue.
const PRICE_OPS = ["below", "above", "at_or_below", "at_or_above"] as const;
type PriceOp = (typeof PRICE_OPS)[number];

function isPriceOp(v: string): v is PriceOp {
	return (PRICE_OPS as readonly string[]).includes(v);
}

// Regex to extract a numeric price and a trailing operator keyword from the
// raw triggerValue string (e.g. "3200.03 below", "15.50 at_or_above").
const TRIGGER_VALUE_RE = /^([\d.]+)\s+(below|above|at_or_below|at_or_above)$/i;

// useFormatTriggerValue returns a formatter that renders a localized trigger
// description from the structured triggerPayload when available, falling back
// to parsing the raw triggerValue string for legacy cards.
export function useFormatTriggerValue(): (step: Step) => string {
	const { t } = useTranslation("app");

	return (step: Step) => {
		const payload = step.triggerPayload;

		// Price trigger with structured payload.
		if (
			step.triggerType === "price" &&
			payload?.priceOp &&
			payload.priceValue != null &&
			isPriceOp(payload.priceOp)
		) {
			const op = t(`decisionCard.executionPlan.trigger.op.${payload.priceOp}`, {
				defaultValue: payload.priceOp,
			});
			return `${payload.priceValue.toFixed(4)} ${op}`;
		}

		// Time trigger: check for "execute immediately" pattern.
		if (step.triggerType === "time") {
			const raw = step.triggerValue.toLowerCase().trim();
			if (raw === "execute immediately" || raw === "\u7acb\u5373\u6267\u884c") {
				return t("decisionCard.executionPlan.trigger.executeImmediately");
			}
		}

		// Fallback: try to parse operator from the raw triggerValue string so
		// legacy cards (missing triggerPayload) still get localized operators.
		const match = step.triggerValue.match(TRIGGER_VALUE_RE);
		if (match) {
			const opKey = match[2].toLowerCase();
			if (isPriceOp(opKey)) {
				const op = t(`decisionCard.executionPlan.trigger.op.${opKey}`, {
					defaultValue: opKey,
				});
				return `${match[1]} ${op}`;
			}
		}

		return step.triggerValue;
	};
}
