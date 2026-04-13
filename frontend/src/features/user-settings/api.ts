import { requestV1 as request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { OnboardingStatus, PatchUserSettings, UserSettings } from "./types";

// getUserSettings loads the full settings snapshot for the authenticated user.
export function getUserSettings() {
	return request<ApiResponse<UserSettings>>("/user/settings");
}

// patchUserSettings sends a sparse update. Fields omitted from the payload are
// left unchanged on the server.
export function patchUserSettings(patch: PatchUserSettings) {
	return request<ApiResponse<UserSettings>>("/user/settings", {
		method: "PATCH",
		body: JSON.stringify(patch),
	});
}

// getOnboardingStatus returns whether the current user has finished the
// onboarding flow and, if so, the completion timestamp.
export function getOnboardingStatus() {
	return request<ApiResponse<OnboardingStatus>>("/onboarding");
}

// markOnboardingCompleted stamps the onboarding_completed_at column server
// side. Calling it twice is a no-op (the service layer is idempotent).
export function markOnboardingCompleted() {
	return request<ApiResponse<OnboardingStatus>>("/onboarding/complete", {
		method: "POST",
	});
}

// skipOnboarding stamps the onboarding_skipped_at column server side and
// atomically clears onboarding_completed_at. Wired to the wizard's "Skip
// for now" CTA — the user lands back on the dashboard with a persistent
// re-entry nudge.
export function skipOnboarding() {
	return request<ApiResponse<OnboardingStatus>>("/onboarding/skip", {
		method: "POST",
	});
}

// resetOnboarding clears the completion timestamp. The backend rejects this
// call with HTTP 403 in production builds; it is intended for the dev-only
// "Re-run onboarding" action in Settings.
export function resetOnboarding() {
	return request<ApiResponse<OnboardingStatus>>("/onboarding", {
		method: "DELETE",
	});
}
