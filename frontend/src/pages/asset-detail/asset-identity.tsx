import { Space, Typography } from "@/ui-kit/eat";
import { formatPrice, getPriceChangeColor } from "./utils";

const { Title, Text } = Typography;

interface Props {
	code: string;
	name: string;
	nameEn: string;
	price?: number;
	currency?: "USD" | "CNY";
	changePercent?: number;
}

export function AssetIdentity({ code, name, nameEn, price, currency, changePercent }: Props) {
	const priceColor = getPriceChangeColor(code, changePercent);
	const hasPrice = price !== undefined && price !== null;
	const hasChange = changePercent !== undefined && changePercent !== null;
	const sign = hasChange && (changePercent as number) > 0 ? "+" : "";

	return (
		<Space align="baseline" wrap>
			<Title level={4} style={{ margin: 0 }}>
				{name}
			</Title>
			<Text type="secondary" style={{ fontSize: 13 }}>
				{nameEn} ({code})
			</Text>
			{hasPrice && (
				<Text strong style={{ fontSize: 20, color: priceColor }}>
					{formatPrice(price, currency)}
				</Text>
			)}
			{hasChange && (
				<Text style={{ color: priceColor, fontSize: 14 }}>
					{sign}
					{(changePercent as number).toFixed(2)}%
				</Text>
			)}
		</Space>
	);
}
