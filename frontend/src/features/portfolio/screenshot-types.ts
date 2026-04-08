// Screenshot recognition response shapes returned by
// POST /api/v1/portfolio/import-screenshot. Mirrors the Go service in
// backend/internal/service/screenshot/service.go (RecognizeResponse).
//
// Confidence thresholds map to backend ConfidenceHigh / ConfidenceLow:
//   >= 0.85 -> trustworthy auto-fill
//   >= 0.60 -> usable but should be highlighted for manual review
//   <  0.60 -> low confidence, render blank with red border

export const CONFIDENCE_HIGH = 0.85;
export const CONFIDENCE_LOW = 0.6;

export type RecognizeOverallStatus = "ok" | "degraded" | "failed";

export interface RecognizedField {
	value: string;
	confidence: number;
}

export interface RecognizedHolding {
	assetName: RecognizedField;
	assetCode: RecognizedField;
	costPrice: RecognizedField;
	positionPct: RecognizedField;
	assetTypeGuess: string;
}

export interface RecognizeResponse {
	holdings: RecognizedHolding[];
	overallStatus: RecognizeOverallStatus;
	warning?: string;
}

// EditableRecognizedHolding is the client-side row state used by the
// dual-pane confirm UI. Field values are normalized to numbers / strings
// the holding API understands and a per-row id is added so the table can
// react to add/remove without remounting other rows.
export interface EditableRecognizedHolding {
	rowId: string;
	assetName: string;
	assetNameConfidence: number;
	assetCode: string;
	assetCodeConfidence: number;
	costPrice: number | null;
	costPriceConfidence: number;
	positionRatio: number | null;
	positionRatioConfidence: number;
	assetType: string;
	selected: boolean;
}
