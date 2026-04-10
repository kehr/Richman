import { Typography } from "@/ui-kit/eat";
import type { CSSProperties } from "react";

const { Text } = Typography;

// Preset color pairs per convention.
// "cn": A-share standard — red = up / green = down (红涨绿跌).
// "us": Western P&L standard — green = positive / red = negative.
const CONVENTION_COLORS = {
	cn: { positive: "#f5222d", negative: "#52c41a", neutral: "#8c8c8c" },
	us: { positive: "#52c41a", negative: "#f5222d", neutral: "#8c8c8c" },
} as const;

const ALIGN_ITEMS: Record<"left" | "center" | "right", CSSProperties["alignItems"]> = {
	left: "flex-start",
	center: "center",
	right: "flex-end",
};

export interface DeltaDisplayProps {
	/** Numeric percentage value (e.g. 5 for 5%, -10 for -10%). */
	pct: number;
	/** Pre-formatted amount string shown as the secondary value (e.g. "¥1,500"). */
	amount?: string | null;
	/**
	 * Color convention.
	 * "cn" (default): red = positive (涨/加仓), green = negative (跌/减仓).
	 * "us": green = positive (profit), red = negative (loss).
	 */
	convention?: "cn" | "us";
	/** Override color for positive values (overrides convention). */
	positiveColor?: string;
	/** Override color for negative values (overrides convention). */
	negativeColor?: string;
	/** Override color for zero. Default: #8c8c8c. */
	neutralColor?: string;
	/**
	 * Layout direction.
	 * "vertical" (default): amount stacks below percentage.
	 * "horizontal": amount sits to the right, bottom-aligned.
	 */
	layout?: "vertical" | "horizontal";
	/**
	 * Which value is visually primary (rendered at primarySize, bold).
	 * "pct" (default) | "amount".
	 */
	primary?: "pct" | "amount";
	/** Font size for the primary value. Default: 14. */
	primarySize?: number;
	/** Font size for the secondary value. Default: 11. */
	secondarySize?: number;
	/**
	 * Prepend "+" to positive values.
	 * Default: true — suitable for delta displays (+5%, -10%).
	 * Set false for absolute values where the sign is implicit (e.g. P&L %).
	 */
	showSign?: boolean;
	/** Decimal places for the percentage. Default: 0. */
	precision?: number;
	/**
	 * Whether to render the percentage value.
	 * Useful when only the amount should be shown.
	 * Default: true.
	 */
	showPct?: boolean;
	/**
	 * Whether to render the amount value when one is provided.
	 * Useful when pct and amount are both available but only pct is needed.
	 * Default: true.
	 */
	showAmount?: boolean;
	/** Text / flex alignment for vertical layout. Default: "right". */
	align?: "left" | "center" | "right";
	style?: CSSProperties;
	"data-testid"?: string;
}

// DeltaDisplay renders a signed percentage with an optional amount label in a
// configurable layout. It is the single source of truth for delta-colored
// number display across the app — execution plan steps, dashboard P&L, and
// any future delta surfaces all route through this component so the color
// convention can be changed in one place.
export function DeltaDisplay({
	pct,
	amount,
	convention = "cn",
	positiveColor,
	negativeColor,
	neutralColor,
	layout = "vertical",
	primary = "pct",
	primarySize = 14,
	secondarySize = 11,
	showSign = true,
	precision = 0,
	showPct = true,
	showAmount = true,
	align = "right",
	style,
	"data-testid": testId,
}: DeltaDisplayProps) {
	const preset = CONVENTION_COLORS[convention];
	const positive = positiveColor ?? preset.positive;
	const negative = negativeColor ?? preset.negative;
	const neutral = neutralColor ?? preset.neutral;

	const color = pct > 0 ? positive : pct < 0 ? negative : neutral;

	const sign = showSign && pct > 0 ? "+" : "";
	const pctText = `${sign}${pct.toFixed(precision)}%`;

	const pctSize = primary === "pct" ? primarySize : secondarySize;
	const amtSize = primary === "pct" ? secondarySize : primarySize;

	const hasAmount = showAmount && amount != null;

	const containerStyle: CSSProperties =
		layout === "horizontal"
			? { display: "inline-flex", alignItems: "flex-end", gap: 6, ...style }
			: {
					display: "inline-flex",
					flexDirection: "column",
					alignItems: ALIGN_ITEMS[align],
					...style,
				};

	return (
		<span style={containerStyle} data-testid={testId}>
			{showPct && (
				<Text
					strong={primary === "pct"}
					style={{ fontSize: pctSize, lineHeight: 1.2, color }}
					data-testid={testId ? `${testId}-pct` : undefined}
				>
					{pctText}
				</Text>
			)}
			{hasAmount && (
				<Text
					strong={primary === "amount"}
					type="secondary"
					style={{
						fontSize: amtSize,
						lineHeight: layout === "horizontal" ? 1.4 : 1.3,
					}}
					data-testid={testId ? `${testId}-amount` : undefined}
				>
					{amount}
				</Text>
			)}
		</span>
	);
}
