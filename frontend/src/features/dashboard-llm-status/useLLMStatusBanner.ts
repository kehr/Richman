import { useSyncExternalStore } from "react";

// LLM_BANNER_DISMISS_STORAGE_KEY is the sessionStorage key the dashboard
// banner writes to when the user clicks the close X. The value is a flag
// that survives a single-tab session and is intentionally cleared by
// browser session end — PRD §Dashboard Banner forbids a "permanent
// dismiss" affordance because the underlying problem is still there.
export const LLM_BANNER_DISMISS_STORAGE_KEY = "llm-status-banner-dismissed";

const STORAGE_EVENT = "llm-status-banner-dismissed-change";

function readDismissedFlag(): boolean {
	try {
		return sessionStorage.getItem(LLM_BANNER_DISMISS_STORAGE_KEY) === "1";
	} catch {
		// Storage may be disabled in private browsing mode; default to
		// "not dismissed" so the banner still surfaces.
		return false;
	}
}

function subscribeToDismissedFlag(listener: () => void): () => void {
	// Custom event provides a cross-hook notification channel when one
	// consumer toggles the flag; storage events alone do not fire in the
	// same tab that caused the mutation.
	window.addEventListener(STORAGE_EVENT, listener);
	return () => window.removeEventListener(STORAGE_EVENT, listener);
}

function writeDismissedFlag(value: boolean): void {
	try {
		if (value) {
			sessionStorage.setItem(LLM_BANNER_DISMISS_STORAGE_KEY, "1");
		} else {
			sessionStorage.removeItem(LLM_BANNER_DISMISS_STORAGE_KEY);
		}
	} catch {
		// Ignore: storage may be disabled or over quota.
	}
	try {
		window.dispatchEvent(new CustomEvent(STORAGE_EVENT));
	} catch {
		// CustomEvent is universally supported; try/catch is defensive.
	}
}

// useLLMStatusBanner returns whether the banner has been dismissed for
// the current session and a setter. The hook uses useSyncExternalStore
// so multiple mounted banners stay in sync with each other and with any
// imperative dismissal triggered from elsewhere.
export function useLLMStatusBanner(): {
	dismissed: boolean;
	dismiss: () => void;
	undismiss: () => void;
} {
	const dismissed = useSyncExternalStore(subscribeToDismissedFlag, readDismissedFlag, () => false);
	return {
		dismissed,
		dismiss: () => writeDismissedFlag(true),
		undismiss: () => writeDismissedFlag(false),
	};
}
