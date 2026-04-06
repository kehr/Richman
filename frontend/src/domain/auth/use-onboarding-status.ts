// Temporary onboarding status hook.
// Step 11 will replace this with a real API-backed implementation located in
// features/user-settings. For now this placeholder always reports the user as
// having completed onboarding so existing development accounts are not trapped
// behind the OnboardingGuard before the backing endpoint is wired up.

interface OnboardingStatus {
	completed: boolean;
}

interface UseOnboardingStatusResult {
	data: OnboardingStatus | undefined;
	isLoading: boolean;
	error: Error | null;
}

export function useOnboardingStatus(): UseOnboardingStatusResult {
	return {
		data: { completed: true },
		isLoading: false,
		error: null,
	};
}
