import { requestPublic } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { EventRadarDto } from "./types";

// fetchEventRadar loads the upcoming macro events for the next 7 days.
// Uses requestPublic — no JWT required (public page).
export async function fetchEventRadar(): Promise<EventRadarDto> {
	const res = await requestPublic<ApiResponse<EventRadarDto>>("/events/radar");
	return res.data;
}
