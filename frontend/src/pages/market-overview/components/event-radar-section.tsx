import type { EventDto, EventRadarDto } from "@/features/event-radar";
import { Alert, Button, Card, Skeleton, Space, Tag, Typography, theme } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text, Title } = Typography;
const { useToken } = theme;

interface EventRowProps {
	event: EventDto;
}

function EventRow({ event }: EventRowProps) {
	const { t } = useTranslation("market");
	const { token } = useToken();

	const impactTagColor =
		event.impactLevel === "high"
			? "error"
			: event.impactLevel === "medium"
				? "warning"
				: "default";

	const goldTagColor =
		event.goldDirection === "bullish"
			? "success"
			: event.goldDirection === "bearish"
				? "error"
				: "default";

	const change24hSign =
		event.polymarketChange24h !== null && event.polymarketChange24h > 0 ? "+" : "";

	return (
		<div
			style={{
				display: "flex",
				alignItems: "center",
				gap: 12,
				padding: "10px 0",
				borderBottom: `1px solid ${token.colorBorderSecondary}`,
				flexWrap: "wrap",
			}}
		>
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
				{/* Impact level */}
				<Tag color={impactTagColor} style={{ fontSize: 11, margin: 0 }}>
					{t(`overview.eventRadar.impactLevel.${event.impactLevel}`)}
				</Tag>

				{/* Gold direction */}
				{event.goldDirection && (
					<Tag color={goldTagColor} style={{ fontSize: 11, margin: 0 }}>
						{t(`overview.eventRadar.goldDirection.${event.goldDirection}`)}
					</Tag>
				)}

				{/* Polymarket probability */}
				{event.polymarketProbability !== null && (
					<Text style={{ fontSize: 12, color: token.colorTextSecondary }}>
						{t("overview.eventRadar.probability")} {(event.polymarketProbability * 100).toFixed(0)}%
					</Text>
				)}

				{/* 24h change */}
				{event.polymarketChange24h !== null && (
					<Text
						style={{
							fontSize: 12,
							color:
								event.polymarketChange24h > 0
									? token.colorSuccess
									: event.polymarketChange24h < 0
										? token.colorError
										: token.colorTextTertiary,
						}}
					>
						{t("overview.eventRadar.change24h")}{" "}
						{change24hSign}
						{(event.polymarketChange24h * 100).toFixed(1)}pp
					</Text>
				)}
			</Space>
		</div>
	);
}

interface EventRadarSectionProps {
	data: EventRadarDto | undefined;
	isLoading: boolean;
	isError: boolean;
	onRetry: () => void;
}

// EventRadarSection renders the upcoming macro event list.
// On error, shows a retry prompt (G3.9) rather than hiding the section entirely.
export function EventRadarSection({ data, isLoading, isError, onRetry }: EventRadarSectionProps) {
	const { t } = useTranslation("market");
	const { token } = useToken();

	return (
		<Card
			styles={{ body: { padding: "16px" } }}
			style={{ marginTop: 24 }}
		>
			<Title level={5} style={{ marginBottom: 4, marginTop: 0, fontSize: 14, fontWeight: 600 }}>
				{t("overview.eventRadar.title")}
			</Title>
			<Text style={{ fontSize: 12, color: token.colorTextTertiary, display: "block", marginBottom: 12 }}>
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

			{!isLoading && !isError && data && (
				<>
					{data.events.length === 0 ? (
						<Text style={{ fontSize: 13, color: token.colorTextTertiary }}>
							{t("overview.eventRadar.noEvents")}
						</Text>
					) : (
						<div>
							{data.events.map((event) => (
								<EventRow key={event.id} event={event} />
							))}
						</div>
					)}
				</>
			)}
		</Card>
	);
}
