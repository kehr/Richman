import { Card, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

interface Props {
	text: string;
}

export function InterpretationCard({ text }: Props) {
	const { t } = useTranslation("app");

	return (
		<Card title={t("assetDetail.interpretation.title")} size="small" style={{ marginTop: 16 }}>
			<Text style={{ whiteSpace: "pre-wrap", lineHeight: 1.7 }}>{text}</Text>
		</Card>
	);
}
