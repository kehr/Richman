import { useMutation, useQueryClient } from "@tanstack/react-query";
import { patchUserSettings } from "./api";
import type { RiskPreference } from "./types";
import { USER_SETTINGS_QUERY_KEY } from "./use-user-settings";

// usePatchRiskPreference sends a sparse PATCH that updates only the
// riskPreference field, then invalidates the settings cache.
export function usePatchRiskPreference() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (riskPreference: RiskPreference) => patchUserSettings({ riskPreference }),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
		},
	});
}
