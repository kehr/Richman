// FeedbackTarget identifies what the user is rating.
// "briefing_card" is the primary target for Step 17.
export type FeedbackTarget = "briefing_card" | "analysis" | "execution_plan";

// FeedbackRating is the user's thumbs signal.
export type FeedbackRating = "up" | "down";

// SubmitFeedbackInput is the payload sent to POST /api/v2/feedback.
export interface SubmitFeedbackInput {
	target: FeedbackTarget;
	// targetId is the holdingId (for briefing_card) or other entity id.
	targetId: number;
	rating: FeedbackRating;
	// Optional free-text comment.
	comment?: string;
}

// FeedbackDto is the server response after submitting feedback.
export interface FeedbackDto {
	feedbackId: number;
	rating: FeedbackRating;
}
