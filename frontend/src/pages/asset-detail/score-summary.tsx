import { Space, Tag, Tooltip, Typography } from "@/ui-kit/eat";
import { QuestionCircleOutlined } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { getSignalColor } from "./utils";

const { Text } = Typography;

interface Props {
	score: number;
	signal: string;
	percentileLabel: string;
}

export function ScoreSummary({ score, signal, percentileLabel }: Props) {
	const { t } = useTranslation("app");
	const color = getSignalColor(signal);
	const signalText = t(`assetDetail.scoreSummary.signal.${signal}`, signal);
	const percentileText = t(
		`assetDetail.scoreSummary.percentile.${percentileLabel}`,
		percentileLabel,
	);

	return (
		<Space align="center" wrap>
			<Text strong style={{ fontSize: 22, color }}>
				{score}
			</Text>
			<Text type="secondary" style={{ fontSize: 12 }}>
				/ 100
			</Text>
			<Tag color={color === "#52c41a" ? "green" : color === "#f5222d" ? "red" : "default"}>
				{signalText}
			</Tag>
			<Tooltip title={t("assetDetail.scoreSummary.score")}>
				<Text type="secondary" style={{ fontSize: 12, cursor: "help" }}>
					{percentileText} <QuestionCircleOutlined />
				</Text>
			</Tooltip>
		</Space>
	);
}
