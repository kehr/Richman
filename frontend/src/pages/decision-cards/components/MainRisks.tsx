import { Alert, Space, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

interface MainRisksProps {
	riskWarnings: string[];
}

// MainRisks renders the yellow "main risks" block per PRD section 5. When
// the backend supplies an empty list we still render a placeholder so the
// page does not collapse — the user wants to see risk reasoning even when
// the model produced none ("no significant risks identified").
export function MainRisks({ riskWarnings }: MainRisksProps) {
	const { t } = useTranslation("app");
	const hasRisks = riskWarnings.length > 0;
	return (
		<Alert
			type="warning"
			showIcon
			data-testid="main-risks"
			message={<Text strong>{t("decisionCard.mainRisks.title")}</Text>}
			description={
				<Space direction="vertical" size={4} style={{ width: "100%" }}>
					{hasRisks ? (
						<ul style={{ margin: 0, paddingLeft: 20 }}>
							{riskWarnings.map((risk) => (
								<li key={risk}>{risk}</li>
							))}
						</ul>
					) : (
						<Text type="secondary">{t("decisionCard.mainRisks.noRisks")}</Text>
					)}
					<Text type="secondary">{t("decisionCard.mainRisks.exitCondition")}</Text>
				</Space>
			}
		/>
	);
}
