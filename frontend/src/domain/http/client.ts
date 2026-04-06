import { getToken } from "@/domain/auth/storage";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

export class ApiError extends Error {
	status: number;
	code: string;

	constructor(status: number, code: string, message: string) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
	}
}

export async function request<T>(url: string, options?: RequestInit): Promise<T> {
	const token = getToken();
	const response = await fetch(`${API_BASE}${url}`, {
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: `Bearer ${token}` } : {}),
		},
		...options,
	});

	if (!response.ok) {
		const body = await response.json().catch(() => ({}));
		throw new ApiError(
			response.status,
			body?.error?.code || "UNKNOWN",
			body?.error?.message || response.statusText,
		);
	}

	return response.json();
}
