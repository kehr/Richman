import type { DecisionCardDTO } from "@/features/decision-card";
import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ChangeAnchorList } from "./ChangeAnchorList";

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
		confidence: 0.8,
		actionAdvice: "",
		detailedAdvice: "",
		riskWarnings: [],
		todayHighlights: "",
		weightTrend: 40,
		weightPosition: 30,
		weightCatalyst: 30,
		analyzedAt: "2026-04-07T08:00:00Z",
		createdAt: "2026-04-07T08:00:00Z",
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
		...overrides,
	};
}

describe("ChangeAnchorList", () => {
	it("renders nothing when every card has badgeState 'none'", () => {
		renderWithProviders(
			<ChangeAnchorList
				cards={[makeCard({ cardId: 1 }), makeCard({ cardId: 2 })]}
				cardRefs={new Map()}
			/>,
		);
		expect(screen.queryByTestId("change-anchor-list")).toBeNull();
	});

	it("filters and lists only cards whose badgeState is not 'none'", () => {
		const cards = [
			makeCard({ cardId: 1, badgeState: "none", assetName: "茅台" }),
			makeCard({ cardId: 2, badgeState: "action_upgrade", assetName: "宁德时代" }),
			makeCard({ cardId: 3, badgeState: "signal_flip", assetName: "英伟达" }),
		];
		renderWithProviders(<ChangeAnchorList cards={cards} cardRefs={new Map()} />);
		expect(screen.queryByTestId("change-anchor-row-1")).toBeNull();
		expect(screen.getByTestId("change-anchor-row-2")).toHaveTextContent("宁德时代");
		expect(screen.getByTestId("change-anchor-row-3")).toHaveTextContent("英伟达");
	});

	it("scrolls and highlights the matching card when a row is clicked", async () => {
		const user = userEvent.setup();
		const node = document.createElement("div");
		const scrollIntoView = vi.fn();
		node.scrollIntoView = scrollIntoView as unknown as typeof node.scrollIntoView;
		const cardRefs = new Map<number, HTMLDivElement>([[2, node as HTMLDivElement]]);

		renderWithProviders(
			<ChangeAnchorList
				cards={[makeCard({ cardId: 2, badgeState: "action_upgrade" })]}
				cardRefs={cardRefs}
			/>,
		);

		await user.click(screen.getByTestId("change-anchor-row-2"));
		expect(scrollIntoView).toHaveBeenCalledTimes(1);
		expect(node.classList.contains("dashboard-change-anchor-highlight")).toBe(true);
	});
});
