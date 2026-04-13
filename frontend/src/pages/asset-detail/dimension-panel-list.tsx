import type { DimensionDetailDto, DimensionSubIndicator } from "@/features/asset-detail";
import { Collapse, Table, Tag, Tooltip, Typography } from "@/ui-kit/eat";
import { QuestionCircleOutlined } from "@/ui-kit/eat";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { getSignalColor } from "./utils";

const { Text } = Typography;

interface Props {
	dimensions: DimensionDetailDto[];
}

interface DimItemConfig {
	key: string;
	label: ReactNode;
	children: ReactNode;
}

export function DimensionPanelList({ dimensions }: Props) {
	const { t } = useTranslation("app");

	const subColumns = [
		{ title: t("assetDetail.dimension.subIndicators.name"), dataIndex: "name", key: "name" },
		{
			title: t("assetDetail.dimension.subIndicators.rawValue"),
			dataIndex: "rawValue",
			key: "rawValue",
			render: (v: number | string) => String(v),
		},
		{
			title: t("assetDetail.dimension.subIndicators.percentile"),
			dataIndex: "percentile",
			key: "percentile",
			render: (v: number | null) => (v !== null ? `${v.toFixed(0)}%` : "-"),
		},
		{
			title: t("assetDetail.dimension.subIndicators.normalizedScore"),
			dataIndex: "normalizedScore",
			key: "normalizedScore",
			render: (v: number) => v.toFixed(1),
		},
		{
			title: t("assetDetail.dimension.subIndicators.weight"),
			dataIndex: "weight",
			key: "weight",
			render: (v: number) => `${(v * 100).toFixed(0)}%`,
		},
	];

	const items: DimItemConfig[] = dimensions.map((dim) => {
		const color = getSignalColor(dim.signal);
		const dimName = t(`assetDetail.dimension.${dim.id}.name`, dim.name);
		const explanation = t(`assetDetail.dimension.${dim.id}.explanation`, "");
		const hasLlmAdjustment = dim.llmAdjustment !== null && dim.llmAdjustment !== 0;
		const adj = dim.llmAdjustment ?? 0;
		const scoreDisplay = hasLlmAdjustment
			? `${dim.score} (${t("assetDetail.dimension.baseScore")} ${dim.quantScore} → LLM ${adj > 0 ? "+" : ""}${adj})`
			: `${dim.score}`;
		const dimWeight = `${t("assetDetail.dimension.weight")}: ${(dim.weight * 100).toFixed(0)}%`;

		return {
			key: dim.id,
			label: (
				<div
					key={`label-${dim.id}`}
					style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}
				>
					<Text strong>{dimName}</Text>
					{explanation && (
						<Tooltip title={explanation}>
							<QuestionCircleOutlined style={{ color: "#8c8c8c", cursor: "help" }} />
						</Tooltip>
					)}
					<Tag color={color === "#52c41a" ? "green" : color === "#f5222d" ? "red" : "default"}>
						{scoreDisplay}
					</Tag>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{dimWeight}
					</Text>
				</div>
			),
			children: (
				<div key={`children-${dim.id}`}>
					{dim.llmReason && (
						<Text type="secondary" style={{ display: "block", marginBottom: 8, fontSize: 12 }}>
							{dim.llmReason}
						</Text>
					)}
					<Table<DimensionSubIndicator>
						dataSource={dim.subIndicators}
						columns={subColumns}
						rowKey="name"
						size="small"
						pagination={false}
					/>
				</div>
			),
		};
	});

	return (
		<div style={{ marginTop: 16 }}>
			<Text strong style={{ display: "block", marginBottom: 8 }}>
				{t("assetDetail.dimension.title")}
			</Text>
			<Collapse size="small" items={items} />
		</div>
	);
}
