import type { RiskFactorDto } from "@/features/asset-detail";
import { Card, List, Tag, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

interface Props {
	factors?: RiskFactorDto[];
}

const SEVERITY_COLORS: Record<string, string> = {
	high: "red",
	medium: "orange",
	low: "default",
};

export function RiskFactorList({ factors }: Props) {
	const { t } = useTranslation("app");
	const items = factors ?? [];

	return (
		<Card title={t("assetDetail.risk.factors.title")} size="small" style={{ marginBottom: 16 }}>
			{items.length === 0 ? (
				<Text type="secondary">—</Text>
			) : (
				<List
					dataSource={items}
					renderItem={(f) => (
						<List.Item key={f.id}>
							<div style={{ display: "flex", gap: 8, alignItems: "flex-start", width: "100%" }}>
								<Tag color={SEVERITY_COLORS[f.severity] ?? "default"} style={{ flexShrink: 0 }}>
									{t(`assetDetail.risk.factors.severity.${f.severity}`)}
								</Tag>
								<Text style={{ flex: 1, lineHeight: 1.6 }}>{f.description}</Text>
							</div>
						</List.Item>
					)}
					size="small"
				/>
			)}
		</Card>
	);
}
