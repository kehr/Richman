import { requestV1 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

// Frequency controls how often analysis is triggered for a holding or market.
// null means "inherit from the parent level" (holding inherits from global /
// market schedule, market inherits from global settings).
export type Frequency =
	| "every_window"
	| "daily"
	| "every_2_days"
	| "every_3_days"
	| "weekly"
	| "custom"
	| null;

// WindowPreference selects which market window (pre/post/both) to trigger
// analysis for. null means "follow the market's natural window(s)".
export type WindowPreference = "pre" | "post" | "both" | null;

// WindowDTO describes a single market window's schedule configuration.
export interface WindowDTO {
	enabled: boolean;
	time: string; // "HH:MM" local time
	isCustom: boolean;
}

// MarketScheduleDTO holds per-market frequency and window configuration.
// frequency/frequencyDays override the global defaults for this market only.
export interface MarketScheduleDTO {
	frequency: Frequency;
	// frequencyDays is absent (undefined) when the backend omits it via omitempty.
	frequencyDays: number | null | undefined;
	preWindow: WindowDTO;
	postWindow: WindowDTO;
}

// ScheduleSettingsDTO is the top-level schedule configuration returned and
// accepted by GET/PUT /api/v1/settings/schedule.
export interface ScheduleSettingsDTO {
	globalFrequency: Exclude<Frequency, null>;
	// globalFrequencyDays is absent (undefined) when the backend omits it via omitempty.
	globalFrequencyDays: number | null | undefined;
	markets: {
		a_share: MarketScheduleDTO;
		us_stock: MarketScheduleDTO;
	};
}

// HoldingScheduleDTO is the per-holding schedule override returned and
// accepted by GET/PUT /api/v1/holdings/:id/schedule.
export interface HoldingScheduleDTO {
	holdingId: number;
	frequency: Frequency;
	// frequencyDays is absent (undefined) when the backend omits it via omitempty.
	frequencyDays: number | null | undefined;
	window: WindowPreference;
	nextAnalysisAt: string | null; // ISO 8601
}

// UpdateHoldingScheduleInput is the subset of fields accepted by the
// PUT /api/v1/holdings/:id/schedule body. All fields are nullable to allow
// "reset to inherit" semantics.
export type UpdateHoldingScheduleInput = Pick<
	HoldingScheduleDTO,
	"frequency" | "frequencyDays" | "window"
>;

// fetchScheduleSettings loads the global schedule configuration.
export function fetchScheduleSettings(): Promise<ApiResponse<ScheduleSettingsDTO>> {
	return requestV1<ApiResponse<ScheduleSettingsDTO>>("/settings/schedule");
}

// updateScheduleSettings persists a new global schedule configuration.
export function updateScheduleSettings(
	data: ScheduleSettingsDTO,
): Promise<ApiResponse<ScheduleSettingsDTO>> {
	return requestV1<ApiResponse<ScheduleSettingsDTO>>("/settings/schedule", {
		method: "PUT",
		body: JSON.stringify(data),
	});
}

// fetchHoldingSchedule loads the per-holding schedule override for a single
// holding identified by holdingId.
export function fetchHoldingSchedule(holdingId: number): Promise<ApiResponse<HoldingScheduleDTO>> {
	return requestV1<ApiResponse<HoldingScheduleDTO>>(`/holdings/${holdingId}/schedule`);
}

// updateHoldingSchedule persists per-holding schedule overrides. Pass null for
// any field to clear the override and inherit from the global/market schedule.
export function updateHoldingSchedule(
	holdingId: number,
	data: UpdateHoldingScheduleInput,
): Promise<ApiResponse<HoldingScheduleDTO>> {
	return requestV1<ApiResponse<HoldingScheduleDTO>>(`/holdings/${holdingId}/schedule`, {
		method: "PUT",
		body: JSON.stringify(data),
	});
}
