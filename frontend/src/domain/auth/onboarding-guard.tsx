import { useOnboardingStatus } from "@/features/user-settings";
import { Spin } from "@/ui-kit/eat";
import { type ReactNode, useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

interface OnboardingGuardProps {
	children: ReactNode;
}

const ONBOARDING_PREFIX = "/onboarding";
const ONBOARDING_ENTRY = "/onboarding/welcome";
const POST_ONBOARDING_HOME = "/briefing";

// OnboardingGuard sits between AuthGuard and the protected route tree and
// gates access based on two mutually-exclusive server flags:
//   - completed: the user finished the wizard through the normal flow
//   - skipped:   the user intentionally opted out via the skip button or
//                the nudge dismiss path
//
// State-space handling (PRD appendix A):
//   new user (neither flag):
//     * app-shell request  → redirect to /onboarding/welcome
//     * onboarding request → render
//   skipped (skipped=true, completed=false):
//     * app-shell request  → render (user bypassed the flow)
//     * onboarding request → render (nudge "开始引导" re-entry path)
//   completed (completed=true, skipped=false):
//     * app-shell request  → render
//     * onboarding request → redirect to /dashboard (no revisit)
//
// The skipped + onboarding_route combination is deliberately allowed so a
// user who clicked "跳过引导" can come back through the Dashboard nudge or
// the Settings re-entry CTA without the guard bouncing them. Completion
// remains a one-way door: once completed is true the guard forces the user
// back to the dashboard if they try to revisit the wizard.
export function OnboardingGuard({ children }: OnboardingGuardProps) {
	const navigate = useNavigate();
	const location = useLocation();
	const { data, isLoading } = useOnboardingStatus();

	const isOnboardingRoute = location.pathname.startsWith(ONBOARDING_PREFIX);
	const completed = data?.completed ?? false;
	const skipped = data?.skipped ?? false;
	const isBypassed = completed || skipped;

	useEffect(() => {
		if (isLoading || !data) {
			return;
		}
		// New users (neither bypass flag) cannot access the app shell until
		// they either finish or explicitly skip the wizard.
		if (!isBypassed && !isOnboardingRoute) {
			navigate(ONBOARDING_ENTRY, { replace: true });
			return;
		}
		// Completion is a one-way door: no wizard revisits.
		if (completed && isOnboardingRoute) {
			navigate(POST_ONBOARDING_HOME, { replace: true });
		}
	}, [completed, isBypassed, data, isLoading, isOnboardingRoute, navigate]);

	if (isLoading) {
		return (
			<div
				style={{
					display: "flex",
					justifyContent: "center",
					alignItems: "center",
					height: "100vh",
				}}
			>
				<Spin size="large" />
			</div>
		);
	}

	// While the effect above schedules a redirect, render nothing to avoid a
	// flash of the wrong tree.
	if (!isBypassed && !isOnboardingRoute) {
		return null;
	}
	if (completed && isOnboardingRoute) {
		return null;
	}

	return <>{children}</>;
}
