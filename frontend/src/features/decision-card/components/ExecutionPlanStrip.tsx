import { Space, Typography } from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import type { Execution, Step } from "../types";

const { Text } = Typography;

// STEP_COLORS is the gradient used for the numbered step circles. The
// index is clamped to the last color for any step beyond position 3 so
// very long staged plans still produce legible circles; in practice the
// visible steps are capped at `maxSteps` (default 3).
const STEP_COLORS = ["#1677ff", "#4096ff", "#69b1ff", "#91caff"];

function stepCircleStyle(index: number): CSSProperties {
	return {
		display: "inline-flex",
		alignItems: "center",
		justifyContent: "center",
		width: 20,
		height: 20,
		borderRadius: "50%",
		background: STEP_COLORS[Math.min(index, STEP_COLORS.length - 1)],
		color: "#fff",
		fontSize: 12,
		marginRight: 8,
		flex: "0 0 auto",
	};
}

// formatDeltaPct renders a signed percentage suffix ("+20%" / "-10%"). The
// sign is derived from the raw delta so the caller does not need to
// pre-format it.
function formatDeltaPct(delta: number): string {
	const sign = delta > 0 ? "+" : "";
	return `${sign}${delta.toFixed(0)}%`;
}

function StepRow({ step, index }: { step: Step; index: number }) {
	return (
		<div style={{ display: "flex", alignItems: "center" }} data-testid={`plan-step-${step.order}`}>
			<span style={stepCircleStyle(index)}>{step.order}</span>
			<Text style={{ flex: 1 }}>{step.triggerValue}</Text>
			<Text strong style={{ marginLeft: 8 }}>
				{formatDeltaPct(step.deltaPct)}
			</Text>
		</div>
	);
}

interface ExecutionPlanStripProps {
	execution: Execution;
	maxSteps?: number;
	onShowAll?: () => void;
}

// ExecutionPlanStrip renders a compact summary of the structured execution
// plan attached to a recommendation. Three shapes are supported:
//
//   * one-shot : a single step row
//   * staged   : up to `maxSteps` steps (default 3) with a "+N more" link
//                when the plan has more steps than the cap
//   * monitor  : two monitoring lines for stop-loss and take-profit; no
//                numbered circles because monitor plans have no steps
//
// When `onShowAll` is omitted the "+N more" element is rendered as plain
// text so the strip is still readable on pages that do not have a detail
// navigation target (e.g. screenshot previews in tests).
export function ExecutionPlanStrip({
	execution,
	maxSteps = 3,
	onShowAll,
}: ExecutionPlanStripProps) {
	const { t } = useTranslation("app");

	const steps = execution.steps ?? [];

	// Legacy monitor cards without steps: render stop-loss / take-profit only.
	if (execution.type === "monitor" && steps.length === 0) {
		return (
			<Space direction="vertical" size={2} style={{ width: "100%" }}>
				<Text type="secondary" data-testid="plan-monitor-stop-loss">
					{t("decisionCard.executionPlan.stopLoss")}:{" "}
					{execution.stopLoss != null
						? execution.stopLoss.toFixed(2)
						: t("decisionCard.executionPlan.notSet")}
				</Text>
				<Text type="secondary" data-testid="plan-monitor-take-profit">
					{t("decisionCard.executionPlan.takeProfit")}:{" "}
					{execution.takeProfit != null
						? execution.takeProfit.toFixed(2)
						: t("decisionCard.executionPlan.notSet")}
				</Text>
			</Space>
		);
	}

	const visible = execution.type === "one-shot" ? steps.slice(0, 1) : steps.slice(0, maxSteps);
	const hidden = Math.max(0, steps.length - visible.length);

	return (
		<Space direction="vertical" size={4} style={{ width: "100%" }}>
			{visible.map((step, idx) => (
				<StepRow key={step.order} step={step} index={idx} />
			))}
			{hidden > 0 && (
				<Text
					type="secondary"
					style={{ cursor: onShowAll ? "pointer" : "default" }}
					onClick={
						onShowAll
							? (event) => {
									event.stopPropagation();
									onShowAll();
								}
							: undefined
					}
					onKeyDown={
						onShowAll
							? (event) => {
									if (event.key === "Enter" || event.key === " ") {
										event.preventDefault();
										event.stopPropagation();
										onShowAll();
									}
								}
							: undefined
					}
					role={onShowAll ? "button" : undefined}
					tabIndex={onShowAll ? 0 : undefined}
					data-testid="plan-more-link"
				>
					{t("decisionCard.executionPlan.moreSteps", { count: hidden })}
				</Text>
			)}
			{execution.type === "monitor" && (
				<Text type="secondary" style={{ fontSize: 12 }}>
					{t("decisionCard.executionPlan.stopLoss")}:{" "}
					{execution.stopLoss?.toFixed(2) ?? t("decisionCard.executionPlan.notSet")}
					{" / "}
					{t("decisionCard.executionPlan.takeProfit")}:{" "}
					{execution.takeProfit?.toFixed(2) ?? t("decisionCard.executionPlan.notSet")}
				</Text>
			)}
		</Space>
	);
}
