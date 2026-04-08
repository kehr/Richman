import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	deleteLLMSettings,
	getLLMSettings,
	postLLMConsent,
	postProbeLLMSettings,
	putLLMSettings,
} from "./api";
import type { LLMConsentRequest, LLMSettingsDTO, ProbeResultDTO, UpsertLLMRequest } from "./types";

// LLM_SETTINGS_QUERY_KEY is the stable query key for the user's LLM
// provider config. Exported so sibling pages (settings tab) and hooks can
// invalidate it after a mutation settles.
export const LLM_SETTINGS_QUERY_KEY = ["llm-settings"] as const;

// useLLMSettings fetches the current provider config. Short staleTime so
// the settings tab reflects health changes soon after a probe mutation.
export function useLLMSettings() {
	return useQuery<LLMSettingsDTO>({
		queryKey: LLM_SETTINGS_QUERY_KEY,
		queryFn: async () => {
			const res = await getLLMSettings();
			return res.data;
		},
		staleTime: 10_000,
	});
}

// useUpsertLLMSettings wraps the PUT handler. On success we invalidate
// both the llm-settings cache (so the three-state card re-renders) and
// the dashboard-summary cache (so the degraded-contract banner recomputes
// needsReanalysis against the new provider health).
export function useUpsertLLMSettings() {
	const queryClient = useQueryClient();
	return useMutation<LLMSettingsDTO, Error, UpsertLLMRequest>({
		mutationFn: async (body) => {
			const res = await putLLMSettings(body);
			return res.data;
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: LLM_SETTINGS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
		},
	});
}

// useDeleteLLMSettings wraps the DELETE handler.
export function useDeleteLLMSettings() {
	const queryClient = useQueryClient();
	return useMutation<void, Error, void>({
		mutationFn: async () => {
			await deleteLLMSettings();
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: LLM_SETTINGS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
		},
	});
}

// useProbeLLMSettings wraps the POST /probe handler. Unlike the other
// mutations it intentionally does not invalidate the llm-settings cache on
// its own — the probe endpoint on the backend already persists the latest
// health to the row, so callers invalidate once the mutation settles to
// keep the UX snappy.
export function useProbeLLMSettings() {
	const queryClient = useQueryClient();
	return useMutation<ProbeResultDTO, Error, void>({
		mutationFn: async () => {
			const res = await postProbeLLMSettings();
			return res.data;
		},
		onSettled: () => {
			queryClient.invalidateQueries({ queryKey: LLM_SETTINGS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
		},
	});
}

// useLLMConsent wraps the onboarding consent handler. On success we
// invalidate user-settings (which owns the onboarding status flag) and
// dashboard-summary so the banner can re-evaluate immediately after
// consent flips.
export function useLLMConsent() {
	const queryClient = useQueryClient();
	return useMutation<void, Error, LLMConsentRequest>({
		mutationFn: async (body) => {
			await postLLMConsent(body);
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["user-settings"] });
			queryClient.invalidateQueries({ queryKey: LLM_SETTINGS_QUERY_KEY });
			queryClient.invalidateQueries({ queryKey: ["dashboard-summary"] });
		},
	});
}
