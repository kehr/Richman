import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import PortfolioTransactionsPage from "./PortfolioTransactionsPage";

let holdingsState: { data: unknown[]; isLoading: boolean };
let tradesState: { data: unknown[]; isLoading: boolean };

vi.mock("@/features/portfolio", async () => {
	const actual =
		await vi.importActual<typeof import("@/features/portfolio")>("@/features/portfolio");
	return {
		...actual,
		useHoldings: () => holdingsState,
		useTrades: () => tradesState,
		useCreateTrade: () => ({ mutateAsync: vi.fn(), isPending: false }),
	};
});

vi.mock("@/features/user-settings", () => ({
	useUserSettings: () => ({ data: { totalCapitalCny: 100_000 }, isLoading: false }),
}));

vi.mock("@/domain/money/useMoney", () => ({
	useMoney: () => ({
		hasCapital: true,
		format: (pct: number, amount?: number | null) =>
			amount != null ? `${pct}% · ¥${amount.toFixed(2)}` : `${pct}%`,
		formatAmountOnly: (amount?: number | null) => (amount != null ? `¥${amount.toFixed(2)}` : null),
	}),
}));

function renderPage(holdingId: number) {
	return renderWithProviders(
		<MemoryRouter initialEntries={[`/portfolio/${holdingId}/transactions`]}>
			<Routes>
				<Route path="/portfolio/:id/transactions" element={<PortfolioTransactionsPage />} />
			</Routes>
		</MemoryRouter>,
	);
}

describe("PortfolioTransactionsPage", () => {
	beforeEach(() => {
		holdingsState = {
			data: [
				{
					holdingId: 7,
					assetCode: "510300",
					assetName: "沪深 300",
					assetType: "a_share_broad",
					costPrice: 4.12,
					positionRatio: 20,
					quantity: 0,
				},
			],
			isLoading: false,
		};
		tradesState = {
			data: [
				{
					tradeId: 1,
					holdingId: 7,
					direction: "buy",
					price: 4.0,
					quantity: 1000,
					tradedAt: "2026-04-01T03:00:00.000Z",
				},
				{
					tradeId: 2,
					holdingId: 7,
					direction: "sell",
					price: 4.5,
					quantity: 200,
					tradedAt: "2026-04-05T05:00:00.000Z",
				},
			],
			isLoading: false,
		};
	});

	it("renders the transaction table with both rows", () => {
		renderPage(7);
		expect(screen.getByTestId("portfolio-transactions-page")).toBeInTheDocument();
		expect(screen.getByTestId("transaction-table")).toBeInTheDocument();
		expect(screen.getByText("买入")).toBeInTheDocument();
		expect(screen.getByText("卖出")).toBeInTheDocument();
	});

	it("disables the trade delete buttons with the placeholder tooltip", () => {
		renderPage(7);
		const del1 = screen.getByTestId("trade-delete-1");
		expect(del1).toBeDisabled();
	});

	it("renders the summary card with computed totals", () => {
		renderPage(7);
		const summary = screen.getByTestId("transactions-summary");
		// Total buy = 4.0 * 1000 = 4000.00; total sell = 4.5 * 200 = 900.00.
		expect(summary).toHaveTextContent("¥4000.00");
		expect(summary).toHaveTextContent("¥900.00");
		// Weighted cost is 4000 / 1000 = 4.00.
		expect(summary).toHaveTextContent("¥4.00");
	});

	it("shows a not-found card when the holding does not exist", () => {
		holdingsState = { data: [], isLoading: false };
		renderPage(999);
		expect(screen.getByText("未找到对应持仓")).toBeInTheDocument();
	});
});
