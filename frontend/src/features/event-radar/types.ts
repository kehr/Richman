// Types for the Event Radar feature (SS3.3 in frontend-v2-trd.md).

export interface EventDto {
	id: string;
	date: string;
	title: string;
	impactLevel: "high" | "medium" | "low";
	goldDirection: "bullish" | "bearish" | "neutral" | null;
	polymarketProbability: number | null;
	polymarketChange24h: number | null;
}

export interface EventRadarDto {
	events: EventDto[];
	updatedAt: string;
}
