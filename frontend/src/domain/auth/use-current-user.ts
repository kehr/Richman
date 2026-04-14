import { requestV1 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import { useQuery } from "@tanstack/react-query";
import { getToken } from "./storage";
import type { User } from "./types";

// useCurrentUser fetches the authenticated user via /auth/me. The query
// unwraps the ApiResponse envelope via `select` so consumers receive a
// plain User (or undefined while loading), matching the convention used
// by every other query hook in the app.
export function useCurrentUser() {
	return useQuery({
		queryKey: ["auth", "me"],
		queryFn: () => requestV1<ApiResponse<User>>("/auth/me"),
		select: (res) => res.data,
		retry: false,
		staleTime: 5 * 60 * 1000,
		enabled: !!getToken(),
	});
}
