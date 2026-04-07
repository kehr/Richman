import { renderWithProviders } from "@/test/utils";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import CategoriesPage from "./CategoriesPage";

const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

// Mock the user-settings barrel so the PATCH mutation resolves synchronously
// and the page does not hit the real API in the test environment. The mock
// exposes a single usePatchUserSettings hook; the test only needs the
// mutateAsync + isPending surface.
const mutateAsync = vi.fn(async () => undefined);
vi.mock("@/features/user-settings", () => ({
	usePatchUserSettings: () => ({
		mutateAsync,
		isPending: false,
	}),
}));

describe("CategoriesPage", () => {
	beforeEach(() => {
		mockNavigate.mockReset();
		mutateAsync.mockClear();
	});

	function renderPage() {
		return renderWithProviders(
			<MemoryRouter initialEntries={["/onboarding/categories"]}>
				<CategoriesPage />
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
		expect(mockNavigate).toHaveBeenCalledWith("/onboarding/first-holding");
	});
});
