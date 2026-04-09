import { Badge, Space } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// DimensionValue is the canonical 3-way label for one of the three decision
// card dimensions (trend / position / catalyst). The backend contract is
// that direction values are English canonical strings ("bullish" | "bearish"
// | "neutral") produced by analysis/recommendation.go; new labels added in
// future backend versions fall through to the neutral style rather than
// crash the UI.
export interface DimensionValue {
	label: string;
	current: string;
	previous?: string | null;
}

interface DimensionBadgesProps {
	trend: DimensionValue;
	position: DimensionValue;
	catalyst: DimensionValue;
}

type BadgeStatus = "success" | "error" | "warning" | "processing" | "default";

// statusForValue maps canonical direction values to antd Badge status dots.
function statusForValue(value: string): BadgeStatus {
	switch (value) {
		case "bullish":
		case "upward":
			return "success";
		case "bearish":
		case "downward":
			return "error";
		default:
			return "default";
	}
}

// DimensionBadge renders a single labelled Badge status dot. When a previous
// value differs from current we layer the flip affordance as strikethrough
// old value + arrow inside the Badge text.
function DimensionBadge({ value }: { value: DimensionValue }) {
	const { t } = useTranslation("app");
	const flipped = value.previous != null && value.previous !== value.current;
	const status = statusForValue(value.current);
	const dirKey = (v: string) => t(`decisionCard.dimension.direction.${v}`, { defaultValue: v });

	const text = flipped ? (
		<span data-testid={`dim-${value.label.toLowerCase()}`}>
			{value.label}:{" "}
			<span
				style={{ textDecoration: "line-through", opacity: 0.6 }}
				data-testid={`dim-${value.label.toLowerCase()}-prev`}
			>
				{dirKey(value.previous ?? "")}
			</span>
			{" → "}
			<span data-testid={`dim-${value.label.toLowerCase()}-current`}>{dirKey(value.current)}</span>
		</span>
	) : (
		<span data-testid={`dim-${value.label.toLowerCase()}`}>
			{value.label}:{" "}
			<span data-testid={`dim-${value.label.toLowerCase()}-current`}>{dirKey(value.current)}</span>
		</span>
	);

	return <Badge status={status} text={text} />;
}

// DimensionBadges renders the three-dimension strip shown directly under the
// card header. Consumers pass each dimension with an optional `previous`
// value to opt in to the flip animation.
export function DimensionBadges({ trend, position, catalyst }: DimensionBadgesProps) {
	return (
		<Space size="middle" wrap>
			<DimensionBadge value={trend} />
			<DimensionBadge value={position} />
			<DimensionBadge value={catalyst} />
		</Space>
	);
}
