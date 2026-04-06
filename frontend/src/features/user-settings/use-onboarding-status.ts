import { useQuery } from "@tanstack/react-query";
import { getOnboardingStatus } from "./api";
import type { OnboardingStatus } from "./types";

export const ONBOARDING_STATUS_QUERY_KEY = ["onboarding-status"] as const;

// useOnboardingStatus is consumed by OnboardingGuard to decide whether to
// force-redirect the user into the /onboarding/* branch. A short staleTime
// keeps the flow snappy after the "Mark complete" mutation fires.
export function useOnboardingStatus() {
	return useQuery<OnboardingStatus>({
		queryKey: ONBOARDING_STATUS_QUERY_KEY,
		queryFn: async () => {
			const res = await getOnboardingStatus();
			return res.data;
		},
		staleTime: 10_000,
	});
}
