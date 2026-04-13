import { useQuery } from "@tanstack/react-query";
import { fetchDemoPlan } from "./api";
import type { DemoPlanDto } from "./types";

export function useDemoPlan(code: string, enabled = true) {
	return useQuery<DemoPlanDto>({
		queryKey: ["asset-demo-plan", code] as const,
		queryFn: async () => {
			const res = await fetchDemoPlan(code);
			return res.data;
		},
		enabled: enabled && !!code,
		staleTime: 10 * 60_000,
	});
}
