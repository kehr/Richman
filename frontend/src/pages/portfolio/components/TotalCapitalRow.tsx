import { formatAmount } from "@/domain/money/format";
import { useUserSettings } from "@/features/user-settings";
import { Flex, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

// TotalCapitalRow renders the secondary header row beneath the portfolio
// page title. It shows the configured total capital (with an inline link to
// the settings page to update it), or a prompt nudging the user to set the
// capital so amount columns can be displayed (PRD §4.1).
export function TotalCapitalRow() {
	const { data: settings, isLoading } = useUserSettings();
	const { t, i18n } = useTranslation("app");

	if (isLoading) {
		return null;
	}

	const totalCapital = settings?.totalCapitalCny;
	const hasCapital = totalCapital != null;

	return (
		<Flex align="center" gap={8} data-testid="total-capital-row" style={{ marginBottom: 16 }}>
			{hasCapital ? (
				<>
					<Typography.Text type="secondary">{t("portfolio.totalCapital.label")}</Typography.Text>
					<Typography.Text strong data-testid="total-capital-amount">
						{formatAmount(totalCapital, i18n.language)}
					</Typography.Text>
					<Link to="/settings" data-testid="total-capital-edit">
						{t("portfolio.totalCapital.editLink")}
					</Link>
				</>
			) : (
				<>
					<Typography.Text type="secondary">{t("portfolio.totalCapital.notSet")}</Typography.Text>
					<Link to="/settings" data-testid="total-capital-set">
						{t("portfolio.totalCapital.setLink")}
					</Link>
				</>
			)}
		</Flex>
	);
}
