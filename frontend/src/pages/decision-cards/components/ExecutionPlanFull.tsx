import type { Execution, Step } from "@/features/decision-card";
import { Alert, Card, Space, Typography } from "@/ui-kit/eat";
import type { CSSProperties } from "react";

const { Text, Paragraph, Title } = Typography;

interface ExecutionPlanFullProps {
	execution: Execution;
}

// stepCircleStyle matches the ExecutionPlanStrip visual but uses a slightly
// larger circle (28px) because the detail page has more room. Index is
// clamped to the last gradient stop when plans exceed 4 steps.
const STEP_COLORS = ["#1677ff", "#4096ff", "#69b1ff", "#91caff"];

function stepCircleStyle(index: number): CSSProperties {
	return {
		display: "inline-flex",
		alignItems: "center",
		justifyContent: "center",
		width: 28,
		height: 28,
		borderRadius: "50%",
		background: STEP_COLORS[Math.min(index, STEP_COLORS.length - 1)],
		color: "#fff",
		fontSize: 14,
		fontWeight: 600,
		flex: "0 0 auto",
	};
}

function formatDeltaPct(delta: number): string {
	const sign = delta > 0 ? "+" : "";
	return `${sign}${delta.toFixed(0)}%`;
}

// StepRow renders one full execution step with its trigger condition, delta
// and rationale paragraph. The rationale is rendered as plaintext because
// the project does not pull in a markdown renderer yet; markdown symbols
// from the backend show through unmodified rather than crashing the view.
function StepRow({ step, index }: { step: Step; index: number }) {
	return (
		<div
			data-testid={`plan-full-step-${step.order}`}
			style={{ display: "flex", alignItems: "flex-start", gap: 12 }}
		>
			<span style={stepCircleStyle(index)}>{step.order}</span>
			<Space direction="vertical" size={4} style={{ flex: 1 }}>
				<div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
					<Text strong>{step.triggerValue}</Text>
					<Text strong style={{ color: "#1677ff" }}>
						{formatDeltaPct(step.deltaPct)}
					</Text>
				</div>
				{step.rationale && (
					<Paragraph
						type="secondary"
						style={{ margin: 0, whiteSpace: "pre-wrap" }}
						data-testid={`plan-full-rationale-${step.order}`}
					>
						{step.rationale}
					</Paragraph>
				)}
			</Space>
		</div>
	);
}

// ExecutionPlanFull renders the full execution plan block of the decision
// card detail per PRD section 5. Unlike ExecutionPlanStrip it never truncates
// and surfaces every step's rationale so the user can understand "why" at
// each trigger. Monitor plans render as two lines (stop-loss / take-profit);
// staged and one-shot plans render numbered step rows.
export function ExecutionPlanFull({ execution }: ExecutionPlanFullProps) {
	const validDaysText = `本计划适用未来 ${execution.validDays} 天，超过或触发止损条件会自动重新分析。`;

	if (execution.type === "monitor") {
		return (
			<Card data-testid="plan-full" title={<Title level={5}>执行计划</Title>}>
				<Space direction="vertical" size={8} style={{ width: "100%" }}>
					<Text data-testid="plan-full-stop-loss">
						止损: {execution.stopLoss != null ? execution.stopLoss : "未设置"}
					</Text>
					<Text data-testid="plan-full-take-profit">
						止盈: {execution.takeProfit != null ? execution.takeProfit : "未设置"}
					</Text>
					<Alert type="warning" showIcon message={validDaysText} style={{ marginTop: 8 }} />
				</Space>
			</Card>
		);
	}

	const steps = execution.steps ?? [];

	return (
		<Card data-testid="plan-full" title={<Title level={5}>执行计划</Title>}>
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				{steps.map((step, idx) => (
					<StepRow key={step.order} step={step} index={idx} />
				))}
				<Alert type="warning" showIcon message={validDaysText} />
			</Space>
		</Card>
	);
}
