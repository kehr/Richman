import { useMutation } from "@tanstack/react-query";
import { triggerHoldingAnalysis } from "./api";
import type { TriggerAnalysisResponseDto } from "./types";

export function useTriggerHoldingAnalysis() {
	return useMutation<TriggerAnalysisResponseDto, Error, number>({
		mutationFn: async (holdingId: number) => {
			const res = await triggerHoldingAnalysis(holdingId);
			return res.data;
		},
	});
}
