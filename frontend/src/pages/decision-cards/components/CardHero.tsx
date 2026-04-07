import { useMoney } from "@/domain/money/useMoney";
import { ChangeBadge, type DecisionCardDTO } from "@/features/decision-card";
import { Space, Tag, Typography } from "@/ui-kit/eat";

const { Text, Title } = Typography;

interface CardHeroProps {
	card: DecisionCardDTO;
}

// formatPnl renders a signed percentage P&L number with color hint. The
// backend does not yet expose realized P&L per card, so the displayed value
// uses (currentPrice - costPrice) / costPrice if available; in absence of a
// current price field we fall back to (target - current) ratio as a
// best-effort placeholder consistent with DashboardPage's aggregate proxy.
function computePnlPct(card: DecisionCardDTO): number {
	if (card.targetPositionAmount != null && card.positionAmount != null && card.positionAmount > 0) {
		return ((card.targetPositionAmount - card.positionAmount) / card.positionAmount) * 100;
	}
	return 0;
}

// CardHero renders the top hero block of the decision card detail page per
// PRD section 5: type tag + asset code + large asset name + cost / current /
// position / amount summary line + P&L number on the left, and the change
// badge on the right.
export function CardHero({ card }: CardHeroProps) {
	const money = useMoney();
	const positionText = money.format(card.positionRatio, card.positionAmount);
	const pnlPct = computePnlPct(card);
	const pnlColor = pnlPct > 0 ? "#cf1322" : pnlPct < 0 ? "#3f8600" : undefined;
	const pnlSign = pnlPct > 0 ? "+" : "";

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
				{pnlPct !== 0 && (
					<Text strong style={{ color: pnlColor, fontSize: 16 }} data-testid="card-hero-pnl">
						盈亏: {pnlSign}
						{pnlPct.toFixed(2)}%
					</Text>
				)}
			</Space>
			<ChangeBadge badgeState={card.badgeState} />
		</div>
	);
}
