import { useSkipOnboarding } from "@/features/user-settings";
import { useCallback, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { useOnboardingState } from "./state";

// OnboardingStep is the five-step wizard index. Keep as a literal union so
// array indexing, route table lookups, and the reachedStep watermark all
// share the exact same domain. Step 4 (llm-consent) was added by the LLM
// degraded contract work; the previous "first analysis" terminal step
// moved from position 4 to position 5.
export type OnboardingStep = 1 | 2 | 3 | 4 | 5;

// STEP_PATHS is the single source of truth for step ordering. Both the
// forward/backward navigation math and the current-path reverse lookup read
// this table, so edits (e.g. renaming a route) update both in one place.
export const STEP_PATHS: Record<OnboardingStep, string> = {
	1: "/onboarding/welcome",
	2: "/onboarding/categories",
	3: "/onboarding/first-holding",
	4: "/onboarding/llm-consent",
	5: "/onboarding/first-analysis",
};

// SHAKE_EVENT_NAME is dispatched on `window` when `next()` is called while
// `canGoNext` is false. OnboardingLayout subscribes to trigger a shake
// animation on the primary CTA. Centralizing the name keeps the producer
// and consumer in sync.
export const SHAKE_EVENT_NAME = "onboarding:shake";

export interface UseOnboardingNavReturn {
	currentStep: OnboardingStep;
	reachedStep: OnboardingStep;
	canGoNext: boolean;
	prev: () => void;
	next: () => Promise<void>;
	skip: () => Promise<void>;
	jumpTo: (step: OnboardingStep) => void;
	registerCanGoNext: (predicate: () => boolean) => () => void;
}

// pathToStep maps a location pathname to the corresponding step index. The
// scan walks the entries in descending step order so more-specific prefixes
// (e.g. /onboarding/first-holding) are matched before shorter ones. Defaults
// to 1 when no match is found so the header/indicator always has a sane value
// even on transient mismatches.
function pathToStep(pathname: string): OnboardingStep {
	const steps: OnboardingStep[] = [5, 4, 3, 2, 1];
	for (const step of steps) {
		if (pathname === STEP_PATHS[step] || pathname.startsWith(`${STEP_PATHS[step]}/`)) {
			return step;
		}
	}
	return 1;
}

// clampStep clamps an arbitrary number to the 1..5 OnboardingStep range. Used
// when advancing past step 5 (which would overflow) or retreating below 1.
function clampStep(n: number): OnboardingStep {
	if (n <= 1) return 1;
	if (n >= 5) return 5;
	return n as OnboardingStep;
}

// useOnboardingNav is the unified navigation surface shared by every
// onboarding page. The contract is documented in TRD §4.4:
//   - prev()/next() handle step math and navigate with { replace: true } so
//     the browser history stays linear for back-button users
//   - next() is gated by all currently-registered canGoNext predicates. When
//     the aggregate is false a CustomEvent is dispatched on window so
//     OnboardingLayout can shake the CTA, signaling the reason to the user
//     without coupling the hook to any specific UI library
//   - skip() delegates to the server-side skip mutation and only navigates
//     after the mutation resolves; failures propagate so the skip Modal can
//     toast and stay open
//   - jumpTo() enforces the reachedStep watermark: users can freely revisit
//     past steps but cannot leap ahead to steps they have not yet unlocked
//
// Predicate registration is per-mount: pages call registerCanGoNext in a
// useEffect and the returned cleanup removes the predicate on unmount. A
// version counter forces a re-aggregate whenever any page adds or removes
// a predicate.
export function useOnboardingNav(): UseOnboardingNavReturn {
	const navigate = useNavigate();
	const location = useLocation();
	const { state, update } = useOnboardingState();
	const skipMutation = useSkipOnboarding();

	const currentStep = pathToStep(location.pathname);
	const reachedStep = state.reachedStep;

	// Predicates live in a ref so register/unregister do not force the hook
	// consumer to re-render on every mutation. `predicateVersion` is a
	// monotonic counter bumped alongside each mutation; useMemo then re-runs
	// the aggregation deterministically. Deriving canGoNext via useMemo (as
	// opposed to useState + useEffect) means there is no stale frame where
	// `canGoNext=true` is read before the effect re-aggregates.
	const predicatesRef = useRef<Set<() => boolean>>(new Set());
	const [predicateVersion, setPredicateVersion] = useState(0);

	const canGoNext = useMemo(() => {
		// `predicateVersion` is the deliberate retrigger: the memo reads
		// predicatesRef.current which is untracked, so the counter is the only
		// reactive signal tying re-computation to registration changes.
		void predicateVersion;
		const predicates = predicatesRef.current;
		if (predicates.size === 0) return true;
		return Array.from(predicates).every((p) => {
			try {
				return p();
			} catch {
				// A throwing predicate is treated as false so a bug in one page
				// does not silently let the user advance.
				return false;
			}
		});
	}, [predicateVersion]);

	const registerCanGoNext = useCallback((predicate: () => boolean) => {
		predicatesRef.current.add(predicate);
		setPredicateVersion((v) => v + 1);
		return () => {
			predicatesRef.current.delete(predicate);
			setPredicateVersion((v) => v + 1);
		};
	}, []);

	const prev = useCallback(() => {
		if (currentStep <= 1) return;
		const target = clampStep(currentStep - 1);
		navigate(STEP_PATHS[target], { replace: true });
	}, [currentStep, navigate]);

	const next = useCallback(async () => {
		// Step 5 (FirstAnalysisPage) has no "next" — it owns its own completion
		// CTA. Early-return so keyboard ArrowRight at the terminal step is a
		// true no-op rather than a silent navigation overflow.
		if (currentStep >= 5) return;
		if (!canGoNext) {
			try {
				window.dispatchEvent(new CustomEvent(SHAKE_EVENT_NAME));
			} catch {
				// CustomEvent is universally supported in browsers and jsdom; the
				// try/catch is defensive for exotic runtimes.
			}
			return;
		}
		const target = clampStep(currentStep + 1);
		// Bump reachedStep only forward — never overwrite a higher watermark
		// the user already unlocked earlier in the same session.
		if (target > reachedStep) {
			update({ reachedStep: target });
		}
		navigate(STEP_PATHS[target], { replace: true });
	}, [currentStep, canGoNext, reachedStep, update, navigate]);

	const skip = useCallback(async () => {
		// Let errors bubble up so the skip Modal in step 10 can toast + keep
		// itself open. On success we head straight to the dashboard; the
		// mutation's own onSuccess already wipes the sessionStorage draft.
		await skipMutation.mutateAsync();
		navigate("/dashboard", { replace: true });
	}, [skipMutation, navigate]);

	const jumpTo = useCallback(
		(step: OnboardingStep) => {
			if (step > reachedStep) return;
			navigate(STEP_PATHS[step], { replace: true });
		},
		[reachedStep, navigate],
	);

	return {
		currentStep,
		reachedStep,
		canGoNext,
		prev,
		next,
		skip,
		jumpTo,
		registerCanGoNext,
	};
}
