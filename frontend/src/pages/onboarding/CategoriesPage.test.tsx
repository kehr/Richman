import { renderWithProviders } from "@/test/utils";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import CategoriesPage from "./CategoriesPage";
import { OnboardingStateProvider } from "./state";

const navNext = vi.fn(async () => undefined);

// This test harness mimics the real hook by tracking registered predicates and
// forcing re-evaluations when they change.
const predicateFunctions: (() => boolean)[] = [];
let forceRender: (() => void) | null = null;

vi.mock("./use-onboarding-nav", async () => {
	// biome-ignore lint/suspicious/noExplicitAny: vitest importActual returns any
	const actual: any = await vi.importActual("./use-onboarding-nav");
	const { useMemo, useState } = await vi.importActual<typeof import("react")>("react");

	return {
		...actual,
		useOnboardingNav: () => {
			const [predicateVersion, setPredicateVersion] = useState(0);

			// Store a global trigger so registerCanGoNext can force re-renders
			if (!forceRender) {
				forceRender = () => {
					setPredicateVersion((v) => v + 1);
				};
			}

			const canGoNext = useMemo(() => {
				void predicateVersion;
				if (predicateFunctions.length === 0) return true;
				return predicateFunctions.every((p) => {
					try {
						return p();
					} catch {
						return false;
					}
				});
			}, [predicateVersion]);

			return {
				currentStep: 2,
				reachedStep: 2,
				canGoNext,
				prev: vi.fn(),
				next: navNext,
				skip: vi.fn(),
				jumpTo: vi.fn(),
				registerCanGoNext: (predicate: () => boolean) => {
					predicateFunctions.push(predicate);
					forceRender?.();
					return () => {
						const index = predicateFunctions.indexOf(predicate);
						if (index !== -1) {
							predicateFunctions.splice(index, 1);
						}
						forceRender?.();
					};
				},
			};
		},
	};
});

// Mock the user-settings barrel so the PATCH mutation resolves synchronously
// and the page does not hit the real API in the test environment. The mock
// exposes every hook touched along the OnboardingStateProvider +
// useOnboardingNav path so tests do not need a live API surface.
const mutateAsync = vi.fn(async () => undefined);
vi.mock("@/features/user-settings", () => ({
	usePatchUserSettings: () => ({
		mutateAsync,
		isPending: false,
	}),
	useSkipOnboarding: () => ({
		mutateAsync: vi.fn(async () => undefined),
		isPending: false,
	}),
	useOnboardingStatus: () => ({
		data: { completed: false, skipped: false },
	}),
	useUserSettings: () => ({
		data: { categories: [] },
	}),
}));

describe("CategoriesPage", () => {
	beforeEach(() => {
		navNext.mockClear();
		mutateAsync.mockClear();
		predicateFunctions.length = 0;
	});

	function renderPage() {
		return renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/categories"]}>
				<OnboardingStateProvider>
					<CategoriesPage />
				</OnboardingStateProvider>
			</MemoryRouter>,
		);
	}

	it("disables the next button until at least one category is selected", async () => {
		const user = userEvent.setup();
		renderPage();

		const nextBtn = screen.getByTestId("onboarding-categories-next");
		expect(nextBtn).toBeDisabled();

		await user.click(screen.getByTestId("category-card-gold_etf"));
		expect(screen.getByTestId("category-card-gold_etf")).toHaveAttribute("data-selected", "true");
		expect(nextBtn).not.toBeDisabled();
	});

	it("supports toggling multiple categories and saves them on next", async () => {
		const user = userEvent.setup();
		renderPage();

		await user.click(screen.getByTestId("category-card-gold_etf"));
		await user.click(screen.getByTestId("category-card-us_stock"));
		// Toggle off again to confirm multi-select behaviour.
		await user.click(screen.getByTestId("category-card-gold_etf"));
		await user.click(screen.getByTestId("category-card-a_share_broad"));

		await user.click(screen.getByTestId("onboarding-categories-next"));

		await waitFor(() => {
			expect(mutateAsync).toHaveBeenCalledTimes(1);
		});
		expect(mutateAsync).toHaveBeenCalledWith({
			categories: ["us_stock", "a_share_broad"],
		});
		await waitFor(() => {
			expect(navNext).toHaveBeenCalled();
		});
	});
});
