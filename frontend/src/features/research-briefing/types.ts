// BriefingCardDto mirrors the backend BriefingCardDTO emitted by
// GET /api/v2/briefing (see backend/internal/service/briefing/service.go).
// Field shapes and nullability follow the Go struct exactly; decimal-typed
// values arrive as strings because pgx/decimal serialises them that way.

export interface BriefingCardDto {
	holdingId: number;
	assetCode: string;
	assetName: string;
	assetType: string;
	// Cost basis and position ratio are pgx decimals serialised as strings.
	costPrice: string;
	positionRatio: string;
	quantity: string;
	// Current price observed at the latest analysis (from rs_asset_analyses).
	currentPrice?: number;
	// Unrealized PnL in percent; null until an analysis supplies current price.
	pnlPercent: number | null;
	// Richson composite score 0-100 (null before first analysis).
	overallScore: number | null;
	// Richson signal label (strong_bullish | moderate_bullish | neutral |
	// moderate_bearish | strong_bearish), null before first analysis.
	signalLevel: string | null;
	// Delta vs previous analysis snapshot (null when no prior analysis).
	scoreDelta: number | null;
	// Last 90-day composite scores (oldest first). Empty when no history exists.
	sparklineScores: number[];
	// Most recent decision card id (null when no card has been generated).
	latestCardId: number | null;
	// Action level copied from the latest decision card (1-5).
	actionLevel: number | null;
	// Concentration classification for the asset type (red|orange|blue|green).
	concentrationLevel: string;
	concentrationMessage: string;
	// ISO timestamp of the latest analysis (null before first analysis).
	analyzedAt: string | null;
	// Primary key of the rs_asset_analyses row backing overallScore/signalLevel.
	// Sent with POST /feedback so the backend can associate the rating with the
	// exact analysis. Null when no analysis exists yet - disable feedback in UI.
	assetAnalysisId: number | null;
	// One-sentence attribution for today's score change (shown when |delta| >= 5).
	changeAttribution: string | null;
	// Conflict warning text emitted by richson when dimensions disagree.
	conflictWarning: string | null;
	// Derived trend label: "bullish" | "bearish" | "neutral".
	direction: string;
}

// BriefingDto is the full payload returned by GET /api/v2/briefing.
export interface BriefingDto {
	cards: BriefingCardDto[];
	updatedAt: string;
}
