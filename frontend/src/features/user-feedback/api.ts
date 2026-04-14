import { requestV2 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { FeedbackDto, SubmitFeedbackInput } from "./types";

// submitFeedback sends a thumbs up/down signal to the backend.
// The backend stores ratings as the enumeration "helpful" | "not_helpful"
// (see backend/internal/service/feedback/service.go), so we translate the
// UI-facing "up" | "down" here to keep component code unaware of the wire
// format.
export function submitFeedback(input: SubmitFeedbackInput) {
	const payload = {
		assetAnalysisId: input.assetAnalysisId,
		rating: input.rating === "up" ? "helpful" : "not_helpful",
		...(input.comment ? { comment: input.comment } : {}),
	};
	return requestV2<ApiResponse<FeedbackDto>>("/feedback", {
		method: "POST",
		body: JSON.stringify(payload),
	});
}
