import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getUserSettings, patchUserSettings } from "./api";
import type { PatchUserSettings, UserSettings } from "./types";

export const USER_SETTINGS_QUERY_KEY = ["user-settings"] as const;

// useUserSettings loads the authenticated user's settings snapshot. The short
// staleTime is enough to dedupe the bursty requests that happen when several
// components mount at once (dashboard + decision card list) while still
// refetching after navigation.
export function useUserSettings() {
	return useQuery<UserSettings>({
		queryKey: USER_SETTINGS_QUERY_KEY,
		queryFn: async () => {
			const res = await getUserSettings();
			return res.data;
		},
		staleTime: 10_000,
	});
}

// usePatchUserSettings returns a mutation that sends a sparse PATCH. On
// success we invalidate the settings cache so every consumer (useMoney
// included) re-reads the latest values.
export function usePatchUserSettings() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (patch: PatchUserSettings) => patchUserSettings(patch),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
		},
	});
}
