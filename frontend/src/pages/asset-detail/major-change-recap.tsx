import type { MajorChangeRecapDto } from "@/features/asset-detail";
import { Alert } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface Props {
	recap: MajorChangeRecapDto;
}

// MajorChangeRecap is shown when |scoreDelta| > 20.
export function MajorChangeRecap({ recap }: Props) {
	const { t } = useTranslation("app");

	return (
		<Alert
			type={recap.scoreDelta > 0 ? "success" : "error"}
			showIcon
			message={t("assetDetail.majorChangeRecap.title")}
			description={recap.text}
			style={{ margin: "4px 0", fontSize: 12 }}
		/>
	);
}
