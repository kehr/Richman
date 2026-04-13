import { Space, Typography } from "@/ui-kit/eat";
import { formatPrice, getPriceChangeColor } from "./utils";

const { Title, Text } = Typography;

interface Props {
	code: string;
	name: string;
	nameEn: string;
	price: number;
	currency: "USD" | "CNY";
	changePercent: number;
}

export function AssetIdentity({ code, name, nameEn, price, currency, changePercent }: Props) {
	const priceColor = getPriceChangeColor(code, changePercent);
	const sign = changePercent > 0 ? "+" : "";

	return (
		<Space align="baseline" wrap>
			<Title level={4} style={{ margin: 0 }}>
				{name}
			</Title>
			<Text type="secondary" style={{ fontSize: 13 }}>
				{nameEn} ({code})
			</Text>
			<Text strong style={{ fontSize: 20, color: priceColor }}>
				{formatPrice(price, currency)}
			</Text>
			<Text style={{ color: priceColor, fontSize: 14 }}>
				{sign}
				{changePercent.toFixed(2)}%
			</Text>
		</Space>
	);
}
