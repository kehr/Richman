// Centralized localStorage key registry. All keys in one place prevents
// typos and makes it easy to audit what the app persists.
// All keys use the "richman_" prefix to avoid collisions with other apps
// that may share the same origin.
export const StorageKeys = {
	authToken: "richman_auth_token",
	authUser: "richman_auth_user",
	themeMode: "richman_theme_mode",
	lastAnalysisTaskId: "richman_last_task_id",
	briefingViewMode: "richman_briefing_view_mode",
	disclaimerConfirmed: "richman_disclaimer_confirmed",
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

// migrateStorageKeys copies values from legacy unprefixed keys to the new
// richman_-prefixed keys and then removes the old keys. Called once at app
// startup. Safe to call multiple times (idempotent: old keys absent after
// first run, nothing to migrate on subsequent calls).
export function migrateStorageKeys(): void {
	if (typeof window === "undefined") return;

	const migrations: Array<{ oldKey: string; newKey: string }> = [
		{ oldKey: "auth_token", newKey: StorageKeys.authToken },
		{ oldKey: "auth_user", newKey: StorageKeys.authUser },
		{ oldKey: "theme_mode", newKey: StorageKeys.themeMode },
		// onboarding_nudge_dismissed was already prefixed; remove the old key
		// if present so it does not linger in storage.
		{ oldKey: "richman_onboarding_nudge_dismissed", newKey: "" },
	];

	for (const { oldKey, newKey } of migrations) {
		try {
			const old = localStorage.getItem(oldKey);
			if (old === null) continue;
			// Copy to new key only when a new key is specified and not already set.
			if (newKey && localStorage.getItem(newKey) === null) {
				localStorage.setItem(newKey, old);
			}
			localStorage.removeItem(oldKey);
		} catch {
			// Storage unavailable or quota exceeded — skip silently.
		}
	}
}
