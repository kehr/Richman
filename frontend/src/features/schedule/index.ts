// Public barrel for the schedule feature. Pages must consume the feature
// exclusively through this entry point.

export {
	fetchScheduleSettings,
	fetchHoldingSchedule,
	updateScheduleSettings,
	updateHoldingSchedule,
} from "./api";

export type {
	Frequency,
	WindowPreference,
	WindowDTO,
	MarketScheduleDTO,
	ScheduleSettingsDTO,
	HoldingScheduleDTO,
	UpdateHoldingScheduleInput,
} from "./api";

export {
	SCHEDULE_SETTINGS_QUERY_KEY,
	HOLDING_SCHEDULE_BASE_KEY,
	holdingScheduleQueryKey,
	useScheduleSettings,
	useUpdateScheduleSettings,
	useHoldingSchedule,
	useUpdateHoldingSchedule,
} from "./useSchedule";
