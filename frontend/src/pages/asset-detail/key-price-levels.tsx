import type { KeyPriceLevelDto } from "@/features/asset-detail";
import { Card, Table, Tag } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { formatPrice, formatUsdEquiv } from "./utils";

interface Props {
	levels: KeyPriceLevelDto[];
	currentPrice?: number;
	currency?: "USD" | "CNY";
	usdExchangeRate: number | null;
}

export function KeyPriceLevels({
	levels,
	currentPrice: _currentPrice,
	currency,
	usdExchangeRate,
}: Props) {
	const { t } = useTranslation("app");

	const columns = [
		{
			title: `${t("assetDetail.risk.keyPriceLevels.support")} / ${t("assetDetail.risk.keyPriceLevels.resistance")}`,
			dataIndex: "type",
			key: "type",
			render: (type: string) =>
				type === "support" ? (
					<Tag color="green">{t("assetDetail.risk.keyPriceLevels.support")}</Tag>
				) : (
					<Tag color="red">{t("assetDetail.risk.keyPriceLevels.resistance")}</Tag>
				),
		},
		{
			title: t("assetDetail.risk.keyPriceLevels.support"),
			dataIndex: "price",
			key: "price",
			render: (price: number) => {
				const formatted = formatPrice(price, currency);
				if (currency === "CNY" && usdExchangeRate) {
					const equiv = formatUsdEquiv(price, usdExchangeRate);
					return (
						<span>
							{formatted}
							{equiv && (
								<span style={{ color: "#8c8c8c", fontSize: 11, marginLeft: 4 }}>
									{t("assetDetail.risk.keyPriceLevels.usdEquiv", {
										price: equiv.replace("~$", ""),
									})}
								</span>
							)}
						</span>
					);
				}
				return formatted;
			},
		},
		{
			title: t("assetDetail.risk.keyPriceLevels.distance"),
			dataIndex: "distancePct",
			key: "distancePct",
			render: (pct: number | undefined) => {
				if (pct === undefined || pct === null) return "—";
				const sign = pct > 0 ? "+" : "";
				return (
					<span style={{ color: pct > 0 ? "#52c41a" : "#f5222d" }}>
						{sign}
						{pct.toFixed(1)}%
					</span>
				);
			},
		},
	];

	return (
		<Card
			title={t("assetDetail.risk.keyPriceLevels.title")}
			size="small"
			style={{ marginBottom: 16 }}
		>
			<Table<KeyPriceLevelDto>
				dataSource={levels}
				columns={columns}
				rowKey={(r) => `${r.type}-${r.price}`}
				size="small"
				pagination={false}
			/>
		</Card>
	);
}
