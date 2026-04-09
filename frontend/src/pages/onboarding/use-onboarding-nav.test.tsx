import type { OnboardingStatus, UserSettings } from "@/features/user-settings";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type ReactNode, useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OnboardingStateProvider, useOnboardingState } from "./state";
import { SHAKE_EVENT_NAME, useOnboardingNav } from "./use-onboarding-nav";

// react-router mocks: we stub useNavigate + useLocation per-test so we can
// inspect navigation calls and control the perceived current step without
// wrapping every test in a real <MemoryRouter>. The real module is still
// imported actual to keep any type-only exports we may touch indirectly.
const mockNavigate = vi.fn();
const mockLocation = { pathname: "/onboarding/welcome" } as { pathname: string };

vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
		useLocation: () => mockLocation,
	};
});

// user-settings mocks: status and settings are fixed at "new user" so the
// provider does not wipe sessionStorage. The skip mutation is a stub whose
// mutateAsync can be inspected by the skip() test.
const mockUseOnboardingStatus = vi.fn<() => { data: OnboardingStatus | undefined }>();
const mockUseUserSettings = vi.fn<() => { data: UserSettings | undefined }>();
const mockSkipMutateAsync = vi.fn();

vi.mock("@/features/user-settings", async () => {
	const actual = await vi.importActual<typeof import("@/features/user-settings")>(
		"@/features/user-settings",
	);
	return {
		...actual,
		useOnboardingStatus: () => mockUseOnboardingStatus(),
		useUserSettings: () => mockUseUserSettings(),
		useSkipOnboarding: () => ({ mutateAsync: mockSkipMutateAsync }),
	};
});

function Wrapper({ children }: { children: ReactNode }) {
	return <OnboardingStateProvider>{children}</OnboardingStateProvider>;
}

function setLocation(pathname: string) {
	mockLocation.pathname = pathname;
}

describe("useOnboardingNav", () => {
	beforeEach(() => {
		sessionStorage.clear();
		localStorage.clear();
		mockNavigate.mockReset();
		mockSkipMutateAsync.mockReset();
		mockUseOnboardingStatus.mockReset();
		mockUseUserSettings.mockReset();
		mockUseOnboardingStatus.mockReturnValue({
			data: { completed: false, skipped: false },
		});
		mockUseUserSettings.mockReturnValue({
			data: {
				userId: 1,
				riskPreference: "neutral",
				categories: [],
				onboardingCompleted: false,
			},
		});
		setLocation("/onboarding/welcome");
	});

	afterEach(() => {
		sessionStorage.clear();
		localStorage.clear();
	});

	it("prev() at step 1 is a no-op", () => {
		setLocation("/onboarding/welcome");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });
		act(() => result.current.prev());
		expect(mockNavigate).not.toHaveBeenCalled();
	});

	it("prev() at step 3 navigates back to step 2", () => {
		setLocation("/onboarding/first-holding");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });
		act(() => result.current.prev());
		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/categories", { replace: true });
	});

	it("next() with canGoNext=false dispatches the shake event instead of navigating", async () => {
		setLocation("/onboarding/categories");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });

		// Register a predicate that always fails so canGoNext aggregates to false.
		act(() => {
			result.current.registerCanGoNext(() => false);
		});
		await waitFor(() => expect(result.current.canGoNext).toBe(false));

		const listener = vi.fn();
		window.addEventListener(SHAKE_EVENT_NAME, listener);
		try {
			await act(async () => {
				await result.current.next();
			});
		} finally {
			window.removeEventListener(SHAKE_EVENT_NAME, listener);
		}

		expect(listener).toHaveBeenCalledTimes(1);
		expect(mockNavigate).not.toHaveBeenCalled();
	});

	it("next() with canGoNext=true advances to the next step", async () => {
		setLocation("/onboarding/categories");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });

		act(() => {
			result.current.registerCanGoNext(() => true);
		});
		await waitFor(() => expect(result.current.canGoNext).toBe(true));

		await act(async () => {
			await result.current.next();
		});

		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/first-holding", { replace: true });
	});

	it("jumpTo(step > reachedStep) is a no-op", () => {
		// Seed sessionStorage so reachedStep is 2, not 1.
		sessionStorage.setItem(
			"richman_onboarding_draft",
			JSON.stringify({
				categories: [],
				holdingDraft: { mode: "quick" },
				reachedStep: 2,
				analysisFired: false,
			}),
		);
		setLocation("/onboarding/categories");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });
		act(() => result.current.jumpTo(4));
		expect(mockNavigate).not.toHaveBeenCalled();
	});

	it("jumpTo(step <= reachedStep) navigates", () => {
		sessionStorage.setItem(
			"richman_onboarding_draft",
			JSON.stringify({
				categories: [],
				holdingDraft: { mode: "quick" },
				reachedStep: 3,
				analysisFired: false,
			}),
		);
		setLocation("/onboarding/first-holding");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });
		act(() => result.current.jumpTo(2));
		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/categories", { replace: true });
	});

	it("skip() awaits the mutation and then navigates to /dashboard", async () => {
		mockSkipMutateAsync.mockResolvedValue(undefined);
		setLocation("/onboarding/welcome");
		const { result } = renderHook(() => useOnboardingNav(), { wrapper: Wrapper });

		await act(async () => {
			await result.current.skip();
		});

		expect(mockSkipMutateAsync).toHaveBeenCalledTimes(1);
		expect(mockNavigate).toHaveBeenCalledWith("/dashboard", { replace: true });
	});

	// Regression: a consumer that registers a canGoNext predicate inside a
	// useEffect keyed on `nav` used to infinite-loop because the hook returned
	// a fresh object literal every render. The fix is to wrap the return in
	// useMemo so the nav object identity is stable when nothing real changed.
	// This test mounts a component that mirrors the real CategoriesPage wiring
	// (state.categories + nav.registerCanGoNext + nav in deps) and asserts
	// the render settles without exceeding React's "maximum update depth"
	// safeguard.
	it("does not infinite-loop when a consumer registers a predicate in a nav-keyed effect", () => {
		setLocation("/onboarding/categories");

		function ConsumerProbe() {
			const nav = useOnboardingNav();
			const { state } = useOnboardingState();
			useEffect(() => {
				return nav.registerCanGoNext(() => state.categories.length >= 1);
			}, [nav, state.categories]);
			return <div data-testid="probe">canGoNext={String(nav.canGoNext)}</div>;
		}

		// React logs "Maximum update depth exceeded" via console.error before
		// throwing. If the bug regresses, this render call will either throw
		// directly or stall; a try/catch plus a spy lets us fail loudly with a
		// specific message instead of a test timeout.
		const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
		try {
			const { getByTestId } = render(
				<OnboardingStateProvider>
					<ConsumerProbe />
				</OnboardingStateProvider>,
			);
			expect(getByTestId("probe").textContent).toBe("canGoNext=false");

			const depthErrors = errorSpy.mock.calls.filter((call) =>
				String(call[0]).includes("Maximum update depth exceeded"),
			);
			expect(depthErrors).toHaveLength(0);
		} finally {
			errorSpy.mockRestore();
		}
	});
});
