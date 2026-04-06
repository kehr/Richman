import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { OnboardingGuard } from "./onboarding-guard";

// Mock the status hook so each test can inject its own state.
const mockStatus = vi.fn();
vi.mock("./use-onboarding-status", () => ({
	useOnboardingStatus: () => mockStatus(),
}));

// Mock useNavigate so we can assert the redirect target without actually
// navigating the test DOM.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

function renderAt(path: string) {
	return render(
		<MemoryRouter initialEntries={[path]}>
			<OnboardingGuard>
				<div data-testid="child">child content</div>
			</OnboardingGuard>
		</MemoryRouter>,
	);
}

describe("OnboardingGuard", () => {
	beforeEach(() => {
		mockStatus.mockReset();
		mockNavigate.mockReset();
	});

	it("renders a spinner while loading", () => {
		mockStatus.mockReturnValue({ data: undefined, isLoading: true, error: null });
		const { container } = renderAt("/dashboard");
		expect(container.querySelector(".ant-spin")).not.toBeNull();
		expect(screen.queryByTestId("child")).toBeNull();
		expect(mockNavigate).not.toHaveBeenCalled();
	});

	it("redirects to /onboarding/welcome when user has not completed and is outside onboarding", () => {
		mockStatus.mockReturnValue({
			data: { completed: false },
			isLoading: false,
			error: null,
		});
		renderAt("/dashboard");
		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/welcome", { replace: true });
		// No child content during redirect.
		expect(screen.queryByTestId("child")).toBeNull();
	});

	it("renders onboarding routes when user has not completed", () => {
		mockStatus.mockReturnValue({
			data: { completed: false },
			isLoading: false,
			error: null,
		});
		renderAt("/onboarding/welcome");
		expect(mockNavigate).not.toHaveBeenCalled();
		expect(screen.getByTestId("child")).toBeInTheDocument();
	});

	it("redirects to /dashboard when a completed user hits an onboarding route", () => {
		mockStatus.mockReturnValue({
			data: { completed: true },
			isLoading: false,
			error: null,
		});
		renderAt("/onboarding/welcome");
		expect(mockNavigate).toHaveBeenCalledWith("/dashboard", { replace: true });
		expect(screen.queryByTestId("child")).toBeNull();
	});

	it("passes through for completed users on main app routes", () => {
		mockStatus.mockReturnValue({
			data: { completed: true },
			isLoading: false,
			error: null,
		});
		renderAt("/dashboard");
		expect(mockNavigate).not.toHaveBeenCalled();
		expect(screen.getByTestId("child")).toBeInTheDocument();
	});
});
