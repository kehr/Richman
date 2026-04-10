import { AssetTypeTag } from "@/components/AssetTypeTag";
import { useMoney } from "@/domain/money/useMoney";
import { Card, Divider, Space, Typography } from "@/ui-kit/eat";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { DecisionCardDTO, HoldingAnalysisStatus } from "../types";
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
	// analysisStatus drives in-progress visual state on the card:
	// "running" shows a blue border and updating badge; "done" triggers a
	// 2-second green flash before returning to the default appearance.
	analysisStatus?: HoldingAnalysisStatus;
	// analysisProgress is a 0-1 value rendered as a thin progress bar at the
	// bottom of the card when analysisStatus is "running".
	analysisProgress?: number;
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
	analysisStatus,
	analysisProgress,
}: DecisionCardSummaryProps) {
	const { t } = useTranslation("app");
	const money = useMoney();
	const positionText = money.format(card.positionRatio, card.positionAmount);
	const marketValueText = money.formatAmountOnly(card.positionAmount);
	const interactive = Boolean(onClick);

	const [justUpdated, setJustUpdated] = useState(false);
	const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	useEffect(() => {
		if (analysisStatus === "done") {
			setJustUpdated(true);
			timeoutRef.current = setTimeout(() => setJustUpdated(false), 2000);
		}
		return () => {
			if (timeoutRef.current !== null) {
				clearTimeout(timeoutRef.current);
			}
		};
	}, [analysisStatus]);

	// When running, the card is wrapped in .card-glow-wrapper (global.css) which
	// provides the spinning rainbow border; the Card itself gets border:none so
	// only the gradient wrapper shows. When just-updated (1.5 s flash on
	// completion) a subtle green ring replaces the default border briefly.
	// box-shadow is used instead of borderColor so the ring does not affect
	// layout, and the body background is left untouched to avoid a heavy flash.
	const isRunning = analysisStatus === "running";
	// overflow:hidden is always set so the ant-card-body white background never
	// bleeds past border-radius — this fixes both the shimmer-border corner clip
	// and the change-anchor highlight box-shadow corner clip.
	const borderStyle: React.CSSProperties = isRunning
		? { border: "none", borderRadius: 6, overflow: "hidden" }
		: justUpdated
			? { boxShadow: "0 0 0 2px rgba(82, 196, 26, 0.45)", overflow: "hidden" }
			: { overflow: "hidden" };

	const bodyBg = undefined;

	const handleKeyDown = interactive
		? (event: React.KeyboardEvent<HTMLDivElement>) => {
				if (event.key === "Enter" || event.key === " ") {
					event.preventDefault();
					onClick?.(card);
				}
			}
		: undefined;

	const cardNode = (
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
			style={{ height: "100%", position: "relative", ...borderStyle }}
			styles={{
				body: { height: "100%", display: "flex", flexDirection: "column", background: bodyBg },
			}}
			data-testid={`decision-card-${card.cardId}`}
		>
			{isRunning && (
				<span
					style={{
						position: "absolute",
						top: 6,
						right: 8,
						background: "#1677ff",
						color: "#fff",
						fontSize: 10,
						borderRadius: 4,
						padding: "1px 4px",
						zIndex: 1,
					}}
				>
					{t("analysisProgress.updating")}
				</span>
			)}

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
						<AssetTypeTag assetType={card.assetType} />
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
					positionAmountCny={card.positionAmount}
					positionRatioPct={card.positionRatio}
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
						{Math.round(card.confidence)}%
					</Text>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t("decisionCard.confidenceLabel")}
					</Text>
				</Space>
				<Text type="secondary">{t("decisionCard.viewFullReasoning")}</Text>
			</div>

			{isRunning && analysisProgress !== undefined && (
				<div
					style={{
						height: 2,
						background: "#f0f0f0",
						margin: "8px -24px -24px",
					}}
				>
					<div
						style={{
							height: "100%",
							background: "#1677ff",
							width: `${Math.round((analysisProgress ?? 0) * 100)}%`,
							transition: "width 0.5s ease",
						}}
					/>
				</div>
			)}
		</Card>
	);

	if (isRunning) {
		return (
			<div className="card-glow-wrapper" style={{ height: "100%" }}>
				{cardNode}
			</div>
		);
	}
	return cardNode;
}
