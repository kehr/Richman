import { Space, Tag } from "@/ui-kit/eat";

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

// colorForValue maps the canonical 3-way labels to antd Tag colors. This
// color is applied in both the steady-state and flip-state renders so a
// stable "bullish" position is still visually distinct from a stable
// "bearish" one; the flip state adds an arrow + strikethrough on top of
// the base color to draw the eye to the change.
function colorForValue(value: string): string {
	switch (value) {
		case "bullish":
			return "green";
		case "bearish":
			return "red";
		case "neutral":
			return "default";
		default:
			return "default";
	}
}

// DimensionBadge renders a single labelled badge. The Tag color is always
// driven by the current value so tri-color semantics survive across renders
// (reviewer called out that gating color on flipped-state removed product
// signal). When a non-null previous value differs from the current value
// we add a strikethrough old value + arrow to layer the flip affordance
// on top of the base color.
function DimensionBadge({ value }: { value: DimensionValue }) {
	const flipped = value.previous != null && value.previous !== value.current;
	const color = colorForValue(value.current);
	return (
		<Tag color={color} data-testid={`dim-${value.label.toLowerCase()}`}>
			<span style={{ marginRight: 4 }}>{value.label}:</span>
			{flipped ? (
				<>
					<span
						style={{ textDecoration: "line-through", opacity: 0.6 }}
						data-testid={`dim-${value.label.toLowerCase()}-prev`}
					>
						{value.previous}
					</span>
					<span style={{ margin: "0 4px" }}>→</span>
					<span data-testid={`dim-${value.label.toLowerCase()}-current`}>{value.current}</span>
				</>
			) : (
				<span data-testid={`dim-${value.label.toLowerCase()}-current`}>{value.current}</span>
			)}
		</Tag>
	);
}

// DimensionBadges renders the three-dimension strip shown directly under the
// card header. Consumers pass each dimension with an optional `previous`
// value to opt in to the flip animation.
export function DimensionBadges({ trend, position, catalyst }: DimensionBadgesProps) {
	return (
		<Space size="small" wrap>
			<DimensionBadge value={trend} />
			<DimensionBadge value={position} />
			<DimensionBadge value={catalyst} />
		</Space>
	);
}
