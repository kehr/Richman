import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markOnboardingCompleted } from "./api";
import { ONBOARDING_STATUS_QUERY_KEY } from "./use-onboarding-status";
import { USER_SETTINGS_QUERY_KEY } from "./use-user-settings";

// useMarkOnboardingCompleted finalises the onboarding flow. On success it
// invalidates both the onboarding-status cache (so OnboardingGuard releases
// the redirect) and user-settings (whose DTO mirrors the completion flag).
export function useMarkOnboardingCompleted() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => markOnboardingCompleted(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
		},
	});
}
