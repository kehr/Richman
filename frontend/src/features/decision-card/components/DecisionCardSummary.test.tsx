import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { DecisionCardDTO } from "../types";
import { DecisionCardSummary } from "./DecisionCardSummary";

// Stub useMoney so the summary can render without a surrounding QueryClient.
// The real hook is covered by domain/money tests; here we only need a
// deterministic percent+amount formatter.
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
		cardId: 1,
		userId: 1,
		holdingId: 10,
		assetCode: "600519",
		assetName: "贵州茅台",
		assetType: "a-share",
		costPrice: 1800,
		positionRatio: 15,
		positionAmount: 15000,
		trendDirection: "bullish",
		trendSummary: "t",
		positionDirection: "neutral",
		positionSummary: "p",
		catalystDirection: "bearish",
		catalystSummary: "c",
		confidence: 0.82,
		actionAdvice: "加仓 5%",
		detailedAdvice: "detailed",
		riskWarnings: [],
		todayHighlights: "盘中拉升 2%",
		weightTrend: 40,
		weightPosition: 30,
		weightCatalyst: 30,
		analyzedAt: "2026-04-07T08:00:00Z",
		createdAt: "2026-04-07T08:00:00Z",
		recommendation: {
			action: "small_add",
			actionLevel: 1,
			label: "小幅加仓",
			currentPositionPct: 15,
			targetPositionPct: 20,
			execution: {
				type: "staged",
				steps: [
					{ order: 1, triggerType: "price", triggerValue: "跌至 1750", deltaPct: 2, rationale: "" },
					{ order: 2, triggerType: "price", triggerValue: "跌至 1700", deltaPct: 3, rationale: "" },
				],
				validDays: 7,
			},
		},
		actionLevel: 1,
		targetPositionRatio: 20,
		targetPositionAmount: 20000,
		badgeState: "action_upgrade",
		confidenceDelta: 5,
		prevCardId: null,
		executionFingerprint: "fp",
		synthesisSource: "llm",
		providerUsed: "user",
		...overrides,
	};
}

describe("DecisionCardSummary", () => {
	it("renders header, dimensions, plan and confidence", () => {
		render(<DecisionCardSummary card={makeCard()} />);
		expect(screen.getByText("贵州茅台")).toBeInTheDocument();
		expect(screen.getByText("600519")).toBeInTheDocument();
		expect(screen.getByTestId("change-badge-action_upgrade")).toBeInTheDocument();
		expect(screen.getByTestId("dim-trend-current")).toHaveTextContent("bullish");
		expect(screen.getByTestId("plan-step-1")).toHaveTextContent("跌至 1750");
		expect(screen.getByTestId("card-confidence")).toHaveTextContent("82%");
	});

	it("calls onClick with the card payload when clicked", async () => {
		const user = userEvent.setup();
		const onClick = vi.fn();
		render(<DecisionCardSummary card={makeCard()} onClick={onClick} />);
		await user.click(screen.getByTestId("decision-card-1"));
		expect(onClick).toHaveBeenCalledTimes(1);
		expect(onClick.mock.calls[0][0].cardId).toBe(1);
	});

	it("renders pnl row only when useMoney returns an amount", () => {
		render(<DecisionCardSummary card={makeCard({ positionAmount: null })} />);
		expect(screen.queryByTestId("card-pnl")).toBeNull();
	});

	it("enables dimension flip when previousCard differs", () => {
		const card = makeCard();
		const previous = makeCard({ trendDirection: "bearish", cardId: 0 });
		render(<DecisionCardSummary card={card} previousCard={previous} />);
		expect(screen.getByTestId("dim-trend-prev")).toHaveTextContent("bearish");
		expect(screen.getByTestId("dim-trend-current")).toHaveTextContent("bullish");
	});
});
