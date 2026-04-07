import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import PortfolioListPage from "./PortfolioListPage";

// Mock react-router useNavigate so we can assert navigation targets without
// pulling in a full router tree.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

// Feature hooks are mocked at the barrel layer so the page tests stay free of
// network mocks. holdingsState lets each test toggle the empty / populated /
// at-limit conditions in isolation.
let holdingsState: { data: unknown[]; isLoading: boolean };
let cardsState: { data: unknown[] };
let settingsState: { data: unknown; isLoading: boolean };
const deleteMutate = vi.fn(async () => undefined);

vi.mock("@/features/portfolio", async () => {
	const actual =
		await vi.importActual<typeof import("@/features/portfolio")>("@/features/portfolio");
	return {
		...actual,
		useHoldings: () => holdingsState,
		useDeleteHolding: () => ({ mutateAsync: deleteMutate, isPending: false }),
		useCreateHolding: () => ({ mutateAsync: vi.fn(), isPending: false }),
	};
});

vi.mock("@/features/decision-card", () => ({
	useDecisionCards: () => cardsState,
}));

vi.mock("@/features/user-settings", () => ({
	useUserSettings: () => settingsState,
}));

vi.mock("@/domain/money/useMoney", () => ({
	useMoney: () => ({
		hasCapital: true,
		format: (pct: number, amount?: number | null) =>
			amount != null ? `${pct}% · ¥${amount}` : `${pct}%`,
		formatAmountOnly: (amount?: number | null) => (amount != null ? `¥${amount}` : null),
	}),
}));

function makeHolding(overrides: Partial<Record<string, unknown>> = {}) {
	return {
		holdingId: 1,
		assetCode: "510300",
		assetName: "沪深 300",
		assetType: "a_share_broad",
		costPrice: 4.12,
		positionRatio: 20,
		quantity: 0,
		...overrides,
	};
}

function renderPage() {
	return renderWithProviders(
		<MemoryRouter initialEntries={["/portfolio"]}>
			<PortfolioListPage />
		</MemoryRouter>,
	);
}

describe("PortfolioListPage", () => {
	beforeEach(() => {
		mockNavigate.mockReset();
		deleteMutate.mockClear();
		holdingsState = { data: [], isLoading: false };
		cardsState = { data: [] };
		settingsState = { data: { totalCapitalCny: 100_000 }, isLoading: false };
	});

	it("renders the header counter, total capital row and add buttons", () => {
		renderPage();
		expect(screen.getByTestId("holding-counter")).toHaveTextContent("0/5 个持仓");
		expect(screen.getByTestId("total-capital-row")).toBeInTheDocument();
		expect(screen.getByTestId("add-holding-button")).toBeEnabled();
		expect(screen.getByTestId("screenshot-import-button")).toBeDisabled();
	});

	it("disables the add button when at the 5-holding limit", () => {
		holdingsState = {
			data: Array.from({ length: 5 }, (_, i) =>
				makeHolding({ holdingId: i + 1, assetCode: `${i}` }),
			),
			isLoading: false,
		};
		renderPage();
		expect(screen.getByTestId("holding-counter")).toHaveTextContent("5/5 个持仓");
		expect(screen.getByTestId("add-holding-button")).toBeDisabled();
	});

	it("navigates to the latest decision card when a row is clicked", () => {
		holdingsState = { data: [makeHolding({ holdingId: 7 })], isLoading: false };
		cardsState = { data: [{ cardId: 99, holdingId: 7 }] };
		renderPage();
		fireEvent.click(screen.getByTestId("holding-row-7"));
		expect(mockNavigate).toHaveBeenCalledWith("/decision-cards/99");
	});

	it("falls back to the edit page when no decision card exists", () => {
		holdingsState = { data: [makeHolding({ holdingId: 7 })], isLoading: false };
		cardsState = { data: [] };
		renderPage();
		fireEvent.click(screen.getByTestId("holding-row-7"));
		expect(mockNavigate).toHaveBeenCalledWith("/portfolio/7");
	});
});
