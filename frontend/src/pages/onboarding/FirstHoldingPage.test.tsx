import { renderWithProviders } from "@/test/utils";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import FirstHoldingPage from "./FirstHoldingPage";
import { OnboardingStateProvider } from "./state";

const navNext = vi.fn(async () => undefined);

// This test harness mirrors CategoriesPage.test.tsx: a shared mutable list of
// predicates plus a forceRender ref lets us emulate the real hook's
// "registerCanGoNext triggers a re-aggregate" behavior without depending on
// the router or the ExperienceStateProvider.
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
				currentStep: 3,
				reachedStep: 3,
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

// Mock the user-settings barrel so the OnboardingStateProvider resolves its
// dependent queries without hitting the network. The onboarding-status / user
// settings shape matches the one used by the page at runtime.
const patchMutate = vi.fn(async () => undefined);
vi.mock("@/features/user-settings", () => ({
	usePatchUserSettings: () => ({
		mutateAsync: patchMutate,
		isPending: false,
	}),
	useUserSettings: () => ({
		data: { categories: [], totalCapitalCny: null },
	}),
	useSkipOnboarding: () => ({
		mutateAsync: vi.fn(async () => undefined),
		isPending: false,
	}),
	useOnboardingStatus: () => ({
		data: { completed: false, skipped: false },
	}),
}));

// Control the holdings list so the Alert branch (existing holdings) can be
// toggled per test case. The createHolding mutation just records the input.
const createHoldingMutate = vi.fn(async () => undefined);
let holdingsData: Array<{ id: number }> = [];
vi.mock("@/features/portfolio", () => ({
	useHoldings: () => ({ data: holdingsData }),
	useCreateHolding: () => ({
		mutateAsync: createHoldingMutate,
		isPending: false,
	}),
}));

// Stub the asset catalog so the Select shows deterministic options without
// hitting the network. One asset per test is enough to cover the flow.
vi.mock("@/features/asset-catalog", async () => {
	// biome-ignore lint/suspicious/noExplicitAny: vitest importActual returns any
	const actual: any = await vi.importActual("@/features/asset-catalog");
	return {
		...actual,
		useAssets: () => ({
			data: [
				{
					code: "518880",
					name: "黄金ETF",
					assetType: "gold_etf",
				},
			],
			isLoading: false,
		}),
	};
});

describe("FirstHoldingPage", () => {
	beforeEach(() => {
		navNext.mockClear();
		patchMutate.mockClear();
		createHoldingMutate.mockClear();
		predicateFunctions.length = 0;
		forceRender = null;
		holdingsData = [];
	});

	function renderPage() {
		return renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/first-holding"]}>
				<OnboardingStateProvider>
					<FirstHoldingPage />
				</OnboardingStateProvider>
			</MemoryRouter>,
		);
	}

	it("disables the submit button until required fields are filled", async () => {
		renderPage();
		const submit = screen.getByTestId("onboarding-holding-submit");
		expect(submit).toBeDisabled();
	});

	it("enables the submit button and calls createHolding + nav.next on submit", async () => {
		const user = userEvent.setup();
		renderPage();

		// Pick the asset via the Select dropdown.
		const selects = screen.getAllByRole("combobox");
		await user.click(selects[0]);
		await user.click(await screen.findByText(/518880\s+黄金ETF/));

		// Fill the cost price input (first spinbutton: 成本价).
		const numericInputs = screen.getAllByRole("spinbutton");
		await user.clear(numericInputs[0]);
		await user.type(numericInputs[0], "3.25");
		// The positionRatio input is seeded with 10 on mount so we do not need
		// to touch it; the predicate should already be satisfied.

		const submit = screen.getByTestId("onboarding-holding-submit");
		await waitFor(() => {
			expect(submit).not.toBeDisabled();
		});

		await user.click(submit);

		await waitFor(() => {
			expect(createHoldingMutate).toHaveBeenCalledTimes(1);
		});
		expect(createHoldingMutate).toHaveBeenCalledWith(
			expect.objectContaining({
				assetCode: "518880",
				assetName: "黄金ETF",
				assetType: "gold_etf",
				costPrice: 3.25,
				positionRatio: 10,
				quantity: 0,
			}),
		);
		await waitFor(() => {
			expect(navNext).toHaveBeenCalled();
		});
	});

	it("renders the fast-forward alert when holdings already exist and the button calls nav.next", async () => {
		holdingsData = [{ id: 1 }, { id: 2 }];
		const user = userEvent.setup();
		renderPage();

		const fastForward = screen.getByTestId("onboarding-skip-to-analysis");
		expect(fastForward).toHaveTextContent("用已有持仓直接分析");

		await user.click(fastForward);
		await waitFor(() => {
			expect(navNext).toHaveBeenCalled();
		});
	});
});
