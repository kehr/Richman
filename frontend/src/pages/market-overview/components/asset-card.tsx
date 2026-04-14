import type { AssetCardDto } from "@/features/market-overview";
import { Card, Tag, Typography, theme } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { getDirectionColor } from "../utils";

const { Text } = Typography;
const { useToken } = theme;

interface AssetCardProps {
	asset: AssetCardDto;
}

// AssetCard renders a single asset tile in the card wall.
// An asset is considered "analyzed" when it has an overallScore. Analyzed
// assets link to /market/:code and show score + signal data. Assets with no
// analysis yet render a greyed "waiting analysis" placeholder and are not
// clickable. Per docs/standards/contract-drift.md this component must only
// read fields that the backend `AssetCardDTO` actually returns.
export function AssetCard({ asset }: AssetCardProps) {
	const { t, i18n } = useTranslation("market");
	const { token } = useToken();
	const navigate = useNavigate();

	const displayName = i18n.language === "zh" ? asset.name : asset.nameEn;
	const hasAnalysis = typeof asset.overallScore === "number";

	if (!hasAnalysis) {
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
						{t("overview.assetCard.waitingAnalysis")}
					</Tag>
				</div>
			</Card>
		);
	}

	const dirColor = asset.signalLevel ? getDirectionColor(asset.signalLevel) : "gray";
	const signalTagColor =
		dirColor === "green" ? "success" : dirColor === "red" ? "error" : "default";

	const scoreDelta = asset.scoreDelta ?? null;
	const deltaColor =
		scoreDelta === null || scoreDelta === 0
			? token.colorTextTertiary
			: scoreDelta > 0
				? token.colorSuccess
				: token.colorError;
	const deltaSign = scoreDelta !== null && scoreDelta > 0 ? "+" : "";

	return (
		<Card
			hoverable
			onClick={() => navigate(`/market/${asset.code}`)}
			style={{ cursor: "pointer" }}
			styles={{ body: { padding: "12px 14px" } }}
		>
			<div style={{ marginBottom: 6 }}>
				<Text strong style={{ fontSize: 13, color: token.colorText }}>
					{displayName}
				</Text>
				<Text style={{ fontSize: 11, color: token.colorTextTertiary, marginLeft: 6 }}>
					{asset.code}
				</Text>
			</div>

			{/* Score + delta */}
			<div style={{ marginBottom: 6 }}>
				<Text strong style={{ fontSize: 16, color: token.colorText }}>
					{asset.overallScore}
				</Text>
				<Text style={{ fontSize: 11, color: token.colorTextTertiary, marginLeft: 4 }}>/ 100</Text>
				{scoreDelta !== null && (
					<Text style={{ fontSize: 12, color: deltaColor, marginLeft: 8 }}>
						{deltaSign}
						{scoreDelta.toFixed(1)}
					</Text>
				)}
			</div>

			{/* Signal */}
			{asset.signalLevel && (
				<Tag color={signalTagColor} style={{ fontSize: 11, margin: 0 }}>
					{t(`overview.assetCard.signal.${asset.signalLevel}`)}
				</Tag>
			)}
		</Card>
	);
}
