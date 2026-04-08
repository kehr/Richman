import { useMutation, useQueryClient } from "@tanstack/react-query";
import { skipOnboarding } from "./api";
import { ONBOARDING_STATUS_QUERY_KEY } from "./use-onboarding-status";

// useSkipOnboarding stamps onboarding_skipped_at server side so the user
// lands on the main app shell with a persistent re-entry nudge. On success
// we wipe the sessionStorage wizard draft (so a future re-entry starts
// clean), invalidate both the onboarding-status cache and the auth/me
// snapshot (which carries onboardingSkippedAt on the User object), and
// refetch the onboarding status synchronously — Dashboard's nudge depends
// on the refreshed status being visible before the user navigates away,
// otherwise the guard + nudge can race.
export function useSkipOnboarding() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => skipOnboarding(),
		onSuccess: async () => {
			try {
				sessionStorage.removeItem("richman_onboarding_draft");
			} catch {
				// sessionStorage may be disabled (private mode); fail silent.
			}
			await queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
			await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			await queryClient.refetchQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
		},
	});
}
