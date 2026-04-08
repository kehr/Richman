import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AccountTab } from "./AccountTab";

// Mocks for the feature hooks. patchMutate / resetMutate are spies the
// individual tests assert against; settingsState is mutated per test.
const patchMutate = vi.fn(async () => undefined);
const resetMutate = vi.fn(async () => undefined);
const logoutSpy = vi.fn();
const navigateSpy = vi.fn();

vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => navigateSpy,
	};
});
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

// Mock useCurrentUser explicitly so the test does not depend on the real
// `/auth/me` query's `enabled: !!getToken()` short-circuit. The mock must
// mirror the hook's `select` unwrap so consumers get a flat User object
// instead of the ApiResponse envelope.
vi.mock("@/domain/auth/use-current-user", () => ({
	useCurrentUser: () => ({
		data: { email: "tester@example.com" },
		isLoading: false,
	}),
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
		navigateSpy.mockClear();
		settingsState = {
			data: { totalCapitalCny: 100_000, riskPreference: "neutral" },
			isLoading: false,
		};
	});

	it("renders email, total capital input, risk preference select, and logout button", () => {
		renderTab();
		expect(screen.getByTestId("account-email")).toHaveTextContent("tester@example.com");
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

	it("always renders the re-entry onboarding button regardless of environment", () => {
		renderTab();
		expect(screen.getByTestId("account-reset-onboarding")).toBeInTheDocument();
	});

	it("triggers reset and navigates to /onboarding/welcome after the Popconfirm is confirmed", async () => {
		renderTab();
		// Open the Popconfirm by clicking the trigger button.
		fireEvent.click(screen.getByTestId("account-reset-onboarding"));
		// antd Popconfirm renders its ok button inside a portal; find it by
		// the Chinese label we set in AccountTab.
		const okButton = await screen.findByRole("button", { name: "开始引导" });
		fireEvent.click(okButton);
		await waitFor(() => {
			expect(resetMutate).toHaveBeenCalledTimes(1);
		});
		await waitFor(() => {
			expect(navigateSpy).toHaveBeenCalledWith("/onboarding/welcome");
		});
	});
});
