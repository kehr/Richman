import type { BadgeState, DecisionCardDTO } from "@/features/decision-card";
import { QuestionCircleOutlined, Space, Tooltip, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

const { Text, Title } = Typography;

interface ConclusionBannerProps {
	card: DecisionCardDTO;
	prevCard?: DecisionCardDTO | null;
}

// BORDER_COLORS maps each badge state to the left-border accent color used
// in the conclusion banner. The neutral fall-through matches the default
// antd card border so a "none" state banner is still visually grounded.
const BORDER_COLORS: Record<BadgeState, string> = {
	none: "#d9d9d9",
	data_degraded: "#8c8c8c",
	first_analysis: "#262626",
	action_upgrade: "#52c41a",
	action_downgrade: "#f5222d",
	signal_flip: "#1677ff",
	plan_adjust: "#faad14",
	confidence_shift: "#722ed1",
};

// formatDelta renders the confidence delta as a signed percentage point.
function formatDelta(delta: number): string {
	if (delta === 0) return "0";
	const sign = delta > 0 ? "+" : "";
	return `${sign}${delta.toFixed(0)}`;
}

// ConclusionBanner renders the "今日建议" block at the top of the decision
// card detail. Left side shows the human-readable recommendation label and
// target position narrative, right side surfaces the model confidence plus
// delta compared to the previous card.
export function ConclusionBanner({ card, prevCard }: ConclusionBannerProps) {
	const { t } = useTranslation("app");
	const borderColor = BORDER_COLORS[card.badgeState];
	const confidencePct = Math.round(card.confidence * 100);
	const targetPct = card.targetPositionRatio;
	const currentPct = card.positionRatio;
	const gapPct = targetPct - currentPct;
	const gapSign = gapPct > 0 ? "+" : "";
	const narrative = t("decisionCard.hero.targetGap", { gap: `${gapSign}${gapPct.toFixed(0)}` });

	return (
		<div
			data-testid="conclusion-banner"
			style={{
				display: "flex",
				justifyContent: "space-between",
				alignItems: "center",
				gap: 16,
				padding: 16,
				background: "#fafafa",
				borderRadius: 8,
				borderLeft: `4px solid ${borderColor}`,
			}}
		>
			<Space direction="vertical" size={4} style={{ flex: 1 }}>
				<Text type="secondary">{t("decisionCard.conclusion.todayAdvice")}</Text>
				<Title level={3} style={{ margin: 0 }}>
					{t(`decisionCard.recommendation.${card.recommendation.action}`)}
				</Title>
				<Text>{narrative}</Text>
				{prevCard && (
					<Text type="secondary" data-testid="conclusion-prev">
						{t("decisionCard.conclusion.previousAdvice")}{" "}
						{t(`decisionCard.recommendation.${prevCard.recommendation.action}`)}
					</Text>
				)}
			</Space>
			<Space direction="vertical" size={0} align="end">
				<Space size={4}>
					<Text type="secondary">{t("decisionCard.conclusion.confidence")}</Text>
					<Tooltip title={t("decisionCard.conclusion.confidence")}>
						<Link
							to="/help#confidence"
							aria-label={t("decisionCard.conclusion.confidence")}
							data-testid="conclusion-confidence-help"
						>
							<QuestionCircleOutlined style={{ color: "#8c8c8c" }} />
						</Link>
					</Tooltip>
				</Space>
				<Title level={2} style={{ margin: 0 }} data-testid="conclusion-confidence">
					{confidencePct}%
				</Title>
				{card.confidenceDelta !== 0 && (
					<Text type="secondary" data-testid="conclusion-confidence-delta">
						Δ {formatDelta(card.confidenceDelta)}
					</Text>
				)}
			</Space>
		</div>
	);
}
