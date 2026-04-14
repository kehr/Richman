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
	// Source metadata: present when the upstream data source exposes a stable
	// landing page (FRED release page / Polymarket event page). Null for items
	// that have no such page yet.
	sourceUrl?: string | null;
	sourceName?: string | null;
	releaseId?: number | null;
}

export interface EventRadarDto {
	events: EventDto[];
	updatedAt: string;
}
