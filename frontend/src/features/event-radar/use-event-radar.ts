import { useQuery } from "@tanstack/react-query";
import { fetchEventRadar } from "./api";
import type { EventRadarDto } from "./types";

// EVENT_RADAR_KEY is the stable query key for event radar data.
const EVENT_RADAR_KEY = ["events", "radar"] as const;

// useEventRadar fetches upcoming macro events for the next 7 days.
// staleTime is 15 minutes — events are not updated frequently.
// retry is 2 so that a transient richson 503 gets two more attempts
// before the section shows a retry prompt (G3.9).
export function useEventRadar() {
	return useQuery<EventRadarDto>({
		queryKey: EVENT_RADAR_KEY,
		queryFn: fetchEventRadar,
		staleTime: 15 * 60 * 1000,
		retry: 2,
	});
}
