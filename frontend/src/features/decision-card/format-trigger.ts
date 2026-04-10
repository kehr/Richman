import { useTranslation } from "react-i18next";
import type { Step } from "./types";

// Known price operation keys emitted by the backend. The LLM may produce
// additional values; unrecognized ops fall back to the raw triggerValue.
const PRICE_OPS = ["below", "above", "at_or_below", "at_or_above"] as const;
type PriceOp = (typeof PRICE_OPS)[number];

function isPriceOp(v: string): v is PriceOp {
	return (PRICE_OPS as readonly string[]).includes(v);
}

// useFormatTriggerValue returns a formatter that renders a localized trigger
// description from the structured triggerPayload when available, falling back
// to the raw triggerValue string for legacy cards or unknown trigger types.
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
			const op = t(`decisionCard.executionPlan.trigger.op.${payload.priceOp}`);
			return `${payload.priceValue.toFixed(4)} ${op}`;
		}

		// Time trigger: check for "execute immediately" pattern.
		if (step.triggerType === "time") {
			const raw = step.triggerValue.toLowerCase().trim();
			if (raw === "execute immediately" || raw === "\u7acb\u5373\u6267\u884c") {
				return t("decisionCard.executionPlan.trigger.executeImmediately");
			}
		}

		// Fallback: return the backend-provided string as-is.
		return step.triggerValue;
	};
}
