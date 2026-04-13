import type { DrawdownReferenceDto } from "@/features/asset-detail";
import { Card, Space, Statistic, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

interface Props {
	drawdown: DrawdownReferenceDto;
}

export function DrawdownReference({ drawdown }: Props) {
	const { t } = useTranslation("app");

	return (
		<Card title={t("assetDetail.risk.drawdown.title")} size="small" style={{ marginBottom: 16 }}>
			<Space size="large" wrap>
				<Statistic
					title={
						<span>
							{t("assetDetail.risk.drawdown.currentBull")}
							<br />
							<Text style={{ fontSize: 11, color: "#8c8c8c" }}>
								{t("assetDetail.risk.drawdown.date", {
									date: drawdown.currentBullMaxDrawdownDate,
								})}
							</Text>
						</span>
					}
					value={`${drawdown.currentBullMaxDrawdown.toFixed(1)}%`}
					valueStyle={{ color: "#f5222d" }}
				/>
				<Statistic
					title={t("assetDetail.risk.drawdown.historicalAvg")}
					value={`${drawdown.historicalAvgDrawdown.toFixed(1)}%`}
					valueStyle={{ color: "#fa8c16" }}
				/>
			</Space>
		</Card>
	);
}
