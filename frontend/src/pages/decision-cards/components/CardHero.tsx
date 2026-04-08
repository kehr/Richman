import { useMoney } from "@/domain/money/useMoney";
import { ChangeBadge, type DecisionCardDTO } from "@/features/decision-card";
import { QuestionCircleOutlined, Space, Tag, Tooltip, Typography } from "@/ui-kit/eat";
import { Link } from "react-router";

const { Text, Title } = Typography;

interface CardHeroProps {
	card: DecisionCardDTO;
}

// computeTargetGapPct renders the suggested reallocation magnitude as a
// percentage of the current allocated capital. This is NOT profit-and-loss:
// it expresses how far the model wants the user to move from their current
// position size to the recommended target. We display it as a neutral
// metric (no red/green coloring) so users do not confuse it with realized
// P&L. Real realized P&L will be wired in once the backend exposes mark
// price + cost basis on the decision_card DTO.
function computeTargetGapPct(card: DecisionCardDTO): number | null {
	if (card.targetPositionAmount != null && card.positionAmount != null && card.positionAmount > 0) {
		return ((card.targetPositionAmount - card.positionAmount) / card.positionAmount) * 100;
	}
	return null;
}

// CardHero renders the top hero block of the decision card detail page per
// PRD section 5: type tag + asset code + large asset name + cost + position
// summary + suggested reallocation magnitude on the left, and the change
// badge on the right. Current price and realized P&L are intentionally
// omitted until the backend DTO carries them; the file-level comment on
// computeTargetGapPct documents the placeholder semantics.
export function CardHero({ card }: CardHeroProps) {
	const money = useMoney();
	const positionText = money.format(card.positionRatio, card.positionAmount);
	const targetGapPct = computeTargetGapPct(card);
	const gapSign = targetGapPct != null && targetGapPct > 0 ? "+" : "";

	return (
		<div
			data-testid="card-hero"
			style={{
				display: "flex",
				justifyContent: "space-between",
				alignItems: "flex-start",
				gap: 16,
			}}
		>
			<Space direction="vertical" size={4} style={{ flex: 1 }}>
				<Space wrap>
					<Tag>{card.assetType}</Tag>
					<Text type="secondary">{card.assetCode}</Text>
				</Space>
				<Title level={2} style={{ margin: 0, fontSize: 28 }}>
					{card.assetName}
				</Title>
				<Space size="middle" wrap>
					<Text type="secondary">成本: {card.costPrice.toFixed(2)}</Text>
					<Text type="secondary">仓位: {positionText}</Text>
				</Space>
				{targetGapPct != null && targetGapPct !== 0 && (
					<Text type="secondary" data-testid="card-hero-target-gap">
						目标偏离: {gapSign}
						{targetGapPct.toFixed(2)}%（建议调仓幅度，非盈亏）
					</Text>
				)}
			</Space>
			<Space size={4}>
				<ChangeBadge badgeState={card.badgeState} />
				<Tooltip title="查看变化徽章说明">
					<Link to="/help#badge" aria-label="变化徽章帮助" data-testid="card-hero-badge-help">
						<QuestionCircleOutlined style={{ color: "#8c8c8c" }} />
					</Link>
				</Tooltip>
			</Space>
		</div>
	);
}
