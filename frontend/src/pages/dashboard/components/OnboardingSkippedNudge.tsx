import { useOnboardingStatus } from "@/features/user-settings";
import { Alert, Button, Space } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

// DISMISS_KEY is the localStorage key that records a "不再提示" click on the
// Dashboard onboarding-skipped nudge. It is exported so other re-entry
// surfaces (e.g. the Settings re-entry CTA in step 17) can clear it when
// the user restarts the wizard from a different entry point — otherwise a
// restart-from-settings would not re-surface the dashboard nudge after the
// user skips again.
export const ONBOARDING_NUDGE_DISMISS_KEY = "richman_onboarding_nudge_dismissed";

function readDismissed(): boolean {
	try {
		return localStorage.getItem(ONBOARDING_NUDGE_DISMISS_KEY) === "1";
	} catch {
		// Private mode or storage disabled: treat as not-dismissed so the nudge
		// still shows and the user retains a path back to onboarding.
		return false;
	}
}

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
	// Read the dismissed flag once on mount via a lazy initializer so we don't
	// hit localStorage on every render.
	const [dismissed, setDismissed] = useState<boolean>(() => readDismissed());

	if (isLoading) return null;
	if (!status?.skipped) return null;
	if (dismissed) return null;

	const handleDismiss = () => {
		try {
			localStorage.setItem(ONBOARDING_NUDGE_DISMISS_KEY, "1");
		} catch {
			// Private mode: fall back to in-memory dismissal for this session.
		}
		setDismissed(true);
	};

	const handleRestart = () => {
		navigate("/onboarding/welcome");
	};

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
