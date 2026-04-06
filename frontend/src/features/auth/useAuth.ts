import { clearAuth, setToken, setUser } from "@/domain/auth/storage";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { login, register } from "./api";
import type { LoginInput, RegisterInput } from "./api";

export function useLogin() {
	const navigate = useNavigate();
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (input: LoginInput) => login(input),
		onSuccess: (res) => {
			setToken(res.data.token);
			setUser(res.data.user);
			queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			navigate("/dashboard", { replace: true });
		},
	});
}

export function useRegister() {
	const navigate = useNavigate();
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: (input: RegisterInput) => register(input),
		onSuccess: (res) => {
			setToken(res.data.token);
			setUser(res.data.user);
			queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			navigate("/dashboard", { replace: true });
		},
	});
}

export function useLogout() {
	const navigate = useNavigate();
	const queryClient = useQueryClient();

	return () => {
		clearAuth();
		queryClient.clear();
		navigate("/login", { replace: true });
	};
}
