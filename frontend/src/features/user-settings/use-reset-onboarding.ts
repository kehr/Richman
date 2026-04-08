import { useMutation, useQueryClient } from "@tanstack/react-query";
import { resetOnboarding } from "./api";
import { ONBOARDING_STATUS_QUERY_KEY } from "./use-onboarding-status";
import { USER_SETTINGS_QUERY_KEY } from "./use-user-settings";

// useResetOnboarding clears the onboarding completion timestamp. The backend
// returns HTTP 403 in production builds, so this hook is wired to the
// dev-only "Re-run onboarding" action on the Settings page.
export function useResetOnboarding() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => resetOnboarding(),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
		},
	});
}
