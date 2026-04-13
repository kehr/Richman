import { useMutation, useQueryClient } from "@tanstack/react-query";
import { resetOnboarding } from "./api";
import { ONBOARDING_STATUS_QUERY_KEY } from "./use-onboarding-status";
import { USER_SETTINGS_QUERY_KEY } from "./use-user-settings";

// useResetOnboarding clears both onboarding timestamps server side.
// Kept for backward compatibility; not exposed in UI as of v2.
export function useResetOnboarding() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => resetOnboarding(),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
			await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			await queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
		},
	});
}
