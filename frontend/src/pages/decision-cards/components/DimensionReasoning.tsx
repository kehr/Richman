import type { DecisionCardDTO } from "@/features/decision-card";
import {
	Card,
	Col,
	QuestionCircleOutlined,
	Row,
	Space,
	Tag,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { Link } from "react-router";

const { Text, Title, Paragraph } = Typography;

interface DimensionReasoningProps {
	card: DecisionCardDTO;
	prevCard?: DecisionCardDTO | null;
}

interface DimensionView {
	key: "trend" | "position" | "catalyst";
	label: string;
	current: string;
	previous?: string;
	weight: number;
	prevWeight?: number;
	summary: string;
}

// dimensionColor maps the canonical 3-way labels to a hex used for the
// dimension card border. Falls through to neutral gray when the backend
// returns an unknown direction so the UI never crashes on a typo.
function dimensionColor(value: string): string {
	switch (value) {
		case "bullish":
			return "#52c41a";
		case "bearish":
			return "#f5222d";
		case "neutral":
			return "#8c8c8c";
		default:
			return "#8c8c8c";
	}
}

// formatWeightDelta renders a small "(40 → 45)" trace when the dimension
// weight changed compared to the previous card. Returns an empty string when
// no previous card is known or the weight is unchanged so the consumer can
// safely concat without conditionals.
function formatWeightDelta(current: number, previous?: number): string {
	if (previous == null || previous === current) return "";
	return ` (${previous.toFixed(0)} → ${current.toFixed(0)})`;
}

// DimensionCard renders a single dimension panel. When `flipped` is true the
// border switches to the dimension color and a callout is appended below the
// summary explaining the flip is the main driver of the badge change.
function DimensionCard({ view, flipped }: { view: DimensionView; flipped: boolean }) {
	const color = dimensionColor(view.current);
	const containerStyle: CSSProperties = flipped
		? {
				borderColor: color,
				borderWidth: 2,
				background: "#fffbe6",
			}
		: {};
	return (
		<Card
			size="small"
			style={containerStyle}
			data-testid={`dimension-card-${view.key}`}
			title={
				<Space>
					<Text strong>{view.label}</Text>
					{view.previous && view.previous !== view.current && (
						<Tag color={color} data-testid={`dimension-flip-${view.key}`}>
							{view.previous} → {view.current}
						</Tag>
					)}
					{(!view.previous || view.previous === view.current) && (
						<Tag color={color}>{view.current}</Tag>
					)}
				</Space>
			}
		>
			<Space direction="vertical" size={4} style={{ width: "100%" }}>
				<Text type="secondary">
					权重: {view.weight.toFixed(0)}
					{formatWeightDelta(view.weight, view.prevWeight)}
				</Text>
				<Paragraph style={{ margin: 0 }}>{view.summary || "(暂无说明)"}</Paragraph>
				{flipped && (
					<Text type="warning" strong data-testid={`dimension-flip-note-${view.key}`}>
						提示：此维度发生翻转是本次建议升级的主要驱动因素
					</Text>
				)}
			</Space>
		</Card>
	);
}

// DimensionReasoning renders the three-dimension reasoning grid (trend /
// position / catalyst) per PRD section 5. Each card surfaces the current
// direction, weight micro-adjust trace, and the text conclusion produced by
// analysis. The dimension whose direction flipped vs the previous card gets
// a colored border and explainer callout.
export function DimensionReasoning({ card, prevCard }: DimensionReasoningProps) {
	const views: DimensionView[] = [
		{
			key: "trend",
			label: "趋势 Trend",
			current: card.trendDirection,
			previous: prevCard?.trendDirection,
			weight: card.weightTrend,
			prevWeight: prevCard?.weightTrend,
			summary: card.trendSummary,
		},
		{
			key: "position",
			label: "仓位 Position",
			current: card.positionDirection,
			previous: prevCard?.positionDirection,
			weight: card.weightPosition,
			prevWeight: prevCard?.weightPosition,
			summary: card.positionSummary,
		},
		{
			key: "catalyst",
			label: "催化 Catalyst",
			current: card.catalystDirection,
			previous: prevCard?.catalystDirection,
			weight: card.weightCatalyst,
			prevWeight: prevCard?.weightCatalyst,
			summary: card.catalystSummary,
		},
	];

	return (
		<Card
			title={
				<Space size={4}>
					<Title level={5} style={{ margin: 0 }}>
						三维度推理
					</Title>
					<Tooltip title="查看三维分析说明">
						<Link
							to="/help#dimensions"
							aria-label="三维分析帮助"
							data-testid="dimension-reasoning-help"
						>
							<QuestionCircleOutlined style={{ color: "#8c8c8c" }} />
						</Link>
					</Tooltip>
				</Space>
			}
			data-testid="dimension-reasoning"
		>
			<Row gutter={[12, 12]}>
				{views.map((v) => (
					<Col key={v.key} xs={24} md={8}>
						<DimensionCard view={v} flipped={Boolean(v.previous && v.previous !== v.current)} />
					</Col>
				))}
			</Row>
		</Card>
	);
}
