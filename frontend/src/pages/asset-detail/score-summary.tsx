import { Space, Tag, Tooltip, Typography } from "@/ui-kit/eat";
import { QuestionCircleOutlined } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { PLACEHOLDER, getSignalColor } from "./utils";

const { Text } = Typography;

interface Props {
	score?: number;
	signal?: string;
	percentileLabel?: string;
}

export function ScoreSummary({ score, signal, percentileLabel }: Props) {
	const { t } = useTranslation("app");
	const color = getSignalColor(signal);
	const signalText = signal ? t(`assetDetail.scoreSummary.signal.${signal}`, signal) : null;
	const percentileText = percentileLabel
		? t(`assetDetail.scoreSummary.percentile.${percentileLabel}`, percentileLabel)
		: null;

	return (
		<Space align="center" wrap>
			<Text strong style={{ fontSize: 22, color }}>
				{score !== undefined && score !== null ? score : PLACEHOLDER}
			</Text>
			<Text type="secondary" style={{ fontSize: 12 }}>
				/ 100
			</Text>
			{signalText && (
				<Tag color={color === "#52c41a" ? "green" : color === "#f5222d" ? "red" : "default"}>
					{signalText}
				</Tag>
			)}
			{percentileText && (
				<Tooltip title={t("assetDetail.scoreSummary.score")}>
					<Text type="secondary" style={{ fontSize: 12, cursor: "help" }}>
						{percentileText} <QuestionCircleOutlined />
					</Text>
				</Tooltip>
			)}
		</Space>
	);
}
