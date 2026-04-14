// Types for the Event Radar feature (SS3.3 in frontend-v2-trd.md).
// Field names mirror the richson GET /events/radar response; the Go backend
// proxies the payload without renaming. Keep this file in sync with
// richson/src/richson/schemas/events.py::EventItem.

export interface EventDto {
	date: string;
	title: string;
	category: string;
	impact: "high" | "medium" | "low";
	goldDirection: "bullish" | "bearish" | "neutral" | null;
	probability: number | null;
	probabilitySource: string | null;
	probabilityChange24h: number | null;
}

export interface EventRadarDto {
	events: EventDto[];
	updatedAt: string;
}
