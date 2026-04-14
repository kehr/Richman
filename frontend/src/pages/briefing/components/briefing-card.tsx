import type { BriefingCardDto } from "@/features/research-briefing";
import type { FeedbackRating } from "@/features/user-feedback";
import {
	Badge,
	Button,
	Card,
	DislikeOutlined,
	Flex,
	LikeOutlined,
	Space,
	Tag,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { BriefingViewMode } from "./briefing-header";
import { ScoreSparkline } from "./score-sparkline";

interface BriefingCardProps {
	card: BriefingCardDto;
	viewMode: BriefingViewMode;
	onClick: () => void;
	onFeedback: (rating: FeedbackRating) => void;
	feedbackPending?: boolean;
}

// BriefingCard renders one holding's briefing summary (TRD SS6.2).
// Compact mode shows: header (asset name + score + direction) + position info.
// Detailed mode adds: sparkline, change attribution, conflict warning,
// concentration banner, feedback buttons.
export function BriefingCard({
	card,
	viewMode,
	onClick,
	onFeedback,
	feedbackPending,
}: BriefingCardProps) {
	const { t } = useTranslation("app");
	const [localRating, setLocalRating] = useState<FeedbackRating | null>(null);

	const isDetailed = viewMode === "detailed";

	// Backend serialises decimal fields (cost_price, position_ratio) as strings;
	// parse once here so layout code can format/compare numerically. parseFloat
	// returns NaN for empty strings which we treat as "value unavailable".
	const costPriceNum = parseDecimalOrNull(card.costPrice);
	const positionRatioNum = parseDecimalOrNull(card.positionRatio);

	// Direction badge color
	const directionColor =
		card.direction === "bullish" ? "success" : card.direction === "bearish" ? "error" : "default";

	const directionLabel = card.direction
		? t(`briefing.direction.${card.direction}`, { defaultValue: card.direction })
		: null;

	const scoreColor =
		card.overallScore == null
			? "default"
			: card.overallScore >= 60
				? "success"
				: card.overallScore >= 40
					? "warning"
					: "error";

	const handleFeedback = (rating: FeedbackRating) => {
		setLocalRating(rating);
		onFeedback(rating);
	};

	const showChangeDelta =
		isDetailed &&
		card.scoreDelta != null &&
		Math.abs(card.scoreDelta) >= 5 &&
		card.changeAttribution;

	return (
		<Card
			hoverable
			onClick={onClick}
			size="small"
			style={{ cursor: "pointer" }}
			styles={{ body: { padding: "12px 16px" } }}
		>
			{/* Header: asset name + score + direction */}
			<Flex align="flex-start" justify="space-between" gap={8}>
				<Flex vertical gap={2}>
					<Typography.Text strong style={{ fontSize: 15 }}>
						{card.assetName}
					</Typography.Text>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{card.assetCode}
					</Typography.Text>
				</Flex>
				<Flex align="center" gap={6}>
					{card.overallScore != null && (
						<Badge
							count={card.overallScore}
							color={scoreColor === "success" ? "green" : scoreColor === "error" ? "red" : "orange"}
							style={{ fontSize: 12 }}
						/>
					)}
					{directionLabel && (
						<Tag color={directionColor} style={{ marginRight: 0 }}>
							{directionLabel}
						</Tag>
					)}
				</Flex>
			</Flex>

			{/* Position info: cost + ratio + PnL */}
			<Flex gap={16} style={{ marginTop: 10 }}>
				{costPriceNum != null && (
					<Flex vertical gap={0}>
						<Typography.Text type="secondary" style={{ fontSize: 11 }}>
							{t("briefing.card.costLabel")}
						</Typography.Text>
						<Typography.Text style={{ fontSize: 13 }}>{costPriceNum.toFixed(3)}</Typography.Text>
					</Flex>
				)}
				{positionRatioNum != null && (
					<Flex vertical gap={0}>
						<Typography.Text type="secondary" style={{ fontSize: 11 }}>
							{t("briefing.card.positionLabel")}
						</Typography.Text>
						<Typography.Text style={{ fontSize: 13 }}>
							{positionRatioNum.toFixed(1)}%
						</Typography.Text>
					</Flex>
				)}
				{card.pnlPercent != null && (
					<Flex vertical gap={0}>
						<Typography.Text type="secondary" style={{ fontSize: 11 }}>
							{t("briefing.card.pnlLabel")}
						</Typography.Text>
						<Typography.Text
							style={{
								fontSize: 13,
								color:
									card.pnlPercent > 0
										? "var(--ant-color-success)"
										: card.pnlPercent < 0
											? "var(--ant-color-error)"
											: undefined,
							}}
						>
							{card.pnlPercent > 0 ? "+" : ""}
							{card.pnlPercent.toFixed(2)}%
						</Typography.Text>
					</Flex>
				)}
			</Flex>

			{/* Sparkline — only in detailed mode */}
			{isDetailed && card.sparklineScores.length > 1 && (
				<div
					style={{ marginTop: 10 }}
					onClick={(e) => e.stopPropagation()}
					onKeyDown={(e) => e.stopPropagation()}
					role="presentation"
				>
					<ScoreSparkline data={card.sparklineScores} />
				</div>
			)}

			{/* Today's score change attribution — detailed, delta >= 5 */}
			{showChangeDelta && card.scoreDelta != null && (
				<Flex align="center" gap={4} style={{ marginTop: 8 }}>
					<Typography.Text
						type={card.scoreDelta > 0 ? "success" : "danger"}
						style={{ fontSize: 12 }}
					>
						{card.scoreDelta > 0 ? "+" : ""}
						{card.scoreDelta}
					</Typography.Text>
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{card.changeAttribution}
					</Typography.Text>
				</Flex>
			)}

			{/* Conflict warning — detailed only */}
			{isDetailed && card.conflictWarning && (
				<Typography.Text type="warning" style={{ fontSize: 12, display: "block", marginTop: 6 }}>
					{card.conflictWarning}
				</Typography.Text>
			)}

			{/* Concentration banner — detailed only, non-green */}
			{isDetailed && card.concentrationLevel !== "green" && (
				<Typography.Text type="secondary" style={{ fontSize: 12, display: "block", marginTop: 6 }}>
					{card.concentrationMessage}
				</Typography.Text>
			)}

			{/* Feedback buttons — detailed mode only, requires a backing analysis row. */}
			{isDetailed && card.assetAnalysisId != null && (
				<Flex
					align="center"
					justify="flex-end"
					gap={4}
					style={{ marginTop: 10 }}
					onClick={(e) => e.stopPropagation()}
					onKeyDown={(e) => e.stopPropagation()}
					role="presentation"
				>
					<Space size={4}>
						<Tooltip title={t("briefing.feedback.upTooltip")}>
							<Button
								type={localRating === "up" ? "primary" : "text"}
								size="small"
								icon={<LikeOutlined />}
								disabled={feedbackPending}
								onClick={() => handleFeedback("up")}
								aria-label={t("briefing.feedback.upTooltip")}
							/>
						</Tooltip>
						<Tooltip title={t("briefing.feedback.downTooltip")}>
							<Button
								type={localRating === "down" ? "primary" : "text"}
								size="small"
								danger={localRating === "down"}
								icon={<DislikeOutlined />}
								disabled={feedbackPending}
								onClick={() => handleFeedback("down")}
								aria-label={t("briefing.feedback.downTooltip")}
							/>
						</Tooltip>
					</Space>
				</Flex>
			)}
		</Card>
	);
}

// parseDecimalOrNull converts a backend decimal string (e.g. "12.345", "0")
// to a number. Returns null for empty/invalid values so callers can render an
// "unavailable" placeholder instead of NaN.
function parseDecimalOrNull(raw: string | null | undefined): number | null {
	if (raw == null || raw === "") return null;
	const n = Number.parseFloat(raw);
	return Number.isFinite(n) ? n : null;
}
