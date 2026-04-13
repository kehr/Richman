import { Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

interface Props {
	text: string;
}

// ChangeSummary is shown when scoreDelta >= 5.
export function ChangeSummary({ text }: Props) {
	const { t } = useTranslation("app");

	return (
		<div style={{ padding: "4px 0" }}>
			<Text type="secondary" style={{ fontSize: 12 }}>
				{t("assetDetail.changeSummary.title")}:{" "}
			</Text>
			<Text style={{ fontSize: 12 }}>{text}</Text>
		</div>
	);
}
