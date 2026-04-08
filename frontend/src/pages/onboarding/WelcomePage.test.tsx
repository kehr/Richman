import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import WelcomePage from "./WelcomePage";
import { OnboardingStateProvider } from "./state";

// Mock useOnboardingNav so we can assert the next() call without wiring the
// full router. All other hooks come from the real module via importActual.
const mockNext = vi.fn();
vi.mock("./use-onboarding-nav", async () => {
	// biome-ignore lint/suspicious/noExplicitAny: vitest importActual returns any
	const actual: any = await vi.importActual("./use-onboarding-nav");
	return {
		...actual,
		useOnboardingNav: () => ({
			currentStep: 1,
			reachedStep: 1,
			canGoNext: true,
			prev: vi.fn(),
			next: mockNext,
			skip: vi.fn(),
			jumpTo: vi.fn(),
			registerCanGoNext: () => () => {},
		}),
	};
});

// OnboardingLayout (rendered inside WelcomePage) calls useOnboardingNav which
// in turn calls useSkipOnboarding / useOnboardingStatus / useUserSettings via
// the OnboardingStateProvider. Stub the user-settings barrel to keep the test
// offline and the provider happy.
vi.mock("@/features/user-settings", () => ({
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

describe("WelcomePage", () => {
	beforeEach(() => {
		mockNext.mockReset();
	});

	it("renders title, three dimension cards, and CTA button", () => {
		renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/welcome"]}>
				<OnboardingStateProvider>
					<WelcomePage />
				</OnboardingStateProvider>
			</MemoryRouter>,
		);
		expect(screen.getByText("欢迎，让我们开始")).toBeInTheDocument();
		expect(screen.getByTestId("dimension-card-trend")).toBeInTheDocument();
		expect(screen.getByTestId("dimension-card-position")).toBeInTheDocument();
		expect(screen.getByTestId("dimension-card-catalyst")).toBeInTheDocument();
		expect(screen.getByTestId("onboarding-welcome-next")).toBeInTheDocument();
	});

	it("calls nav.next() when the CTA button is clicked", async () => {
		const user = userEvent.setup();
		renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/welcome"]}>
				<OnboardingStateProvider>
					<WelcomePage />
				</OnboardingStateProvider>
			</MemoryRouter>,
		);
		await user.click(screen.getByTestId("onboarding-welcome-next"));
		expect(mockNext).toHaveBeenCalled();
	});
});
