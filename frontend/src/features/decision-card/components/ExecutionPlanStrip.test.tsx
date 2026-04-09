import { renderWithProviders } from "@/test/utils";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Execution, Step } from "../types";
import { ExecutionPlanStrip } from "./ExecutionPlanStrip";

function makeStep(order: number, delta: number, trigger = `触发 ${order}`): Step {
	return {
		order,
		triggerType: "price",
		triggerValue: trigger,
		deltaPct: delta,
		rationale: `rationale ${order}`,
	};
}

describe("ExecutionPlanStrip", () => {
	it("renders a single step for one-shot plans", () => {
		const execution: Execution = {
			type: "one-shot",
			steps: [makeStep(1, 20)],
			validDays: 7,
		};
		renderWithProviders(<ExecutionPlanStrip execution={execution} />);
		expect(screen.getByTestId("plan-step-1")).toHaveTextContent("触发 1");
		expect(screen.getByTestId("plan-step-1")).toHaveTextContent("+20%");
		expect(screen.queryByTestId("plan-more-link")).toBeNull();
	});

	it("caps staged plans at 3 steps and shows a '+N more' link", () => {
		const execution: Execution = {
			type: "staged",
			steps: [makeStep(1, 10), makeStep(2, 10), makeStep(3, 10), makeStep(4, 10), makeStep(5, 10)],
			validDays: 7,
		};
		const onShowAll = vi.fn();
		renderWithProviders(<ExecutionPlanStrip execution={execution} onShowAll={onShowAll} />);
		expect(screen.getByTestId("plan-step-1")).toBeInTheDocument();
		expect(screen.getByTestId("plan-step-2")).toBeInTheDocument();
		expect(screen.getByTestId("plan-step-3")).toBeInTheDocument();
		expect(screen.queryByTestId("plan-step-4")).toBeNull();
		expect(screen.getByTestId("plan-more-link")).toHaveTextContent("+ 2 more");
	});

	it("renders stop-loss and take-profit rows for monitor plans", () => {
		const execution: Execution = {
			type: "monitor",
			stopLoss: 95.5,
			takeProfit: 120,
			validDays: 30,
		};
		renderWithProviders(<ExecutionPlanStrip execution={execution} />);
		expect(screen.getByTestId("plan-monitor-stop-loss")).toHaveTextContent("95.5");
		expect(screen.getByTestId("plan-monitor-take-profit")).toHaveTextContent("120");
	});

	it("shows 'Not set' when monitor guardrails are missing", () => {
		const execution: Execution = {
			type: "monitor",
			stopLoss: null,
			takeProfit: null,
			validDays: 30,
		};
		renderWithProviders(<ExecutionPlanStrip execution={execution} />);
		expect(screen.getByTestId("plan-monitor-stop-loss")).toHaveTextContent("Not set");
		expect(screen.getByTestId("plan-monitor-take-profit")).toHaveTextContent("Not set");
	});
});
