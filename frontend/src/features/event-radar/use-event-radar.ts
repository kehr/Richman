import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { fetchEventRadar } from "./api";
import type { EventRadarDto } from "./types";

// EVENT_RADAR_KEY is the stable query key for event radar data.
const EVENT_RADAR_KEY = ["events", "radar"] as const;

// useEventRadar fetches upcoming macro events for the next 7 days.
// staleTime is 15 minutes — events are not updated frequently.
// retry is 0: the backend radar path already has a 20s budget and
// richson runs a scheduler warmup every 10min, so a timeout here means
// upstream is genuinely unreachable. Retrying would multiply the wait
// without improving success odds and degrade the perceived load time.
// placeholderData keeps the prior frame visible during background
// refetches so the panel does not flash a spinner every 15 minutes.
export function useEventRadar() {
	return useQuery<EventRadarDto>({
		queryKey: EVENT_RADAR_KEY,
		queryFn: fetchEventRadar,
		staleTime: 15 * 60 * 1000,
		retry: 0,
		placeholderData: keepPreviousData,
	});
}
