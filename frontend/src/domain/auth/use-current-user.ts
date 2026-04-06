"use client";

import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import { useQuery } from "@tanstack/react-query";
import { getToken } from "./storage";
import type { User } from "./types";

export function useCurrentUser() {
	return useQuery({
		queryKey: ["auth", "me"],
		queryFn: () => request<ApiResponse<User>>("/auth/me"),
		retry: false,
		staleTime: 5 * 60 * 1000,
		enabled: !!getToken(),
	});
}
