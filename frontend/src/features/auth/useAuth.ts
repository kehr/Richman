"use client";

import { clearAuth, setToken, setUser } from "@/domain/auth/storage";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { login, register } from "./api";
import type { LoginInput, RegisterInput } from "./api";

export function useLogin() {
	const router = useRouter();
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (input: LoginInput) => login(input),
		onSuccess: (res) => {
			setToken(res.data.token);
			setUser(res.data.user);
			queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			router.replace("/dashboard");
		},
	});
}

export function useRegister() {
	const router = useRouter();
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (input: RegisterInput) => register(input),
		onSuccess: (res) => {
			setToken(res.data.token);
			setUser(res.data.user);
			queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			router.replace("/dashboard");
		},
	});
}

export function useLogout() {
	const router = useRouter();
	const queryClient = useQueryClient();

	return () => {
		clearAuth();
		queryClient.clear();
		router.replace("/login");
	};
}
