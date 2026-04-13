// BriefingCard DTO returned by GET /api/v2/briefing.
// Each card corresponds to one holding and aggregates score, trend, and
// execution summary so the briefing page can render without reaching into
// the full decision-card payload.

export interface BriefingCardDto {
	holdingId: number;
	assetCode: string;
	assetName: string;
	assetType: string;
	// Overall score 0-100. Null when no analysis has run yet.
	overallScore: number | null;
	// Direction: bullish | bearish | neutral
	direction: string;
	// Last 90 days score history, oldest first.
	scoreTrend: Array<{ date: string; score: number }>;
	// Score change in the last analysis cycle (null if no prior analysis).
	scoreDelta: number | null;
	// One-sentence attribution for today's score change (shown when |delta| >= 5).
	changeAttribution: string | null;
	// Conflict warning text if dimensional conflicts exist.
	conflictWarning: string | null;
	// One-sentence summary of the primary execution scenario.
	actionSummary: string | null;
	// Holding position info.
	costPrice: number | null;
	positionRatio: number | null;
	// Unrealized PnL percentage (null when costPrice/currentPrice unavailable).
	unrealizedPnlPct: number | null;
	// Entry mode: "tag" | "quick" | "detail"
	entryMode: string;
}

// BriefingDto is the full payload returned by GET /api/v2/briefing.
export interface BriefingDto {
	cards: BriefingCardDto[];
	generatedAt: string;
}
