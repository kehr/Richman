import { Alert } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface Props {
	drift: number; // percentage points
	analysisTime: string;
}

// FreshnessIndicator shows a tiered warning when the current price has drifted
// significantly from the price at analysis time.
// > 2%: yellow warning, > 5%: orange warning, > 10%: red error
export function FreshnessIndicator({ drift }: Props) {
	const { t } = useTranslation("app");
	const pct = drift.toFixed(1);

	let type: "warning" | "error" = "warning";
	let key: "mild" | "moderate" | "severe" = "mild";

	if (drift > 10) {
		type = "error";
		key = "severe";
	} else if (drift > 5) {
		type = "warning";
		key = "moderate";
	}

	return (
		<Alert
			type={type}
			showIcon
			message={t(`assetDetail.freshness.${key}`, { percent: pct })}
			style={{
				margin: "4px 0",
				fontSize: 12,
				...(drift > 5 && drift <= 10 ? { borderColor: "#fa8c16", color: "#fa8c16" } : {}),
			}}
		/>
	);
}
