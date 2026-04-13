import { Tag, Tooltip } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// HoldingUpgradeTag is displayed on holdings created in "tag" entry mode (TRD SS7.3).
// It nudges the user to add cost and position details.
export function HoldingUpgradeTag() {
	const { t } = useTranslation("app");

	return (
		<Tooltip title={t("portfolio.upgradeTag.tooltip")}>
			<Tag color="blue" style={{ cursor: "default", marginRight: 0 }}>
				{t("portfolio.upgradeTag.label")}
			</Tag>
		</Tooltip>
	);
}
