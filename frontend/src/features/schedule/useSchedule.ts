import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	fetchHoldingSchedule,
	fetchScheduleSettings,
	updateHoldingSchedule,
	updateScheduleSettings,
} from "./api";
import type { HoldingScheduleDTO, ScheduleSettingsDTO, UpdateHoldingScheduleInput } from "./api";

// SCHEDULE_SETTINGS_QUERY_KEY is the stable query key for the global schedule
// configuration. Exported so sibling components can manually invalidate.
export const SCHEDULE_SETTINGS_QUERY_KEY = ["schedule-settings"] as const;

// holdingScheduleQueryKey returns the query key for a specific holding's
// schedule override. Exporting the factory keeps invalidation consistent.
export function holdingScheduleQueryKey(holdingId: number) {
	return ["holding-schedule", holdingId] as const;
}

// HOLDING_SCHEDULE_BASE_KEY is the prefix used to invalidate all holding
// schedule queries at once after a mutation that may affect multiple holdings.
export const HOLDING_SCHEDULE_BASE_KEY = ["holding-schedule"] as const;

// useScheduleSettings fetches the global schedule configuration.
export function useScheduleSettings() {
	return useQuery<ScheduleSettingsDTO>({
		queryKey: SCHEDULE_SETTINGS_QUERY_KEY,
		queryFn: fetchScheduleSettings,
		staleTime: 30_000,
	});
}

// useUpdateScheduleSettings wraps the PUT /api/v1/settings/schedule handler.
// On success it invalidates the schedule-settings cache so any consumer
// immediately re-fetches the updated configuration.
export function useUpdateScheduleSettings() {
	const queryClient = useQueryClient();
	return useMutation<ScheduleSettingsDTO, Error, ScheduleSettingsDTO>({
		mutationFn: updateScheduleSettings,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: SCHEDULE_SETTINGS_QUERY_KEY });
		},
	});
}

// useHoldingSchedule fetches the per-holding schedule override identified by
// holdingId. The query is keyed per-holding so that each card caches
// independently.
export function useHoldingSchedule(holdingId: number) {
	return useQuery<HoldingScheduleDTO>({
		queryKey: holdingScheduleQueryKey(holdingId),
		queryFn: () => fetchHoldingSchedule(holdingId),
		staleTime: 30_000,
	});
}

// useUpdateHoldingSchedule wraps the PUT /api/v1/holdings/:id/schedule handler.
// On success it invalidates the specific holding's cache entry. It also
// invalidates the broader base key in case any list view aggregates schedules.
export function useUpdateHoldingSchedule() {
	const queryClient = useQueryClient();
	return useMutation<
		HoldingScheduleDTO,
		Error,
		{ holdingId: number; data: UpdateHoldingScheduleInput }
	>({
		mutationFn: ({ holdingId, data }) => updateHoldingSchedule(holdingId, data),
		onSuccess: (_result, variables) => {
			queryClient.invalidateQueries({
				queryKey: holdingScheduleQueryKey(variables.holdingId),
			});
			queryClient.invalidateQueries({
				queryKey: HOLDING_SCHEDULE_BASE_KEY,
			});
		},
	});
}
