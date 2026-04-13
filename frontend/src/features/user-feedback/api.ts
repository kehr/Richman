import { requestV2 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { FeedbackDto, SubmitFeedbackInput } from "./types";

// submitFeedback sends a thumbs up/down signal to the backend.
// The endpoint records the rating and associates it with the authenticated user.
export function submitFeedback(input: SubmitFeedbackInput) {
	return requestV2<ApiResponse<FeedbackDto>>("/feedback", {
		method: "POST",
		body: JSON.stringify(input),
	});
}
