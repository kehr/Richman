import { useQuery } from "@tanstack/react-query";
import { getMyCodes } from "./api";
import type { MyCodesResponse } from "./types";

export const MY_CODES_QUERY_KEY = ["invite", "my-codes"] as const;

// useMyCodesQuery fetches the authenticated user's personal invite codes and
// unlock progress. staleTime is 60 s — codes rarely change within a session.
export function useMyCodesQuery() {
	return useQuery<MyCodesResponse>({
		queryKey: MY_CODES_QUERY_KEY,
		queryFn: async () => {
			const res = await getMyCodes();
			return res.data;
		},
		staleTime: 60_000,
	});
}
