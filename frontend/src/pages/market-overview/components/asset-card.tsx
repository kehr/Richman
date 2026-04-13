import type { AssetCardDto } from "@/features/market-overview";
import { Card, Tag, Typography, theme } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { formatCurrencyPrice, getDirectionColor, getPriceChangeColor } from "../utils";

const { Text } = Typography;
const { useToken } = theme;

interface AssetCardProps {
	asset: AssetCardDto;
}

// AssetCard renders a single asset tile in the card wall.
// Active assets link to /market/:code and show live price + score data.
// Greyed assets display name + "Coming Soon" and are not clickable.
export function AssetCard({ asset }: AssetCardProps) {
	const { t, i18n } = useTranslation("market");
	const { token } = useToken();
	const navigate = useNavigate();

	const displayName = i18n.language === "zh" ? asset.nameZh : asset.nameEn;

	if (!asset.isActive) {
		return (
			<Card
				style={{
					opacity: 0.45,
					cursor: "not-allowed",
					userSelect: "none",
				}}
				styles={{ body: { padding: "12px 14px" } }}
			>
				<Text strong style={{ fontSize: 13, color: token.colorTextDisabled }}>
					{displayName}
				</Text>
				<div style={{ marginTop: 6 }}>
					<Tag color="default" style={{ fontSize: 11 }}>
						{t("overview.assetCard.comingSoon")}
					</Tag>
				</div>
			</Card>
		);
	}

	const changeColor = getPriceChangeColor(asset.code, asset.changePercent ?? 0);
	const colorMap: Record<string, string> = {
		red: token.colorError,
		green: token.colorSuccess,
		gray: token.colorTextTertiary,
	};
	const priceColor = colorMap[changeColor] ?? token.colorTextTertiary;

	const dirColor = asset.signal ? getDirectionColor(asset.signal) : "gray";
	const signalTagColor =
		dirColor === "green" ? "success" : dirColor === "red" ? "error" : "default";

	const sign = (asset.changePercent ?? 0) > 0 ? "+" : "";

	return (
		<Card
			hoverable
			onClick={() => navigate(`/market/${asset.code}`)}
			style={{ cursor: "pointer" }}
			styles={{ body: { padding: "12px 14px" } }}
		>
			<div style={{ marginBottom: 4 }}>
				<Text strong style={{ fontSize: 13, color: token.colorText }}>
					{displayName}
				</Text>
				<Text style={{ fontSize: 11, color: token.colorTextTertiary, marginLeft: 6 }}>
					{asset.code}
				</Text>
			</div>

			{/* Price + change */}
			{asset.price !== null && (
				<div style={{ marginBottom: 8 }}>
					<Text strong style={{ fontSize: 15, color: token.colorText }}>
						{formatCurrencyPrice(asset.price, asset.currency)}
					</Text>
					{asset.changePercent !== null && (
						<Text style={{ fontSize: 12, color: priceColor, marginLeft: 6 }}>
							{sign}
							{asset.changePercent.toFixed(2)}%
						</Text>
					)}
				</div>
			)}

			{/* Score + signal */}
			<div
				style={{
					display: "flex",
					alignItems: "center",
					gap: 6,
					flexWrap: "wrap",
				}}
			>
				{asset.overallScore !== null && (
					<Text style={{ fontSize: 12, color: token.colorTextSecondary }}>
						{t("overview.assetCard.score")} {asset.overallScore}
					</Text>
				)}
				{asset.signal && (
					<Tag color={signalTagColor} style={{ fontSize: 11, margin: 0 }}>
						{t(`overview.assetCard.signal.${asset.signal}`)}
					</Tag>
				)}
			</div>

			{/* Percentile label */}
			{asset.percentileLabel && (
				<div style={{ marginTop: 4 }}>
					<Text style={{ fontSize: 11, color: token.colorTextTertiary }}>
						{asset.percentileLabel}
					</Text>
				</div>
			)}
		</Card>
	);
}
