import { renderWithProviders } from "@/test/utils";
import { waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import FirstAnalysisPage from "./FirstAnalysisPage";
import {
	DEFAULT_ONBOARDING_STATE,
	ONBOARDING_DRAFT_STORAGE_KEY,
	OnboardingStateProvider,
} from "./state";

// Mock the feature barrels used by the page so the tests do not hit a real
// API surface. Every hook touched along the provider + page path must be
// stubbed because OnboardingStateProvider consumes useOnboardingStatus and
// useUserSettings on mount.
const rerunMutateAsync = vi.fn(async () => undefined);
const markCompletedMutateAsync = vi.fn(async () => undefined);

vi.mock("@/features/decision-card", () => ({
	useRerunAnalysis: () => ({
		mutateAsync: rerunMutateAsync,
		isPending: false,
	}),
}));

vi.mock("@/features/user-settings", () => ({
	useMarkOnboardingCompleted: () => ({
		mutateAsync: markCompletedMutateAsync,
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

describe("FirstAnalysisPage", () => {
	beforeEach(() => {
		rerunMutateAsync.mockClear();
		markCompletedMutateAsync.mockClear();
		sessionStorage.clear();
	});

	afterEach(() => {
		sessionStorage.clear();
	});

	function renderPage() {
		return renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/first-analysis"]}>
				<OnboardingStateProvider>
					<FirstAnalysisPage />
				</OnboardingStateProvider>
			</MemoryRouter>,
		);
	}

	it("fires the analysis trigger exactly once when analysisFired is false", async () => {
		renderPage();
		await waitFor(() => {
			expect(rerunMutateAsync).toHaveBeenCalledTimes(1);
		});
	});

	it("skips the analysis trigger when state.analysisFired is already true", async () => {
		// Seed sessionStorage with a draft that has analysisFired=true so the
		// provider's lazy init reads it on first render. This simulates the
		// user arriving back at step 4 after navigating to step 3 and forward
		// again within the same session.
		sessionStorage.setItem(
			ONBOARDING_DRAFT_STORAGE_KEY,
			JSON.stringify({
				...DEFAULT_ONBOARDING_STATE,
				analysisFired: true,
			}),
		);

		renderPage();

		// Give React a frame to run any effects. We assert the negative: even
		// after the paint, the mutation must not have been dispatched.
		await new Promise((resolve) => setTimeout(resolve, 0));
		expect(rerunMutateAsync).not.toHaveBeenCalled();
	});
});
