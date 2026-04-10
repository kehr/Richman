import { StorageKeys, storageRemove } from "@/domain/storage/local-storage";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { resetOnboarding } from "./api";
import { ONBOARDING_STATUS_QUERY_KEY } from "./use-onboarding-status";
import { USER_SETTINGS_QUERY_KEY } from "./use-user-settings";

// useResetOnboarding clears both onboarding timestamps server side. The
// backend returns HTTP 403 in production builds, so this hook is wired to
// the dev-only "Re-run onboarding" action on the Settings page (and to the
// production Settings re-entry CTA when the user has skipped).
//
// onSuccess has to undo every piece of client-side state that was left
// behind by a prior onboarding run so the next run starts fresh:
//   1. sessionStorage draft — the wizard persists in-flight answers here
//   2. localStorage nudge-dismissed flag — Dashboard's skipped-nudge hides
//      itself via this key, and a reset should make the nudge reappear if
//      the user ever skips again
//   3. onboarding-status cache — the guard keys off this
//   4. auth/me cache — the User object carries onboardingCompletedAt /
//      onboardingSkippedAt and must refetch
//   5. user-settings cache — mirrors the completion flag in its DTO
//
// Storage writes are wrapped in try/catch because private-mode / disabled
// storage must not block the mutation's success path.
export function useResetOnboarding() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: () => resetOnboarding(),
		onSuccess: async () => {
			try {
				sessionStorage.removeItem("richman_onboarding_draft");
			} catch {
				// sessionStorage may be disabled (private mode); fail silent.
			}
			storageRemove(StorageKeys.onboardingNudgeDismissed);
			await queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
			await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			await queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
		},
	});
}
