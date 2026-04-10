import { Badge, Card, Flex, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// HKPlaceholderCard shows the HK market slot as a non-interactive coming-soon
// placeholder. Dashed border and reduced opacity signal that this market is not
// yet configurable.
export function HKPlaceholderCard() {
	const { t } = useTranslation("settings");

	return (
		<Card
			size="small"
			style={{
				borderStyle: "dashed",
				opacity: 0.55,
				marginBottom: 12,
			}}
		>
			<Flex align="center" gap={10}>
				<Typography.Text strong>{t("schedule.markets.hk_stock")}</Typography.Text>
				<Badge status="default" text={t("schedule.markets.hkComingSoon")} />
			</Flex>
		</Card>
	);
}
