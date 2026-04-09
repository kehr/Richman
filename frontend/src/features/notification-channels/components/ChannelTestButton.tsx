import { Button, Tooltip } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// ChannelTestButton is a disabled placeholder for the "test send" action.
// The backend has no test-send endpoint yet (see Step 17 trade-delete pattern),
// so we render an always-disabled button with a tooltip explaining the
// situation rather than wiring up a frontend-only mock path.
export function ChannelTestButton() {
	const { t } = useTranslation("settings");

	return (
		<Tooltip title={t("channels.list.testTooltip")}>
			<Button size="small" disabled data-testid="channel-test-button">
				{t("channels.list.testButton")}
			</Button>
		</Tooltip>
	);
}
