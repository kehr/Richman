import { useMoney } from "@/domain/money/useMoney";
import { Space, Typography } from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import { useFormatTriggerValue } from "../format-trigger";
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

// deltaPctColor follows A-share convention: red = increase (涨/加仓),
// green = decrease (跌/减仓), gray = neutral (0%).
function deltaPctColor(delta: number): string {
	if (delta > 0) return "#f5222d";
	if (delta < 0) return "#52c41a";
	return "#8c8c8c";
}

function StepRow({
	step,
	index,
	amountStr,
}: { step: Step; index: number; amountStr?: string | null }) {
	const formatTrigger = useFormatTriggerValue();

	return (
		<div style={{ display: "flex", alignItems: "center" }} data-testid={`plan-step-${step.order}`}>
			<span style={stepCircleStyle(index)}>{step.order}</span>
			<Text style={{ flex: 1 }}>{formatTrigger(step)}</Text>
			<div
				style={{
					display: "flex",
					flexDirection: "column",
					alignItems: "flex-end",
					marginLeft: 8,
				}}
			>
				<Text strong style={{ color: deltaPctColor(step.deltaPct) }}>
					{formatDeltaPct(step.deltaPct)}
				</Text>
				{amountStr != null && (
					<Text type="secondary" style={{ fontSize: 11, lineHeight: 1.3 }}>
						{amountStr}
					</Text>
				)}
			</div>
		</div>
	);
}

interface ExecutionPlanStripProps {
	execution: Execution;
	maxSteps?: number;
	onShowAll?: () => void;
	// positionAmountCny and positionRatioPct are used together to derive total
	// capital so each step can show the change amount alongside the percent.
	// Both must be non-null and positionRatioPct must be > 0 to enable display.
	positionAmountCny?: number | null;
	positionRatioPct?: number;
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
	positionAmountCny,
	positionRatioPct,
}: ExecutionPlanStripProps) {
	const { t } = useTranslation("app");
	const money = useMoney();

	const steps = execution.steps ?? [];

	// Derive total capital from the holding's current position so each step can
	// display the absolute amount change alongside the percentage delta. Only
	// computed when the user has configured capital (hasCapital) so the display
	// stays consistent with every other money amount in the app.
	const totalCapitalCny =
		money.hasCapital &&
		positionAmountCny != null &&
		positionRatioPct != null &&
		positionRatioPct > 0
			? positionAmountCny / (positionRatioPct / 100)
			: null;

	// Format a step's absolute change amount in the user's display currency.
	// Delegates to money.formatAmountOnly so conversion and degraded-mode
	// currency fallback are handled consistently with the rest of the app.
	function stepAmountStr(step: Step): string | null {
		if (totalCapitalCny == null || totalCapitalCny <= 0 || step.deltaPct === 0) return null;
		const amountCny = Math.round(Math.abs((totalCapitalCny * step.deltaPct) / 100));
		return money.formatAmountOnly(amountCny);
	}

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
				<StepRow key={step.order} step={step} index={idx} amountStr={stepAmountStr(step)} />
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
