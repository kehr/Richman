import { useMoney } from "@/domain/money/useMoney";
import { Card, Divider, Space, Tag, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import type { DecisionCardDTO } from "../types";
import { ChangeBadge } from "./ChangeBadge";
import { DimensionBadges } from "./DimensionBadges";
import { ExecutionPlanStrip } from "./ExecutionPlanStrip";
import { SourcePill } from "./SourcePill";

const { Text, Title } = Typography;

interface DecisionCardSummaryProps {
	card: DecisionCardDTO;
	// previousCard is an optional neighbour that, when passed, enables the
	// dimension flip effect. Callers that only have the latest card (e.g. the
	// Dashboard list) can omit this prop and the badges will render in their
	// neutral state.
	previousCard?: DecisionCardDTO | null;
	onClick?: (card: DecisionCardDTO) => void;
	onShowFullPlan?: (card: DecisionCardDTO) => void;
}

// DecisionCardSummary composes the four sub-components into the full card
// shape defined in PRD §3.2. Layout from top to bottom:
//
//   1. Header: asset name / code / type tag / change badge
//   2. Cost + position row + market value (formatted via useMoney)
//   3. Dimension badges (trend / position / catalyst)
//   4. Recommendation label + execution plan strip
//   5. Today's highlights paragraph
//   6. Confidence + "查看完整推理" footer
//
// The outer Card is `hoverable` when an `onClick` handler is provided so
// the Dashboard list gets the lift effect while the detail page (which
// embeds the same component without navigation) stays static. The whole
// card acts as a single interactive surface: an `aria-label` exposes the
// asset name + "查看完整推理" to screen readers and keyboard-Enter works
// through the native Card focus behaviour.
export function DecisionCardSummary({
	card,
	previousCard,
	onClick,
	onShowFullPlan,
}: DecisionCardSummaryProps) {
	const { t } = useTranslation("app");
	const money = useMoney();
	const positionText = money.format(card.positionRatio, card.positionAmount);
	const marketValueText = money.formatAmountOnly(card.positionAmount);
	const interactive = Boolean(onClick);

	const handleKeyDown = interactive
		? (event: React.KeyboardEvent<HTMLDivElement>) => {
				if (event.key === "Enter" || event.key === " ") {
					event.preventDefault();
					onClick?.(card);
				}
			}
		: undefined;

	return (
		<Card
			hoverable={interactive}
			onClick={onClick ? () => onClick(card) : undefined}
			onKeyDown={handleKeyDown}
			role={interactive ? "button" : undefined}
			tabIndex={interactive ? 0 : undefined}
			aria-label={
				interactive
					? `${card.assetName} ${card.assetCode} ${t("decisionCard.viewFullReasoning")}`
					: undefined
			}
			style={{ height: "100%" }}
			styles={{ body: { height: "100%", display: "flex", flexDirection: "column" } }}
			data-testid={`decision-card-${card.cardId}`}
		>
			<div
				style={{
					display: "flex",
					justifyContent: "space-between",
					alignItems: "flex-start",
					gap: 12,
				}}
			>
				<Space direction="vertical" size={2} style={{ flex: 1 }}>
					<Space wrap>
						<Tag>{card.assetType}</Tag>
						<Text strong style={{ fontSize: 16 }}>
							{card.assetName}
						</Text>
						<Text type="secondary">{card.assetCode}</Text>
					</Space>
					<Space size="middle" wrap>
						<Text type="secondary">
							{t("decisionCard.costLabel")}: {card.costPrice.toFixed(2)}
						</Text>
						<Text type="secondary">
							{t("decisionCard.positionLabel")}: {positionText}
						</Text>
						{marketValueText && (
							<Text type="secondary" data-testid="card-market-value">
								{t("decisionCard.marketValueLabel")}: {marketValueText}
							</Text>
						)}
					</Space>
				</Space>
				<Space direction="vertical" size={4} align="end">
					<SourcePill source={card.synthesisSource} provider={card.providerUsed} />
					<ChangeBadge badgeState={card.badgeState} />
				</Space>
			</div>

			<div style={{ marginTop: 12 }}>
				<DimensionBadges
					trend={{
						label: t("decisionCard.dimension.badgeLabel.trend"),
						current: card.trendDirection,
						previous: previousCard?.trendDirection,
					}}
					position={{
						label: t("decisionCard.dimension.badgeLabel.position"),
						current: card.positionDirection,
						previous: previousCard?.positionDirection,
					}}
					catalyst={{
						label: t("decisionCard.dimension.badgeLabel.catalyst"),
						current: card.catalystDirection,
						previous: previousCard?.catalystDirection,
					}}
				/>
			</div>

			<div
				style={{
					marginTop: 12,
					padding: 12,
					background: "#fafafa",
					borderRadius: 8,
				}}
				data-testid="card-recommendation-box"
			>
				<Title level={5} style={{ margin: 0, marginBottom: 8 }}>
					{t(`decisionCard.recommendation.${card.recommendation.action}`)}
				</Title>
				<ExecutionPlanStrip
					execution={card.recommendation.execution}
					onShowAll={onShowFullPlan ? () => onShowFullPlan(card) : undefined}
				/>
			</div>

			{card.todayHighlights && (
				<Text type="secondary" style={{ display: "block", marginTop: 8, fontSize: 12 }}>
					{card.todayHighlights}
				</Text>
			)}

			<Divider style={{ margin: "12px 0", marginTop: "auto" }} />

			<div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
				<Space size={4} align="baseline">
					<Text strong style={{ fontSize: 18 }} data-testid="card-confidence">
						{Math.round(card.confidence * 100)}%
					</Text>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t("decisionCard.confidenceLabel")}
					</Text>
				</Space>
				<Text type="secondary">{t("decisionCard.viewFullReasoning")}</Text>
			</div>
		</Card>
	);
}
