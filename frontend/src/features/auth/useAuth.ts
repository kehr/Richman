import { clearAuth, setToken, setUser } from "@/domain/auth/storage";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { login, register } from "./api";
import type { LoginInput, RegisterInput } from "./api";

// useLogin runs the login mutation and, on success, persists the token and
// navigates to a target path. Callers may pass a `redirectTo` override (e.g.
// LoginPage parses ?returnTo= and forwards a validated relative path here);
// when omitted the user lands on /dashboard.
export function useLogin(options?: { redirectTo?: string }) {
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const target = options?.redirectTo ?? "/dashboard";

	return useMutation({
		mutationFn: (input: LoginInput) => login(input),
		onSuccess: (res) => {
			setToken(res.data.token);
			setUser(res.data.user);
			queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
			navigate(target, { replace: true });
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
