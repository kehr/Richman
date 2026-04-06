const TOKEN_KEY = "auth_token";
const USER_KEY = "auth_user";

export function setToken(token: string): void {
	if (typeof window === "undefined") return;
	localStorage.setItem(TOKEN_KEY, token);
}

export function getToken(): string | null {
	if (typeof window === "undefined") return null;
	return localStorage.getItem(TOKEN_KEY);
}

export function removeToken(): void {
	if (typeof window === "undefined") return;
	localStorage.removeItem(TOKEN_KEY);
}

export function setUser(user: unknown): void {
	if (typeof window === "undefined") return;
	localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function getUser(): unknown | null {
	if (typeof window === "undefined") return null;
	const raw = localStorage.getItem(USER_KEY);
	if (!raw) return null;
	try {
		return JSON.parse(raw);
	} catch {
		return null;
	}
}

export function clearAuth(): void {
	removeToken();
	if (typeof window === "undefined") return;
	localStorage.removeItem(USER_KEY);
}
