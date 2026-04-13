import { Card, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

// EventCalendar is a placeholder for the event radar component filtered by
// the current asset code. Full implementation depends on the event-radar
// feature module (Step 17).
export function EventCalendar() {
	const { t } = useTranslation("app");

	return (
		<Card title={t("assetDetail.risk.events.title")} size="small" style={{ marginBottom: 16 }}>
			<Text type="secondary">—</Text>
		</Card>
	);
}
