import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { DashboardTopStrip, computeNextAnalysisTime, formatHm } from "./DashboardTopStrip";

// Stub useMoney so the strip renders without a surrounding settings query.
vi.mock("@/domain/money/useMoney", () => ({
	useMoney: () => ({
		hasCapital: true,
		format: (pct: number, amount?: number | null) =>
			amount != null ? `${pct}% · ¥${amount}` : `${pct}%`,
		formatAmountOnly: (amount?: number | null) => (amount != null ? `¥${amount}` : null),
	}),
}));

describe("DashboardTopStrip", () => {
	const baseProps = {
		holdingCount: 5,
		totalCapitalCny: 100_000,
		totalPositionRatio: 72,
		aggregatePnlAmount: 1500,
		aggregatePnlPct: 1.5,
		lastAnalyzedAt: new Date("2026-04-07T08:30:00"),
		nextAnalysisAt: new Date("2026-04-07T15:30:00"),
		onRerun: vi.fn(),
		rerunLoading: false,
		onConfigureCapital: vi.fn(),
	};

	it("renders title, time labels and four stats", () => {
		renderWithProviders(<DashboardTopStrip {...baseProps} />);
		expect(screen.getByText("今日决策")).toBeInTheDocument();
		expect(screen.getByTestId("dashboard-top-strip-times")).toHaveTextContent(
			"最后分析 08:30 · 下次自动 15:30",
		);
		expect(screen.getByTestId("stat-holding-count")).toHaveTextContent("5");
		expect(screen.getByTestId("stat-total-capital")).toHaveTextContent("¥100,000");
		expect(screen.getByTestId("stat-aggregate-pnl")).toHaveTextContent("1.5%");
		expect(screen.getByTestId("stat-allocated-position")).toHaveTextContent("72%");
	});

	it("shows the capital CTA when totalCapitalCny is null", async () => {
		const user = userEvent.setup();
		const onConfigureCapital = vi.fn();
		renderWithProviders(
			<DashboardTopStrip
				{...baseProps}
				totalCapitalCny={null}
				onConfigureCapital={onConfigureCapital}
			/>,
		);
		const cta = screen.getByTestId("stat-total-capital-cta");
		expect(cta).toBeInTheDocument();
		await user.click(cta);
		expect(onConfigureCapital).toHaveBeenCalledTimes(1);
	});

	it("calls onRerun when the button is clicked", async () => {
		const user = userEvent.setup();
		const onRerun = vi.fn();
		renderWithProviders(<DashboardTopStrip {...baseProps} onRerun={onRerun} />);
		await user.click(screen.getByTestId("dashboard-rerun-button"));
		expect(onRerun).toHaveBeenCalledTimes(1);
	});
});

describe("formatHm", () => {
	it("pads hours and minutes to two digits", () => {
		expect(formatHm(new Date("2026-04-07T06:05:00"))).toBe("06:05");
	});

	it("returns a placeholder for null", () => {
		expect(formatHm(null)).toBe("--:--");
	});
});

describe("computeNextAnalysisTime", () => {
	it("returns the next slot after now within the schedule", () => {
		// Tuesday 2026-04-07 07:00 local: next slot should be the same day's
		// 08:30 A-share AM brief.
		const now = new Date("2026-04-07T07:00:00");
		const next = computeNextAnalysisTime(now);
		expect(next).not.toBeNull();
		expect(next?.getHours()).toBe(8);
		expect(next?.getMinutes()).toBe(30);
	});

	it("skips to the next weekday when run after the last slot", () => {
		// Friday 2026-04-10 16:00 local: last A-share slot (15:30) already
		// passed, so the next slot is Saturday 06:00 (US digest runs Tue-Sat).
		const now = new Date("2026-04-10T16:00:00");
		const next = computeNextAnalysisTime(now);
		expect(next).not.toBeNull();
		expect(next?.getDay()).toBe(6);
		expect(next?.getHours()).toBe(6);
	});
});
