import { Space, Tag } from "@/ui-kit/eat";

// DimensionValue is the canonical 3-way label for one of the three decision
// card dimensions (trend / position / catalyst). The backend sends these as
// free strings (e.g. "bullish", "bearish", "neutral") so we do not restrict
// the prop type — the component just compares current vs previous for the
// flip effect and renders the raw value.
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

// colorForValue maps the canonical 3-way labels to antd Tag colors. Unknown
// values fall through to the neutral default so forward-compatible labels
// from future backend versions do not crash the UI.
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

// DimensionBadge renders a single labelled badge with optional flip styling.
// When a non-null previous value differs from the current value we render
// the previous value with a strikethrough followed by an arrow and the new
// value; the surrounding Tag is colored by the current value so the eye is
// drawn to the new state.
function DimensionBadge({ value }: { value: DimensionValue }) {
	const flipped = value.previous != null && value.previous !== value.current;
	const color = flipped ? colorForValue(value.current) : "default";
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
