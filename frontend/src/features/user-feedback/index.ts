// Public barrel for the user-feedback feature.
// Pages must consume this feature exclusively through this entry point.

export { submitFeedback } from "./api";
export { useSubmitFeedback } from "./use-submit-feedback";
export type { FeedbackRating, SubmitFeedbackInput, FeedbackDto } from "./types";
