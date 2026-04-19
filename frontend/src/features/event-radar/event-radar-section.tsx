import { Alert, Button, Card, Skeleton, Space, Tag, Typography, theme } from "@/ui-kit/eat";
import type React from "react";
import { useTranslation } from "react-i18next";
import type { EventDto, EventRadarDto } from "./types";

const { Text, Title } = Typography;
const { useToken } = theme;

interface EventRowProps {
	event: EventDto;
}

// Only allow https links to render as anchors. Defends against javascript: /
// data: schemes if a future upstream payload ever leaks them.
function isSafeHttpsUrl(value: string | null | undefined): value is string {
	return typeof value === "string" && value.startsWith("https://");
}

function EventRow({ event }: EventRowProps) {
	const { t } = useTranslation("market");
	const { token } = useToken();

	const impactTagColor =
		event.impact === "high" ? "error" : event.impact === "medium" ? "warning" : "default";

	const goldTagColor =
		event.goldDirection === "bullish"
			? "success"
			: event.goldDirection === "bearish"
				? "error"
				: "default";

	// Source tag color distinguishes data provenance at a glance:
	// - FRED / Federal Reserve: blue (official statistical / policy source)
	// - Polymarket: purple (prediction-market-derived probability)
	// - anything else: default grey
	const sourceTagColor =
		event.sourceName === "Polymarket"
			? "purple"
			: event.sourceName === "FRED" || event.sourceName === "Federal Reserve"
				? "blue"
				: "default";

	const hasProbability = typeof event.probability === "number";
	const hasChange24h = typeof event.probabilityChange24h === "number";
	const change24hSign = hasChange24h && (event.probabilityChange24h as number) > 0 ? "+" : "";

	const isClickable = isSafeHttpsUrl(event.sourceUrl);

	const baseStyle: React.CSSProperties = {
		display: "flex",
		alignItems: "center",
		gap: 12,
		padding: "10px 8px",
		borderBottom: `1px solid ${token.colorBorderSecondary}`,
		flexWrap: "wrap",
		borderRadius: 4,
		textDecoration: "none",
		color: "inherit",
		cursor: isClickable ? "pointer" : "default",
		transition: "background-color 0.15s",
	};

	const content = (
		<>
			{/* Date */}
			<Text
				style={{
					fontSize: 12,
					color: token.colorTextTertiary,
					minWidth: 60,
					flexShrink: 0,
				}}
			>
				{event.date}
			</Text>

			{/* Title */}
			<Text
				style={{
					fontSize: 13,
					color: token.colorText,
					flex: 1,
					minWidth: 120,
				}}
			>
				{event.title}
			</Text>

			<Space size={6} wrap>
				{/* Source provenance — visible so users can weigh predictive
				    Polymarket entries differently from official FRED releases. */}
				{event.sourceName && (
					<Tag color={sourceTagColor} style={{ fontSize: 11, margin: 0 }}>
						{event.sourceName}
					</Tag>
				)}

				{/* Impact level */}
				<Tag color={impactTagColor} style={{ fontSize: 11, margin: 0 }}>
					{t(`overview.eventRadar.impactLevel.${event.impact}`)}
				</Tag>

				{/* Gold direction */}
				{event.goldDirection && (
					<Tag color={goldTagColor} style={{ fontSize: 11, margin: 0 }}>
						{t(`overview.eventRadar.goldDirection.${event.goldDirection}`)}
					</Tag>
				)}

				{/* Polymarket probability */}
				{hasProbability && (
					<Text style={{ fontSize: 12, color: token.colorTextSecondary }}>
						{t("overview.eventRadar.probability")}{" "}
						{((event.probability as number) * 100).toFixed(0)}%
					</Text>
				)}

				{/* 24h change */}
				{hasChange24h && (
					<Text
						style={{
							fontSize: 12,
							color:
								(event.probabilityChange24h as number) > 0
									? token.colorSuccess
									: (event.probabilityChange24h as number) < 0
										? token.colorError
										: token.colorTextTertiary,
						}}
					>
						{t("overview.eventRadar.change24h")} {change24hSign}
						{((event.probabilityChange24h as number) * 100).toFixed(1)}pp
					</Text>
				)}
			</Space>
		</>
	);

	if (isClickable) {
		const sourceLabel = event.sourceName
			? t("overview.eventRadar.sourceLabel", { name: event.sourceName })
			: t("overview.eventRadar.openSourceTooltip");
		return (
			<a
				href={event.sourceUrl as string}
				target="_blank"
				rel="noopener noreferrer"
				title={sourceLabel}
				style={baseStyle}
				onMouseEnter={(e) => {
					e.currentTarget.style.backgroundColor = token.colorBgTextHover;
				}}
				onMouseLeave={(e) => {
					e.currentTarget.style.backgroundColor = "transparent";
				}}
			>
				{content}
			</a>
		);
	}

	return <div style={baseStyle}>{content}</div>;
}

interface EventRadarSectionProps {
	data: EventRadarDto | undefined;
	isLoading: boolean;
	isError: boolean;
	onRetry: () => void;
}

// EventRadarSection renders the upcoming macro event list.
// On error, shows a retry prompt (G3.9) rather than hiding the section entirely.
// Each row links to its upstream source (FRED release / Polymarket event) when
// a sourceUrl is present.
export function EventRadarSection({ data, isLoading, isError, onRetry }: EventRadarSectionProps) {
	const { t } = useTranslation("market");
	const { token } = useToken();

	return (
		<Card styles={{ body: { padding: "16px" } }} style={{ marginTop: 24 }}>
			<Title level={5} style={{ marginBottom: 4, marginTop: 0, fontSize: 14, fontWeight: 600 }}>
				{t("overview.eventRadar.title")}
			</Title>
			<Text
				style={{ fontSize: 12, color: token.colorTextTertiary, display: "block", marginBottom: 12 }}
			>
				{t("overview.eventRadar.subtitle")}
			</Text>

			{isLoading && <Skeleton active paragraph={{ rows: 4 }} />}

			{isError && (
				<Alert
					type="warning"
					message={t("overview.eventRadar.loadError")}
					action={
						<Button size="small" onClick={onRetry}>
							{t("overview.eventRadar.retry")}
						</Button>
					}
					showIcon
				/>
			)}

			{!isLoading &&
				!isError &&
				data &&
				(data.events.length === 0 ? (
					<Text style={{ fontSize: 13, color: token.colorTextTertiary }}>
						{t("overview.eventRadar.noEvents")}
					</Text>
				) : (
					<div>
						{data.events.map((event) => {
							// Stable key strategy: prefer FRED release_id, fall back to the
							// source URL for Polymarket (slug carries identity), then to the
							// title for items without any source metadata.
							const key =
								event.releaseId !== null && event.releaseId !== undefined
									? `fred-${event.releaseId}-${event.date}`
									: `${event.sourceName ?? "unknown"}-${event.sourceUrl ?? event.title}-${event.date}`;
							return <EventRow key={key} event={event} />;
						})}
					</div>
				))}
		</Card>
	);
}
