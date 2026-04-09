import type { DecisionCardDTO } from "@/features/decision-card";
import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import DecisionCardDetailPage from "./DecisionCardDetailPage";

const mockNavigate = vi.fn();
vi.mock("react-router", async () => {
	const actual = await vi.importActual<typeof import("react-router")>("react-router");
	return {
		...actual,
		useNavigate: () => mockNavigate,
	};
});

let detailState: { data: DecisionCardDTO | undefined; isLoading: boolean; error: unknown };
let prevState: { data: DecisionCardDTO | undefined; isLoading: boolean; error: unknown };
let cardsState: { data: DecisionCardDTO[]; isLoading: boolean; error: unknown };

vi.mock("@/features/decision-card", async () => {
	const actual = await vi.importActual<typeof import("@/features/decision-card")>(
		"@/features/decision-card",
	);
	return {
		...actual,
		// useDecisionCardDetail is called twice (current then prev). The second
		// call is identified by checking the requested cardId so the mock can
		// return the prev card instead of duplicating the current one.
		useDecisionCardDetail: (cardId: number) => {
			if (cardId === 41) return prevState;
			return detailState;
		},
		useDecisionCards: () => cardsState,
	};
});

vi.mock("@/domain/money/useMoney", () => ({
	useMoney: () => ({
		hasCapital: true,
		format: (pct: number, amount?: number | null) =>
			amount != null ? `${pct}% · ¥${amount}` : `${pct}%`,
		formatAmountOnly: (amount?: number | null) => (amount != null ? `¥${amount}` : null),
	}),
}));

function makeCard(overrides: Partial<DecisionCardDTO> = {}): DecisionCardDTO {
	return {
		cardId: 42,
		userId: 1,
		holdingId: 10,
		assetCode: "600519",
		assetName: "贵州茅台",
		assetType: "a-share",
		costPrice: 1800,
		positionRatio: 20,
		positionAmount: 20000,
		trendDirection: "bullish",
		trendSummary: "趋势向上",
		positionDirection: "neutral",
		positionSummary: "仓位中性",
		catalystDirection: "bullish",
		catalystSummary: "有正面催化",
		confidence: 0.82,
		actionAdvice: "",
		detailedAdvice: "",
		riskWarnings: ["高位回调风险", "宏观流动性收紧"],
		todayHighlights: "",
		weightTrend: 40,
		weightPosition: 30,
		weightCatalyst: 30,
		analyzedAt: "2026-04-07T00:30:00Z",
		createdAt: "2026-04-07T00:30:00Z",
		recommendation: {
			action: "small_add",
			actionLevel: 1,
			label: "小幅加仓",
			currentPositionPct: 20,
			targetPositionPct: 25,
			execution: {
				type: "staged",
				validDays: 5,
				steps: [
					{
						order: 1,
						triggerType: "price",
						triggerValue: "回调到 1750",
						deltaPct: 5,
						rationale: "首段加仓承接技术回调",
					},
				],
			},
		},
		actionLevel: 1,
		targetPositionRatio: 25,
		targetPositionAmount: 25000,
		badgeState: "action_upgrade",
		confidenceDelta: 5,
		prevCardId: 41,
		executionFingerprint: "fp",
		synthesisSource: "llm",
		providerUsed: "user",
		...overrides,
	};
}

function renderPage(cardId = "42") {
	return renderWithProviders(
		<MemoryRouter initialEntries={[`/decision-cards/${cardId}`]}>
			<Routes>
				<Route path="/decision-cards/:id" element={<DecisionCardDetailPage />} />
			</Routes>
		</MemoryRouter>,
	);
}

describe("DecisionCardDetailPage", () => {
	beforeEach(() => {
		mockNavigate.mockReset();
		detailState = { data: makeCard(), isLoading: false, error: null };
		prevState = { data: undefined, isLoading: false, error: null };
		cardsState = { data: [], isLoading: false, error: null };
	});

	it("renders all five blocks and the meta sidebar when data is loaded", () => {
		renderPage();
		expect(screen.getByTestId("card-hero")).toBeInTheDocument();
		expect(screen.getByTestId("conclusion-banner")).toBeInTheDocument();
		expect(screen.getByTestId("plan-full")).toBeInTheDocument();
		expect(screen.getByTestId("dimension-reasoning")).toBeInTheDocument();
		expect(screen.getByTestId("main-risks")).toBeInTheDocument();
		expect(screen.getByTestId("meta-sidebar")).toBeInTheDocument();
		expect(screen.getByText("贵州茅台")).toBeInTheDocument();
		expect(screen.getByText("小幅加仓")).toBeInTheDocument();
		expect(screen.getByText("高位回调风险")).toBeInTheDocument();
	});

	it("renders not-found state when query returns no card", () => {
		detailState = { data: undefined, isLoading: false, error: null };
		renderPage();
		expect(screen.getByTestId("detail-not-found")).toBeInTheDocument();
	});

	it("renders error state when query fails", () => {
		detailState = { data: undefined, isLoading: false, error: new Error("boom") };
		renderPage();
		expect(screen.getByTestId("detail-error")).toBeInTheDocument();
	});
});
