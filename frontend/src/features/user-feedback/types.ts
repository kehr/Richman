// FeedbackRating is the user's thumbs signal as seen in the UI. The wire
// format used by the backend is "helpful" | "not_helpful" - translation
// happens inside submitFeedback() to keep UI code simple.
export type FeedbackRating = "up" | "down";

// SubmitFeedbackInput is the payload callers provide to submitFeedback.
// It mirrors the backend shape (see backend/internal/service/feedback/service.go
// CreateFeedbackInput): each feedback row points at a single rs_asset_analyses
// primary key via assetAnalysisId.
export interface SubmitFeedbackInput {
	assetAnalysisId: number;
	rating: FeedbackRating;
	// Optional free-text comment (<= 500 runes, enforced server-side).
	comment?: string;
}

// FeedbackDto is the server response after submitting feedback.
export interface FeedbackDto {
	feedbackId: number;
}
