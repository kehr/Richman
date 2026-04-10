import { StorageKeys } from "@/domain/storage/local-storage";
import { useLocalStorage } from "@/domain/storage/use-local-storage";
import { useOnboardingStatus } from "@/features/user-settings";
import { Alert, Button, Space } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

// OnboardingSkippedNudge renders at the top of the Dashboard for users who
// explicitly skipped the onboarding wizard (status.skipped === true &&
// !status.completed). It gives them a low-friction regret path back into the
// flow without forcing them through the guard redirect. Dismissal is
// persisted in localStorage; EmptyHoldingsHero still exposes a secondary
// text link so dismissed users are not dead-ended.
export function OnboardingSkippedNudge() {
	const { t } = useTranslation("app");
	const { data: status, isLoading } = useOnboardingStatus();
	const navigate = useNavigate();
	const [dismissed, setDismissed] = useLocalStorage<boolean>(
		StorageKeys.onboardingNudgeDismissed,
		false,
	);

	if (isLoading) return null;
	if (!status?.skipped) return null;
	if (dismissed) return null;

	const handleDismiss = () => setDismissed(true);
	const handleRestart = () => navigate("/onboarding/welcome");

	return (
		<Alert
			data-testid="onboarding-skipped-nudge"
			type="info"
			showIcon
			message={t("dashboard.skippedNudge.message")}
			action={
				<Space>
					<Button
						type="primary"
						size="small"
						onClick={handleRestart}
						data-testid="onboarding-skipped-nudge-restart"
					>
						{t("dashboard.skippedNudge.restart")}
					</Button>
					<Button
						size="small"
						onClick={handleDismiss}
						data-testid="onboarding-skipped-nudge-dismiss"
					>
						{t("dashboard.skippedNudge.dismiss")}
					</Button>
				</Space>
			}
			style={{ marginBottom: 16 }}
		/>
	);
}
