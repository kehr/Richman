// score-bar.tsx — visual score bar with confidence band overlay.
// 0-100 scale, 5 color zones, gradient band between bandLow and bandHigh.

import { Tooltip } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface Props {
	score: number;
	bandLow: number;
	bandHigh: number;
}

const ZONE_COLORS = [
	{ max: 20, color: "#f5222d" }, // 0-19 red
	{ max: 40, color: "#fa8c16" }, // 20-39 orange
	{ max: 60, color: "#8c8c8c" }, // 40-59 gray
	{ max: 80, color: "#52c41a" }, // 60-79 green
	{ max: 101, color: "#237804" }, // 80-100 dark green
];

function getZoneColor(score: number): string {
	return ZONE_COLORS.find((z) => score < z.max)?.color ?? "#237804";
}

export function ScoreBar({ score, bandLow, bandHigh }: Props) {
	const { t } = useTranslation("app");
	const bandLeft = `${bandLow}%`;
	const bandWidth = `${Math.max(0, bandHigh - bandLow)}%`;
	const markerLeft = `${Math.min(100, Math.max(0, score))}%`;
	const markerColor = getZoneColor(score);

	return (
		<Tooltip title={`${t("assetDetail.scoreBar.bandLabel")}: ${bandLow}–${bandHigh}`}>
			<div
				style={{
					position: "relative",
					height: 8,
					background:
						"linear-gradient(to right, #f5222d 0%, #f5222d 20%, #fa8c16 20%, #fa8c16 40%, #8c8c8c 40%, #8c8c8c 60%, #52c41a 60%, #52c41a 80%, #237804 80%)",
					borderRadius: 4,
					width: "100%",
					minWidth: 120,
					cursor: "help",
				}}
			>
				{/* Confidence band overlay */}
				<div
					style={{
						position: "absolute",
						left: bandLeft,
						width: bandWidth,
						top: 0,
						height: "100%",
						background: "rgba(255,255,255,0.35)",
						borderRadius: 4,
					}}
				/>
				{/* Score point marker */}
				<div
					style={{
						position: "absolute",
						left: markerLeft,
						top: "50%",
						transform: "translate(-50%, -50%)",
						width: 12,
						height: 12,
						borderRadius: "50%",
						background: markerColor,
						border: "2px solid white",
						boxShadow: "0 0 4px rgba(0,0,0,0.3)",
					}}
				/>
			</div>
		</Tooltip>
	);
}
