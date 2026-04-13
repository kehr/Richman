import { Button, Card, Space, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Paragraph, Title } = Typography;

interface EmptyHoldingsHeroProps {
	onAddHolding: () => void;
}

// EmptyHoldingsHero is the dashboard state shown when the authenticated user
// has no holdings. It is intentionally minimal: a big centered card with a
// single primary CTA that routes back to the portfolio add flow.
export function EmptyHoldingsHero({ onAddHolding }: EmptyHoldingsHeroProps) {
	const { t } = useTranslation("app");

	return (
		<Card
			data-testid="empty-holdings-hero"
			styles={{
				body: {
					display: "flex",
					justifyContent: "center",
					alignItems: "center",
					minHeight: 360,
					padding: 48,
				},
			}}
		>
			<Space direction="vertical" align="center" size={16}>
				<Title level={2} style={{ margin: 0, textAlign: "center" }}>
					{t("dashboard.emptyHero.title")}
				</Title>
				<Paragraph type="secondary" style={{ marginBottom: 0, textAlign: "center" }}>
					{t("dashboard.emptyHero.description")}
				</Paragraph>
				<Button
					type="primary"
					size="large"
					onClick={onAddHolding}
					data-testid="empty-holdings-hero-cta"
				>
					{t("dashboard.emptyHero.addButton")}
				</Button>
			</Space>
		</Card>
	);
}
