import { useOnboardingStatus, useUserSettings } from "@/features/user-settings";
import {
	type ReactNode,
	createContext,
	useCallback,
	useContext,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";

// OnboardingState is the sessionStorage-persisted wizard draft that the four
// onboarding pages share. It is intentionally decoupled from server state:
// `categories` is mirrored from user settings on first load (and on resume of
// a returning user) but subsequent edits only live client-side until the
// relevant mutation fires in step 12. `holdingDraft` is pure client state
// assembled across step 13 form inputs and only posted in step 13's submit.
// `reachedStep` is a monotonically-increasing watermark used by
// `useOnboardingNav.jumpTo` to prevent forward jumps beyond progress made so
// far. `analysisFired` records whether step 14 has already dispatched its
// one-shot analysis trigger; it is load-bearing for survival of a refresh
// inside step 14.
export interface HoldingDraft {
	mode: "quick" | "detail" | "screenshot";
	assetCode?: string;
	assetName?: string;
	assetType?: string;
	costPrice?: number;
	positionRatio?: number;
	quantity?: number;
}

export interface OnboardingState {
	categories: string[];
	holdingDraft: HoldingDraft;
	// reachedStep is the max onboarding step the user has unlocked. Step 4
	// was added by the LLM degraded contract work (LLM consent) which
	// pushed the final "first analysis" step to position 5. The literal
	// union intentionally mirrors STEP_PATHS in use-onboarding-nav.
	reachedStep: 1 | 2 | 3 | 4 | 5;
	analysisFired: boolean;
}

export const DEFAULT_ONBOARDING_STATE: OnboardingState = {
	categories: [],
	holdingDraft: { mode: "quick" },
	reachedStep: 1,
	analysisFired: false,
};

export const ONBOARDING_DRAFT_STORAGE_KEY = "richman_onboarding_draft";

const STORAGE_WRITE_DEBOUNCE_MS = 500;

interface OnboardingStateContextValue {
	state: OnboardingState;
	update: (patch: Partial<OnboardingState>) => void;
	updateHoldingDraft: (patch: Partial<HoldingDraft>) => void;
	clear: () => void;
}

export const OnboardingStateContext = createContext<OnboardingStateContextValue | null>(null);

// readInitialState resolves the bootstrap value for the provider's useState
// initializer. It is called exactly once per mount (lazy init via the function
// form of useState) and guarantees a fallback to DEFAULT_ONBOARDING_STATE for
// every failure mode: storage disabled (private mode), JSON corruption, or a
// missing key.
function readInitialState(): OnboardingState {
	try {
		const raw = sessionStorage.getItem(ONBOARDING_DRAFT_STORAGE_KEY);
		if (!raw) return DEFAULT_ONBOARDING_STATE;
		const parsed = JSON.parse(raw) as Partial<OnboardingState> | null;
		if (!parsed || typeof parsed !== "object") return DEFAULT_ONBOARDING_STATE;
		// Merge with DEFAULT_ONBOARDING_STATE so any partial or forward-compat
		// payload still yields a fully-shaped state tree. holdingDraft is a
		// nested object so it needs its own shallow merge.
		return {
			...DEFAULT_ONBOARDING_STATE,
			...parsed,
			holdingDraft: {
				...DEFAULT_ONBOARDING_STATE.holdingDraft,
				...(parsed.holdingDraft ?? {}),
			},
		};
	} catch {
		// sessionStorage unavailable or JSON.parse threw — fall back silently.
		return DEFAULT_ONBOARDING_STATE;
	}
}

// clearStorage wraps sessionStorage.removeItem in a try/catch. All callers
// treat storage as best-effort; failures must not break the flow.
function clearStorage() {
	try {
		sessionStorage.removeItem(ONBOARDING_DRAFT_STORAGE_KEY);
	} catch {
		// Ignore: storage may be disabled in private mode.
	}
}

interface OnboardingStateProviderProps {
	children: ReactNode;
}

// OnboardingStateProvider owns the wizard draft for the entire /onboarding/*
// branch. Mount it above the four onboarding pages (e.g. inside the route
// shell that wraps `<Outlet />`). Two server-side concerns are folded in:
//
//  1. Cross-tab pollution: if the onboarding-status query resolves with
//     completed=true or skipped=true (the user finished or skipped elsewhere)
//     any stale draft in sessionStorage is wiped and the in-memory state is
//     reset. This runs on every status change so a completion/skip event that
//     happens mid-session is also honored.
//
//  2. Returning-user categories: if user-settings arrives with a non-empty
//     categories array that differs from local state, adopt the server value
//     once. This handles re-entering onboarding after skipping with server
//     state already populated — otherwise the user would re-pick categories
//     they previously chose.
//
// Writes back to sessionStorage are debounced ~500ms to avoid thrashing
// storage on stagger animations or rapid toggles.
export function OnboardingStateProvider({ children }: OnboardingStateProviderProps) {
	const { data: onboardingStatus } = useOnboardingStatus();
	const { data: userSettings } = useUserSettings();

	const [state, setState] = useState<OnboardingState>(() => readInitialState());

	// Track whether we already adopted server categories so we do not fight the
	// user if they shrink the selection locally after the server load completes.
	const categoriesAdoptedRef = useRef(false);

	// Debounced write timer. A ref keeps the handle stable across renders so we
	// can clear the previous write before scheduling a new one.
	const writeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

	// Concern 1: cross-tab pollution cleanup. Runs on every status update so a
	// completion/skip that happens after mount is also handled.
	useEffect(() => {
		if (!onboardingStatus) return;
		if (onboardingStatus.completed || onboardingStatus.skipped) {
			clearStorage();
			setState((prev) => (prev === DEFAULT_ONBOARDING_STATE ? prev : DEFAULT_ONBOARDING_STATE));
		}
	}, [onboardingStatus]);

	// Concern 2: returning-user server categories adoption. Fires once, only if
	// the server has a non-empty list that differs from the current draft.
	useEffect(() => {
		if (categoriesAdoptedRef.current) return;
		if (!userSettings) return;
		const serverCategories = userSettings.categories ?? [];
		if (serverCategories.length === 0) return;
		const sameLength = serverCategories.length === state.categories.length;
		const sameSet = sameLength && serverCategories.every((c) => state.categories.includes(c));
		if (sameSet) {
			categoriesAdoptedRef.current = true;
			return;
		}
		categoriesAdoptedRef.current = true;
		setState((prev) => ({ ...prev, categories: [...serverCategories] }));
	}, [userSettings, state.categories]);

	// Cascade cleanup: if the user shrinks categories and the currently-drafted
	// holding's assetType is no longer in the selection, wipe the asset-related
	// fields so the user is not left with a dangling asset reference that can
	// not be submitted. Fires after every categories mutation.
	useEffect(() => {
		const { assetType } = state.holdingDraft;
		if (!assetType) return;
		if (state.categories.includes(assetType)) return;
		setState((prev) => ({
			...prev,
			holdingDraft: {
				...prev.holdingDraft,
				assetCode: undefined,
				assetName: undefined,
				assetType: undefined,
			},
		}));
	}, [state.categories, state.holdingDraft]);

	// Persist state to sessionStorage with a trailing-edge debounce. Writes are
	// best-effort: a thrown QuotaExceededError or a disabled storage backend
	// must not break the wizard.
	useEffect(() => {
		if (writeTimerRef.current !== null) {
			clearTimeout(writeTimerRef.current);
		}
		writeTimerRef.current = setTimeout(() => {
			try {
				sessionStorage.setItem(ONBOARDING_DRAFT_STORAGE_KEY, JSON.stringify(state));
			} catch {
				// Ignore: storage may be disabled or quota exceeded.
			}
			writeTimerRef.current = null;
		}, STORAGE_WRITE_DEBOUNCE_MS);
		return () => {
			if (writeTimerRef.current !== null) {
				clearTimeout(writeTimerRef.current);
				writeTimerRef.current = null;
			}
		};
	}, [state]);

	const update = useCallback((patch: Partial<OnboardingState>) => {
		setState((prev) => ({ ...prev, ...patch }));
	}, []);

	const updateHoldingDraft = useCallback((patch: Partial<HoldingDraft>) => {
		setState((prev) => ({
			...prev,
			holdingDraft: { ...prev.holdingDraft, ...patch },
		}));
	}, []);

	const clear = useCallback(() => {
		clearStorage();
		setState(DEFAULT_ONBOARDING_STATE);
	}, []);

	const value = useMemo<OnboardingStateContextValue>(
		() => ({ state, update, updateHoldingDraft, clear }),
		[state, update, updateHoldingDraft, clear],
	);

	return (
		<OnboardingStateContext.Provider value={value}>{children}</OnboardingStateContext.Provider>
	);
}

// useOnboardingState returns the provider's context value. It throws when
// used outside `OnboardingStateProvider` so misuse is caught at the first
// render rather than silently producing undefined state.
export function useOnboardingState(): OnboardingStateContextValue {
	const ctx = useContext(OnboardingStateContext);
	if (!ctx) {
		throw new Error("useOnboardingState must be used inside OnboardingStateProvider");
	}
	return ctx;
}
