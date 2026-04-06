import { Spin } from "@/ui-kit/eat";
import { type ReactNode, useEffect } from "react";
import { useLocation, useNavigate } from "react-router";
import { useOnboardingStatus } from "./use-onboarding-status";

interface OnboardingGuardProps {
	children: ReactNode;
}

const ONBOARDING_PREFIX = "/onboarding";
const ONBOARDING_ENTRY = "/onboarding/welcome";
const POST_ONBOARDING_HOME = "/dashboard";

// OnboardingGuard sits between AuthGuard and the protected route tree. When the
// current user has not finished onboarding, any request outside of the
// /onboarding/* branch is redirected to the welcome step. Conversely, a user
// who already completed onboarding cannot revisit the flow.
export function OnboardingGuard({ children }: OnboardingGuardProps) {
	const navigate = useNavigate();
	const location = useLocation();
	const { data, isLoading } = useOnboardingStatus();

	const isOnboardingRoute = location.pathname.startsWith(ONBOARDING_PREFIX);
	const completed = data?.completed ?? false;

	useEffect(() => {
		if (isLoading || !data) {
			return;
		}
		if (!completed && !isOnboardingRoute) {
			navigate(ONBOARDING_ENTRY, { replace: true });
			return;
		}
		if (completed && isOnboardingRoute) {
			navigate(POST_ONBOARDING_HOME, { replace: true });
		}
	}, [completed, data, isLoading, isOnboardingRoute, navigate]);

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

	// While the effect above schedules a redirect we still render nothing to
	// avoid a flash of the wrong tree.
	if (!completed && !isOnboardingRoute) {
		return null;
	}
	if (completed && isOnboardingRoute) {
		return null;
	}

	return <>{children}</>;
}
