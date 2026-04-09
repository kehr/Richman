import { type DecisionCardDTO, computeNextAnalysisTime } from "@/features/decision-card";
import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { MetaSidebar } from "./MetaSidebar";

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
		trendSummary: "",
		positionDirection: "neutral",
		positionSummary: "",
		catalystDirection: "bearish",
		catalystSummary: "",
		confidence: 0.8,
		actionAdvice: "",
		detailedAdvice: "",
		riskWarnings: [],
		todayHighlights: "",
		weightTrend: 40,
		weightPosition: 30,
		weightCatalyst: 30,
		analyzedAt: "2026-04-07T00:30:00Z",
		createdAt: "2026-04-07T00:30:00Z",
		recommendation: {
			action: "hold",
			actionLevel: 0,
			label: "保持观察",
			currentPositionPct: 15,
			targetPositionPct: 15,
			execution: { type: "monitor", validDays: 7 },
		},
		actionLevel: 0,
		targetPositionRatio: 15,
		targetPositionAmount: 15000,
		badgeState: "none",
		confidenceDelta: 0,
		prevCardId: null,
		executionFingerprint: "fp",
		synthesisSource: "llm",
		providerUsed: "user",
		...overrides,
	};
}

describe("MetaSidebar", () => {
	it("renders analysis time, next analysis, data source placeholder and disclaimer", () => {
		renderWithProviders(<MetaSidebar card={makeCard()} />);
		// Analyzed at 2026-04-07T00:30:00Z = 08:30 Asia/Shanghai
		expect(screen.getByTestId("meta-analyzed-at")).toHaveTextContent("2026-04-07 08:30");
		expect(screen.getByTestId("meta-next-analysis")).toBeInTheDocument();
		// Data source health is intentionally a placeholder until the backend
		// exposes per-source freshness; the block must not claim "正常".
		expect(screen.getByTestId("meta-data-source")).toHaveTextContent(
			"数据源健康检查将在后续版本开放",
		);
		expect(screen.getByTestId("meta-disclaimer")).toBeInTheDocument();
	});

	it("filters out the current card from history and invokes onSelectHistory", async () => {
		const user = userEvent.setup();
		const onSelect = vi.fn();
		const current = makeCard({ cardId: 5 });
		const history = [
			makeCard({ cardId: 5 }), // self, must be filtered
			makeCard({ cardId: 4, analyzedAt: "2026-04-06T00:30:00Z" }),
			makeCard({ cardId: 3, analyzedAt: "2026-04-05T00:30:00Z" }),
		];
		renderWithProviders(
			<MetaSidebar card={current} historicalCards={history} onSelectHistory={onSelect} />,
		);
		expect(screen.queryByTestId("meta-history-5")).toBeNull();
		await user.click(screen.getByTestId("meta-history-4"));
		expect(onSelect).toHaveBeenCalledWith(4);
	});
});

describe("computeNextAnalysisTime", () => {
	it("returns a Date strictly after the input", () => {
		const now = new Date("2026-04-07T00:00:00Z"); // Tue 08:00 Shanghai
		const next = computeNextAnalysisTime(now);
		expect(next).not.toBeNull();
		expect((next as Date).getTime()).toBeGreaterThan(now.getTime());
	});
});
