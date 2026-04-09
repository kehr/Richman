import type { DecisionCardDTO } from "@/features/decision-card";
import {
	Badge,
	Card,
	Col,
	QuestionCircleOutlined,
	Row,
	Space,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
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

type BadgeStatus = "success" | "error" | "warning" | "processing" | "default";

// directionStatus maps the canonical direction values to an antd Badge status.
// Enumerable status values should use Badge status dots, not colored Tags.
function directionStatus(value: string): BadgeStatus {
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

// dimensionBorderColor is used only for the flipped-card border highlight.
function dimensionBorderColor(value: string): string {
	switch (value) {
		case "bullish":
		case "upward":
			return "#52c41a";
		case "bearish":
		case "downward":
			return "#f5222d";
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
	const { t } = useTranslation("app");
	const dirLabel = (v: string) => t(`decisionCard.dimension.direction.${v}`, { defaultValue: v });
	const status = directionStatus(view.current);
	const containerStyle: CSSProperties = flipped
		? {
				borderColor: dimensionBorderColor(view.current),
				borderWidth: 2,
				background: "#fffbe6",
			}
		: {};
	const badgeText =
		view.previous && view.previous !== view.current
			? `${dirLabel(view.previous)} → ${dirLabel(view.current)}`
			: dirLabel(view.current);
	return (
		<Card
			size="small"
			style={containerStyle}
			data-testid={`dimension-card-${view.key}`}
			title={
				<div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
					<Text strong>{view.label}</Text>
					<Badge
						status={status}
						text={badgeText}
						data-testid={
							view.previous && view.previous !== view.current
								? `dimension-flip-${view.key}`
								: undefined
						}
					/>
				</div>
			}
		>
			<Space direction="vertical" size={4} style={{ width: "100%" }}>
				<Text type="secondary">
					{t("decisionCard.dimension.weight")}: {view.weight.toFixed(0)}
					{formatWeightDelta(view.weight, view.prevWeight)}
				</Text>
				<Paragraph style={{ margin: 0 }}>
					{view.summary || t("decisionCard.dimension.noSummary")}
				</Paragraph>
				{flipped && (
					<Text type="warning" strong data-testid={`dimension-flip-note-${view.key}`}>
						{t("decisionCard.dimension.flipNote")}
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
	const { t } = useTranslation("app");
	const views: DimensionView[] = [
		{
			key: "trend",
			label: t("decisionCard.dimension.label.trend"),
			current: card.trendDirection,
			previous: prevCard?.trendDirection,
			weight: card.weightTrend,
			prevWeight: prevCard?.weightTrend,
			summary: card.trendSummary,
		},
		{
			key: "position",
			label: t("decisionCard.dimension.label.position"),
			current: card.positionDirection,
			previous: prevCard?.positionDirection,
			weight: card.weightPosition,
			prevWeight: prevCard?.weightPosition,
			summary: card.positionSummary,
		},
		{
			key: "catalyst",
			label: t("decisionCard.dimension.label.catalyst"),
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
						{t("decisionCard.dimension.title")}
					</Title>
					<Tooltip title={t("decisionCard.dimension.title")}>
						<Link
							to="/help#dimensions"
							aria-label={t("decisionCard.dimension.title")}
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
