import { Alert } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface Props {
	type: string;
	message: string;
}

// ConflictWarning is shown when the API returns a conflictType.
export function ConflictWarning({ type: _type, message }: Props) {
	const { t } = useTranslation("app");

	return (
		<Alert
			type="warning"
			showIcon
			message={t("assetDetail.conflict.warning")}
			description={message}
			style={{ margin: "4px 0", fontSize: 12 }}
		/>
	);
}
