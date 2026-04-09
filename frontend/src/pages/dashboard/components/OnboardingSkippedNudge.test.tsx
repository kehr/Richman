import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ONBOARDING_NUDGE_DISMISS_KEY, OnboardingSkippedNudge } from "./OnboardingSkippedNudge";

// Mock the status hook so each test can inject its own state. The nudge
// imports it from @/features/user-settings (barrel), so the mock must target
// that same module specifier — matches the pattern used in
// onboarding-guard.test.tsx.
const mockStatus = vi.fn();
vi.mock("@/features/user-settings", () => ({
	useOnboardingStatus: () => mockStatus(),
}));

// Mock useNavigate so we can assert the "开始引导" target without a real
// router side effect.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

function renderNudge() {
	return renderWithProviders(
		<MemoryRouter initialEntries={["/dashboard"]}>
			<OnboardingSkippedNudge />
		</MemoryRouter>,
	);
}

describe("OnboardingSkippedNudge", () => {
	beforeEach(() => {
		mockStatus.mockReset();
		mockNavigate.mockReset();
		// Every test starts from a clean slate so cross-test localStorage
		// state (dismissed flag) never leaks.
		window.localStorage.clear();
	});

	it("renders when skipped=true and not dismissed", () => {
		mockStatus.mockReturnValue({
			data: { completed: false, skipped: true },
			isLoading: false,
		});
		renderNudge();
		expect(screen.getByTestId("onboarding-skipped-nudge")).toBeInTheDocument();
	});

	it("renders nothing when skipped is false", () => {
		mockStatus.mockReturnValue({
			data: { completed: false, skipped: false },
			isLoading: false,
		});
		renderNudge();
		expect(screen.queryByTestId("onboarding-skipped-nudge")).toBeNull();
	});

	it("renders nothing while status is loading", () => {
		mockStatus.mockReturnValue({ data: undefined, isLoading: true });
		renderNudge();
		expect(screen.queryByTestId("onboarding-skipped-nudge")).toBeNull();
	});

	it("renders nothing when the dismissed flag is set in localStorage", () => {
		window.localStorage.setItem(ONBOARDING_NUDGE_DISMISS_KEY, "1");
		mockStatus.mockReturnValue({
			data: { completed: false, skipped: true },
			isLoading: false,
		});
		renderNudge();
		expect(screen.queryByTestId("onboarding-skipped-nudge")).toBeNull();
	});

	it("dismisses the nudge and persists the flag when 不再提示 is clicked", () => {
		mockStatus.mockReturnValue({
			data: { completed: false, skipped: true },
			isLoading: false,
		});
		renderNudge();
		expect(screen.getByTestId("onboarding-skipped-nudge")).toBeInTheDocument();

		fireEvent.click(screen.getByTestId("onboarding-skipped-nudge-dismiss"));

		expect(screen.queryByTestId("onboarding-skipped-nudge")).toBeNull();
		expect(window.localStorage.getItem(ONBOARDING_NUDGE_DISMISS_KEY)).toBe("1");
	});

	it("navigates to /onboarding/welcome when 开始引导 is clicked", () => {
		mockStatus.mockReturnValue({
			data: { completed: false, skipped: true },
			isLoading: false,
		});
		renderNudge();

		fireEvent.click(screen.getByTestId("onboarding-skipped-nudge-restart"));

		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/welcome");
	});
});
