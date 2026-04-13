import { useMutation } from "@tanstack/react-query";
import { submitFeedback } from "./api";
import type { SubmitFeedbackInput } from "./types";

// useSubmitFeedback provides a mutation for submitting user thumbs feedback.
// Callers can track isPending to show optimistic UI on the feedback buttons.
export function useSubmitFeedback() {
	return useMutation({
		mutationFn: (input: SubmitFeedbackInput) => submitFeedback(input),
	});
}
