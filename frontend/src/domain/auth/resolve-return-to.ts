// DEFAULT_AFTER_AUTH is where the user lands when no valid `?returnTo=` is
// present on the URL after a login or register success.
export const DEFAULT_AFTER_AUTH = "/dashboard";

// resolveReturnTo validates a raw `?returnTo=` query string value and either
// returns the safe relative path or falls back to /dashboard. Rejection
// rules (security critical, see Step 20 spec):
//   1. Must exist and be a non-empty string
//   2. Must start with a single "/" (not "//", not "/\\")
//   3. Must NOT contain "://" anywhere (prevents "/https://evil")
//   4. Must NOT contain ASCII control characters
// Anything else collapses to /dashboard so an attacker cannot bounce a
// freshly-authenticated user to an external host.
export function resolveReturnTo(raw: string | null): string {
	if (raw == null || raw.length === 0) {
		return DEFAULT_AFTER_AUTH;
	}
	if (raw[0] !== "/") {
		return DEFAULT_AFTER_AUTH;
	}
	if (raw[1] === "/" || raw[1] === "\\") {
		return DEFAULT_AFTER_AUTH;
	}
	if (raw.includes("://")) {
		return DEFAULT_AFTER_AUTH;
	}
	// Reject any ASCII control characters (0x00-0x1f, 0x7f) without using a
	// regex literal so Biome's noControlCharactersInRegex stays happy.
	for (let i = 0; i < raw.length; i++) {
		const code = raw.charCodeAt(i);
		if (code <= 0x1f || code === 0x7f) {
			return DEFAULT_AFTER_AUTH;
		}
	}
	return raw;
}
