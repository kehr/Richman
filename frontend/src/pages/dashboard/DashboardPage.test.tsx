import { renderWithProviders } from "@/test/utils";
import { screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import DashboardPage from "./DashboardPage";

// Mock react-router useNavigate so we can assert navigation targets without
// spinning up a full router tree.
const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

// Mutable state for the feature-hook mocks. Each test tweaks these before
// rendering so the composition can be exercised across empty / populated
// states without re-mocking on every call.
let holdingsState: { data: unknown[]; isLoading: boolean };
let cardsState: { data: unknown[]; isLoading: boolean; error: unknown };
let settingsState: { data: { totalCapitalCny?: number } | undefined };

vi.mock("@/features/portfolio", () => ({
	useHoldings: () => holdingsState,
}));

vi.mock("@/features/decision-card", async () => {
	const actual = await vi.importActual<typeof import("@/features/decision-card")>(
		"@/features/decision-card",
	);
	return {
		...actual,
		useDecisionCards: () => ({ ...cardsState, refetch: vi.fn() }),
		useRerunAnalysis: () => ({ mutateAsync: vi.fn(async () => undefined), isPending: false }),
		useReanalyzeAll: () => ({ mutateAsync: vi.fn(async () => undefined), isPending: false }),
	};
});

vi.mock("@/features/dashboard-summary", () => ({
	useDashboardSummary: () => ({
		data: {
			llmStatus: {
				configured: false,
				userProviderHealth: "not_configured",
				systemDefaultAvailable: false,
				needsReanalysis: false,
			},
		},
		isLoading: false,
	}),
}));

vi.mock("@/features/user-settings", () => ({
	useUserSettings: () => settingsState,
}));

vi.mock("@/domain/money/useMoney", () => ({
	useMoney: () => {
		// Drive hasCapital from the shared settingsState so each test case can
		// exercise both the "with capital" and "no capital" money-format paths.
		const cny = settingsState.data?.totalCapitalCny;
		const hasCapital = cny != null;
		return {
			hasCapital,
			format: (pct: number, amount?: number | null) =>
				hasCapital && amount != null ? `${pct}% · ¥${amount}` : `${pct}%`,
			formatAmountOnly: (amount?: number | null) =>
				hasCapital && amount != null ? `¥${amount}` : null,
		};
	},
}));

function renderPage() {
	return renderWithProviders(
		<MemoryRouter initialEntries={["/dashboard"]}>
			<DashboardPage />
		</MemoryRouter>,
	);
}

describe("DashboardPage", () => {
	beforeEach(() => {
		mockNavigate.mockReset();
		holdingsState = { data: [], isLoading: false };
		cardsState = { data: [], isLoading: false, error: null };
		settingsState = { data: { totalCapitalCny: 100_000 } };
	});

	it("renders EmptyHoldingsHero when there are no holdings", () => {
		renderPage();
		expect(screen.getByTestId("empty-holdings-hero")).toBeInTheDocument();
		expect(screen.queryByTestId("dashboard-top-strip")).toBeNull();
	});

	it("renders the three-region layout when holdings are populated", () => {
		holdingsState = {
			data: [
				{
					holdingId: 1,
					assetCode: "600519",
					assetName: "贵州茅台",
					assetType: "a-share",
					costPrice: 1800,
					positionRatio: 20,
					quantity: 10,
				},
			],
			isLoading: false,
		};
		cardsState = {
			data: [
				{
					cardId: 42,
					userId: 1,
					holdingId: 1,
					assetCode: "600519",
					assetName: "贵州茅台",
					assetType: "a-share",
					costPrice: 1800,
					positionRatio: 20,
					positionAmount: 20000,
					trendDirection: "bullish",
					trendSummary: "",
					positionDirection: "neutral",
					positionSummary: "",
					catalystDirection: "bullish",
					catalystSummary: "",
					confidence: 0.8,
					actionAdvice: "",
					detailedAdvice: "",
					riskWarnings: [],
					todayHighlights: "",
					weightTrend: 40,
					weightPosition: 30,
					weightCatalyst: 30,
					analyzedAt: "2026-04-07T08:30:00Z",
					createdAt: "2026-04-07T08:30:00Z",
					recommendation: {
						action: "small_add",
						actionLevel: 1,
						label: "小幅加仓",
						currentPositionPct: 20,
						targetPositionPct: 25,
						execution: { type: "monitor", validDays: 7 },
					},
					actionLevel: 1,
					targetPositionRatio: 25,
					targetPositionAmount: 25000,
					badgeState: "action_upgrade",
					confidenceDelta: 3,
					prevCardId: 41,
					executionFingerprint: "fp",
				},
			],
			isLoading: false,
			error: null,
		};

		renderPage();
		expect(screen.getByTestId("dashboard-top-strip")).toBeInTheDocument();
		expect(screen.getByTestId("decision-card-wall")).toBeInTheDocument();
		expect(screen.getByTestId("change-anchor-list")).toBeInTheDocument();
		expect(screen.getByTestId("stat-holding-count")).toHaveTextContent("1");
		expect(screen.getByTestId("decision-card-42")).toBeInTheDocument();
	});

	it("renders percent-only aggregates when the user has no total capital configured", () => {
		holdingsState = {
			data: [
				{
					holdingId: 1,
					assetCode: "600519",
					assetName: "贵州茅台",
					assetType: "a-share",
					costPrice: 1800,
					positionRatio: 20,
					quantity: 10,
				},
			],
			isLoading: false,
		};
		cardsState = {
			data: [
				{
					cardId: 42,
					userId: 1,
					holdingId: 1,
					assetCode: "600519",
					assetName: "贵州茅台",
					assetType: "a-share",
					costPrice: 1800,
					positionRatio: 20,
					// No positionAmount when capital is not set; aggregatePnlAmount
					// should fall through to null and the top strip should render
					// percent only.
					positionAmount: null,
					trendDirection: "bullish",
					trendSummary: "",
					positionDirection: "neutral",
					positionSummary: "",
					catalystDirection: "bullish",
					catalystSummary: "",
					confidence: 0.8,
					actionAdvice: "",
					detailedAdvice: "",
					riskWarnings: [],
					todayHighlights: "",
					weightTrend: 40,
					weightPosition: 30,
					weightCatalyst: 30,
					analyzedAt: "2026-04-07T08:30:00Z",
					createdAt: "2026-04-07T08:30:00Z",
					recommendation: {
						action: "hold",
						actionLevel: 0,
						label: "持有等待",
						currentPositionPct: 20,
						targetPositionPct: 20,
						execution: { type: "monitor", validDays: 7 },
					},
					actionLevel: 0,
					targetPositionRatio: 20,
					targetPositionAmount: null,
					badgeState: "none",
					confidenceDelta: 0,
					prevCardId: null,
					executionFingerprint: "fp",
				},
			],
			isLoading: false,
			error: null,
		};
		settingsState = { data: {} };

		renderPage();
		// DashboardTopStrip still renders; the aggregate-P&L block should not
		// expose a ¥ amount because the user has not set total capital.
		expect(screen.getByTestId("dashboard-top-strip")).toBeInTheDocument();
		expect(screen.getByTestId("dashboard-top-strip").textContent).not.toContain("¥");
	});

	it("shows the card wall loading skeleton while cards are loading", async () => {
		holdingsState = {
			data: [
				{
					holdingId: 1,
					assetCode: "600519",
					assetName: "贵州茅台",
					assetType: "a-share",
					costPrice: 1800,
					positionRatio: 20,
					quantity: 10,
				},
			],
			isLoading: false,
		};
		cardsState = { data: [], isLoading: true, error: null };
		renderPage();
		await waitFor(() => {
			expect(screen.getByTestId("decision-card-wall-loading")).toBeInTheDocument();
		});
	});
});
