import type { IndexSnapshotDto, MarketRegimeDto } from "@/features/market-overview";
import { Card, Space, Spin, Typography, theme } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { getPriceChangeColor } from "../utils";

const { Text } = Typography;
const { useToken } = theme;

interface RegimeBadgeProps {
	regime: MarketRegimeDto["regime"];
	label: string;
}

function RegimeBadge({ regime, label }: RegimeBadgeProps) {
	const { token } = useToken();
	const bgColor =
		regime === "risk_on"
			? token.colorSuccess
			: regime === "risk_off"
				? token.colorError
				: token.colorTextTertiary;

	return (
		<span
			style={{
				display: "inline-block",
				padding: "2px 10px",
				borderRadius: token.borderRadiusSM,
				backgroundColor: bgColor,
				color: "#fff",
				fontWeight: 600,
				fontSize: 13,
				lineHeight: "22px",
				whiteSpace: "nowrap",
			}}
		>
			{label}
		</span>
	);
}

interface IndexMiniCardProps {
	index: IndexSnapshotDto;
}

function IndexMiniCard({ index }: IndexMiniCardProps) {
	const { token } = useToken();
	const changeColor = getPriceChangeColor(index.code, index.changePercent);
	const colorMap: Record<string, string> = {
		red: token.colorError,
		green: token.colorSuccess,
		gray: token.colorTextTertiary,
	};
	const color = colorMap[changeColor] ?? token.colorTextTertiary;
	const sign = index.changePercent > 0 ? "+" : "";

	return (
		<div
			style={{
				padding: "4px 10px",
				borderRadius: token.borderRadius,
				background: token.colorBgLayout,
				textAlign: "center",
				minWidth: 80,
			}}
		>
			<div style={{ fontSize: 11, color: token.colorTextSecondary, whiteSpace: "nowrap" }}>
				{index.name}
			</div>
			<div style={{ fontSize: 13, fontWeight: 600, color: token.colorText }}>
				{index.price.toLocaleString()}
			</div>
			<div style={{ fontSize: 11, color, fontWeight: 500 }}>
				{sign}
				{index.changePercent.toFixed(2)}%
			</div>
		</div>
	);
}

interface MarketRegimeBarProps {
	data: MarketRegimeDto | undefined;
	isLoading: boolean;
	isError: boolean;
}

// MarketRegimeBar shows the current market regime signal and key index snapshots.
// When richson returns an error (503 / network failure), the bar is hidden
// entirely — callers pass isError and we render nothing (G3.9).
export function MarketRegimeBar({ data, isLoading, isError }: MarketRegimeBarProps) {
	const { t } = useTranslation("market");
	const { token } = useToken();

	// G3.9: hide the bar completely on error
	if (isError) return null;

	if (isLoading) {
		return (
			<Card
				style={{ marginBottom: 16 }}
				styles={{ body: { padding: "12px 16px" } }}
			>
				<Spin size="small" />
			</Card>
		);
	}

	if (!data) return null;

	const regimeLabel = t(`overview.regime.${data.regime}`);

	return (
		<Card
			style={{ marginBottom: 16 }}
			styles={{ body: { padding: "12px 16px" } }}
		>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "space-between",
					flexWrap: "wrap",
					gap: 12,
				}}
			>
				{/* Left: regime badge + summary */}
				<Space size={10} align="center" wrap>
					<RegimeBadge regime={data.regime} label={regimeLabel} />
					<Text
						style={{
							fontSize: 13,
							color: token.colorTextSecondary,
							maxWidth: 400,
						}}
					>
						{data.summary}
					</Text>
				</Space>

				{/* Right: index mini cards */}
				<Space size={8} wrap>
					{data.indices.map((idx) => (
						<IndexMiniCard key={idx.code} index={idx} />
					))}
				</Space>
			</div>
		</Card>
	);
}
