import { useQuery } from "@tanstack/react-query";
import { getMyInvites } from "./api";
import type { MyInvitesResponse } from "./types";

export const MY_INVITES_QUERY_KEY = ["invite", "my-invites"] as const;

// useMyInvitesQuery fetches the list of users invited by the authenticated user.
export function useMyInvitesQuery() {
	return useQuery<MyInvitesResponse>({
		queryKey: MY_INVITES_QUERY_KEY,
		queryFn: async () => {
			const res = await getMyInvites();
			return res.data;
		},
		staleTime: 60_000,
	});
}
