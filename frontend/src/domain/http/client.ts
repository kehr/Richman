import { getToken } from "@/domain/auth/storage";

// API_V1_BASE is the canonical v1 prefix used by legacy endpoints.
// API_V2_BASE is the v2 prefix for new richson-backed endpoints.
// The host-only base (VITE_API_BASE) is kept separate so both versions
// share the same origin without duplicating the env fallback logic.
// An empty string is a valid value (relative path), so use ?? instead of ||
// to only fall back when the env var is genuinely unset.
const API_HOST = import.meta.env.VITE_API_BASE ?? "http://localhost:8080";
export const API_V1_BASE = `${API_HOST}/api/v1`;
export const API_V2_BASE = `${API_HOST}/api/v2`;

// API_BASE is kept for backward compatibility with call sites that bypass
// the standard helpers (e.g. multipart screenshot upload in portfolio/api.ts).
// New code must use API_V1_BASE or API_V2_BASE.
export const API_BASE = API_V1_BASE;

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

// handleResponse is a shared response handler that parses JSON and maps
// non-2xx responses to ApiError. Shared by all request variants.
async function handleResponse<T>(response: Response): Promise<T> {
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

// _request is the internal primitive. All public variants delegate here.
// It accepts a fully-qualified URL (including base + path) so callers
// can compose the base independently.
async function _request<T>(url: string, options?: RequestInit, includeAuth = true): Promise<T> {
	const token = includeAuth ? getToken() : null;
	const response = await fetch(url, {
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: `Bearer ${token}` } : {}),
			...options?.headers,
		},
		...options,
	});
	return handleResponse<T>(response);
}

// requestV1 sends an authenticated request to a v1 API path.
// All existing feature call sites should use this after migration from request().
export function requestV1<T>(url: string, options?: RequestInit): Promise<T> {
	return _request<T>(`${API_V1_BASE}${url}`, options, true);
}

// requestV2 sends an authenticated request to a v2 API path (richson).
// New features backed by richson use this variant.
export function requestV2<T>(url: string, options?: RequestInit): Promise<T> {
	return _request<T>(`${API_V2_BASE}${url}`, options, true);
}

// requestPublic sends an unauthenticated request to a v2 API path.
// Used by Market Overview and Asset Detail pages that are open to all visitors.
export function requestPublic<T>(url: string, options?: RequestInit): Promise<T> {
	return _request<T>(`${API_V2_BASE}${url}`, options, false);
}

// request is kept as an alias for requestV1 for backward compatibility
// during the migration window. Do not use for new code.
// @deprecated use requestV1 instead
export function request<T>(url: string, options?: RequestInit): Promise<T> {
	return requestV1<T>(url, options);
}
