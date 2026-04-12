import type { DecisionCardDTO } from "@/features/decision-card";
import {
	type AssetQuoteDTO,
	MarketQuoteChart,
	type PriceLine,
	type TimeMarker,
	assetQuoteQueryKey,
	useAssetQuote,
} from "@/features/market-quote";
import { Alert, Button, Card, ReloadOutlined, Skeleton, Space, Typography } from "@/ui-kit/eat";
import { useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

const { Text, Title } = Typography;

interface MarketContextPanelProps {
	card: DecisionCardDTO;
}

interface OverlayLabels {
	cost: string;
	stopLoss: string;
	trigger: string;
	analysis: string;
}

// buildPriceLines extracts horizontal overlay lines from the decision card DTO.
function buildPriceLines(card: DecisionCardDTO, labels: OverlayLabels): PriceLine[] {
	const lines: PriceLine[] = [];

	// Cost price line (always present).
	if (card.costPrice > 0) {
		lines.push({
			price: card.costPrice,
			color: "#8c8c8c",
			lineStyle: "solid",
			label: labels.cost,
		});
	}

	// Stop-loss line (optional).
	const stopLoss = card.recommendation.execution.stopLoss;
	if (stopLoss != null && stopLoss > 0) {
		lines.push({
			price: stopLoss,
			color: "#ff4d4f",
			lineStyle: "dashed",
			label: labels.stopLoss,
		});
	}

	// First price trigger from execution steps (optional).
	const steps = card.recommendation.execution.steps ?? [];
	for (const step of steps) {
		if (step.triggerType === "price" && step.triggerPayload?.priceValue != null) {
			lines.push({
				price: step.triggerPayload.priceValue,
				color: "#fa8c16",
				lineStyle: "dashed",
				label: labels.trigger,
			});
			break;
		}
	}

	return lines;
}

// buildTimeMarkers creates vertical markers for the chart. Currently only the
// analysis timestamp is rendered, but the array pattern supports future markers.
function buildTimeMarkers(
	card: DecisionCardDTO,
	history: AssetQuoteDTO["history"],
	labels: OverlayLabels,
): TimeMarker[] {
	if (!card.analyzedAt || history.length === 0) return [];

	const analysisDate = card.analyzedAt.slice(0, 10);
	const firstDate = history[0].date.slice(0, 10);
	const lastDate = history[history.length - 1].date.slice(0, 10);

	// Only show marker if analysis date falls within chart range.
	if (analysisDate < firstDate || analysisDate > lastDate) return [];

	return [
		{
			time: card.analyzedAt,
			label: labels.analysis,
			color: "#1677ff",
		},
	];
}

// formatTime renders a short HH:MM timestamp from an ISO date string.
function formatTime(iso: string): string {
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return "--";
	return d.toLocaleTimeString("en-GB", {
		hour: "2-digit",
		minute: "2-digit",
		hour12: false,
		timeZone: "Asia/Shanghai",
	});
}

// formatPctChange formats a percentage change with sign and 2 decimal places.
function formatPctChange(pct: number): string {
	const sign = pct > 0 ? "+" : "";
	return `${sign}${pct.toFixed(2)}%`;
}

// pctChangeColor returns a CSS color for positive/negative/zero changes.
function pctChangeColor(pct: number): string {
	if (pct > 0) return "#cf1322";
	if (pct < 0) return "#389e0d";
	return "#8c8c8c";
}

// MarketContextPanel is the composition layer between the market-quote feature
// module and the decision card detail page. It extracts overlay configuration
// from the card DTO, delegates chart rendering to MarketQuoteChart, and handles
// loading / unavailable / error states.
export function MarketContextPanel({ card }: MarketContextPanelProps) {
	const { t } = useTranslation("app");
	const queryClient = useQueryClient();
	const { data, isLoading, isError, isFetching } = useAssetQuote(card.assetType, card.assetCode);

	const overlayLabels: OverlayLabels = useMemo(
		() => ({
			cost: t("decisionCard.marketContext.overlay.cost"),
			stopLoss: t("decisionCard.marketContext.overlay.stopLoss"),
			trigger: t("decisionCard.marketContext.overlay.trigger"),
			analysis: t("decisionCard.marketContext.overlay.analysis"),
		}),
		[t],
	);

	const priceLines = useMemo(() => buildPriceLines(card, overlayLabels), [card, overlayLabels]);

	const timeMarkers = useMemo(
		() => buildTimeMarkers(card, data?.history ?? [], overlayLabels),
		[card, data?.history, overlayLabels],
	);

	const handleRefresh = () => {
		queryClient.invalidateQueries({
			queryKey: assetQuoteQueryKey(card.assetType, card.assetCode),
		});
	};

	// Loading state.
	if (isLoading) {
		return (
			<Card size="small">
				<Skeleton active paragraph={{ rows: 4 }} />
			</Card>
		);
	}

	// Error state.
	if (isError) {
		return (
			<Card size="small">
				<Alert
					type="error"
					showIcon
					message={t("decisionCard.marketContext.error.title")}
					action={
						<Button size="small" onClick={handleRefresh}>
							{t("decisionCard.marketContext.error.retry")}
						</Button>
					}
				/>
			</Card>
		);
	}

	// Unavailable state (e.g., A-share with no live data source).
	if (data?.source === "unavailable") {
		return (
			<Card size="small">
				<Space direction="vertical" size={4}>
					<Text type="secondary">
						{t("decisionCard.marketContext.unavailable.title", {
							assetType: t(`portfolio.assetTypes.${card.assetType}`, {
								defaultValue: card.assetType,
							}),
						})}
					</Text>
					<Text type="secondary">
						{t("decisionCard.marketContext.unavailable.analysisPrice")}:{" "}
						{card.currentPrice.toFixed(2)}
					</Text>
					<Text type="secondary">
						{t("decisionCard.marketContext.unavailable.analysisTime", {
							time: formatTime(card.analyzedAt),
						})}
					</Text>
				</Space>
			</Card>
		);
	}

	// Normal state with live data.
	const current = data?.current;
	const history = data?.history ?? [];

	return (
		<Card
			size="small"
			title={
				<Space>
					<Title level={5} style={{ margin: 0 }}>
						{t("decisionCard.marketContext.title")}
					</Title>
					{current && (
						<Text strong style={{ fontSize: 16 }}>
							{current.price.toFixed(2)}
						</Text>
					)}
					{current && (
						<Text style={{ color: pctChangeColor(current.changePct), fontSize: 13 }}>
							{formatPctChange(current.changePct)}
						</Text>
					)}
				</Space>
			}
			extra={
				<Space size={8}>
					{data?.fetchedAt && (
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("decisionCard.marketContext.updatedAt", {
								time: formatTime(data.fetchedAt),
							})}
						</Text>
					)}
					<Button
						size="small"
						icon={<ReloadOutlined spin={isFetching} />}
						onClick={handleRefresh}
						loading={isFetching}
					>
						{isFetching
							? t("decisionCard.marketContext.refreshing")
							: t("decisionCard.marketContext.refresh")}
					</Button>
				</Space>
			}
		>
			{history.length > 0 ? (
				<MarketQuoteChart history={history} priceLines={priceLines} timeMarkers={timeMarkers} />
			) : (
				<Text type="secondary">{t("decisionCard.marketContext.error.title")}</Text>
			)}

			{current && (
				<Space size="large" style={{ marginTop: 8 }}>
					{card.costPrice > 0 && (
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("decisionCard.marketContext.vsCost")}:{" "}
							<Text
								style={{
									color: pctChangeColor(((current.price - card.costPrice) / card.costPrice) * 100),
									fontSize: 12,
								}}
							>
								{formatPctChange(((current.price - card.costPrice) / card.costPrice) * 100)}
							</Text>
						</Text>
					)}
					{card.currentPrice > 0 && (
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("decisionCard.marketContext.vsAnalysis")}:{" "}
							<Text
								style={{
									color: pctChangeColor(
										((current.price - card.currentPrice) / card.currentPrice) * 100,
									),
									fontSize: 12,
								}}
							>
								{formatPctChange(((current.price - card.currentPrice) / card.currentPrice) * 100)}
							</Text>
						</Text>
					)}
				</Space>
			)}
		</Card>
	);
}
