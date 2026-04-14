// ScoreSparkline renders a lightweight SVG line chart for the 90-day score trend.
// This is a deliberate lightweight alternative to echarts to avoid adding a
// heavy dependency for a small inline preview chart.

interface ScoreSparklineProps {
	// Oldest-first list of composite scores (0-100) emitted by richson.
	data: number[];
	width?: number;
	height?: number;
}

export function ScoreSparkline({ data, width = 240, height = 48 }: ScoreSparklineProps) {
	if (data.length < 2) return null;

	const minScore = Math.min(...data);
	const maxScore = Math.max(...data);
	const range = maxScore - minScore || 1;

	// Normalize score to SVG y coordinate (top = high score).
	const toY = (score: number) => height - ((score - minScore) / range) * (height - 8) - 4;
	const toX = (index: number) => (index / (data.length - 1)) * width;

	const points = data.map((score, i) => `${toX(i).toFixed(1)},${toY(score).toFixed(1)}`);
	const polyline = points.join(" ");

	// Color the line by last score level: green >= 60, orange >= 40, red < 40.
	const lastScore = data[data.length - 1];
	const lineColor =
		lastScore >= 60
			? "var(--ant-color-success)"
			: lastScore >= 40
				? "#fa8c16"
				: "var(--ant-color-error)";

	return (
		<svg
			width={width}
			height={height}
			viewBox={`0 0 ${width} ${height}`}
			style={{ display: "block", overflow: "visible" }}
			aria-hidden="true"
		>
			<polyline
				points={polyline}
				fill="none"
				stroke={lineColor}
				strokeWidth={1.5}
				strokeLinejoin="round"
				strokeLinecap="round"
			/>
		</svg>
	);
}
