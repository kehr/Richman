import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AccountTab } from "./AccountTab";

// Mocks for the feature hooks. patchMutate / resetMutate are spies the
// individual tests assert against; settingsState is mutated per test.
const patchMutate = vi.fn(async () => undefined);
const resetMutate = vi.fn(async () => undefined);
const logoutSpy = vi.fn();
let settingsState: {
	data: { totalCapitalCny: number | null; riskPreference: string } | undefined;
	isLoading: boolean;
};

vi.mock("@/features/user-settings", () => ({
	useUserSettings: () => settingsState,
	usePatchUserSettings: () => ({ mutateAsync: patchMutate, isPending: false }),
	useResetOnboarding: () => ({ mutateAsync: resetMutate, isPending: false }),
}));

vi.mock("@/features/auth", () => ({
	useLogout: () => logoutSpy,
}));

function renderTab() {
	return renderWithProviders(
		<MemoryRouter>
			<AccountTab />
		</MemoryRouter>,
	);
}

describe("AccountTab", () => {
	beforeEach(() => {
		patchMutate.mockClear();
		resetMutate.mockClear();
		logoutSpy.mockClear();
		settingsState = {
			data: { totalCapitalCny: 100_000, riskPreference: "neutral" },
			isLoading: false,
		};
	});

	afterEach(() => {
		vi.unstubAllEnvs();
	});

	it("renders email, total capital input, risk preference select, and logout button", () => {
		renderTab();
		expect(screen.getByTestId("account-email")).toBeInTheDocument();
		expect(screen.getByTestId("account-total-capital-input")).toBeInTheDocument();
		expect(screen.getByTestId("account-risk-preference")).toBeInTheDocument();
		expect(screen.getByTestId("account-logout")).toBeInTheDocument();
	});

	it("saves the parsed total capital number via usePatchUserSettings", async () => {
		renderTab();
		// Wait for the form's useEffect to seed the input from settings.
		await waitFor(() => {
			const input = screen.getByTestId("account-total-capital-input") as HTMLInputElement;
			expect(input.value).toContain("100");
		});
		const input = screen.getByTestId("account-total-capital-input") as HTMLInputElement;
		fireEvent.change(input, { target: { value: "250000" } });
		fireEvent.click(screen.getByTestId("account-total-capital-save"));
		await waitFor(() => {
			expect(patchMutate).toHaveBeenCalledWith({ totalCapitalCny: 250_000 });
		});
	});

	it("calls usePatchUserSettings with the chosen risk preference", async () => {
		renderTab();
		// Open the antd Select dropdown and click the "激进" (aggressive) option.
		const select = screen
			.getByTestId("account-risk-preference")
			.querySelector(".ant-select-selector") as HTMLElement;
		fireEvent.mouseDown(select);
		const option = await screen.findByText("激进");
		fireEvent.click(option);
		await waitFor(() => {
			expect(patchMutate).toHaveBeenCalledWith({ riskPreference: "aggressive" });
		});
	});

	it("renders the dev-only reset onboarding button when import.meta.env.DEV is true", async () => {
		vi.stubEnv("DEV", true);
		renderTab();
		const resetButton = screen.getByTestId("account-reset-onboarding");
		expect(resetButton).toBeInTheDocument();
		fireEvent.click(resetButton);
		await waitFor(() => {
			expect(resetMutate).toHaveBeenCalledTimes(1);
		});
	});
});
