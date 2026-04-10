// Centralized localStorage key registry. All keys in one place prevents
// typos and makes it easy to audit what the app persists.
export const StorageKeys = {
	authToken: "auth_token",
	authUser: "auth_user",
	themeMode: "theme_mode",
	onboardingNudgeDismissed: "richman_onboarding_nudge_dismissed",
	lastAnalysisTaskId: "richman_last_task_id",
} as const;

// storageGet reads and JSON-parses a value. Returns null when the key is
// absent, storage is unavailable, or the stored string is not valid JSON.
export function storageGet<T>(key: string): T | null {
	if (typeof window === "undefined") return null;
	try {
		const item = localStorage.getItem(key);
		if (item === null) return null;
		return JSON.parse(item) as T;
	} catch {
		return null;
	}
}

// storageSet JSON-serializes value and writes it. Silently no-ops when
// storage is unavailable (private mode, quota exceeded, etc.).
export function storageSet<T>(key: string, value: T): void {
	if (typeof window === "undefined") return;
	try {
		localStorage.setItem(key, JSON.stringify(value));
	} catch {}
}

// storageRemove deletes a key. Safe to call even if the key is absent.
export function storageRemove(key: string): void {
	if (typeof window === "undefined") return;
	try {
		localStorage.removeItem(key);
	} catch {}
}
