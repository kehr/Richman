import type { User } from "@/domain/auth/types";
import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";

export interface LoginInput {
	email: string;
	password: string;
}

export interface RegisterInput {
	email: string;
	password: string;
	inviteCode: string;
}

export interface AuthResponse {
	token: string;
	user: User;
}

export function login(input: LoginInput) {
	return request<ApiResponse<AuthResponse>>("/auth/login", {
		method: "POST",
		body: JSON.stringify(input),
	});
}

export function register(input: RegisterInput) {
	return request<ApiResponse<AuthResponse>>("/auth/register", {
		method: "POST",
		body: JSON.stringify(input),
	});
}
