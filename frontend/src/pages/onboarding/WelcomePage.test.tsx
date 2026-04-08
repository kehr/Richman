import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import WelcomePage from "./WelcomePage";
import { OnboardingStateProvider } from "./state";

// Mock react-router useNavigate so we can assert the target without actually
// navigating the test DOM. All other router primitives come from the real
// module via importActual.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
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
		mockNavigate.mockReset();
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

	it("navigates to the categories step when the CTA is clicked", async () => {
		const user = userEvent.setup();
		renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/welcome"]}>
				<OnboardingStateProvider>
					<WelcomePage />
				</OnboardingStateProvider>
			</MemoryRouter>,
		);
		await user.click(screen.getByTestId("onboarding-welcome-next"));
		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/categories");
	});
});
