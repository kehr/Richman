import { useMoney } from "@/domain/money/useMoney";
import { isStructuredRationale, useFormatTriggerValue } from "@/features/decision-card";
import type { Execution, Step, StructuredRationale } from "@/features/decision-card";
import { Alert, Card, Space, Tag, Typography } from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";

const { Text, Paragraph, Title } = Typography;

interface ExecutionPlanFullProps {
	execution: Execution;
	positionAmountCny?: number | null;
	positionRatioPct?: number;
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

// deltaPctColor follows A-share convention: red = increase (涨/加仓),
// green = decrease (跌/减仓), gray = neutral (0%).
function deltaPctColor(delta: number): string {
	if (delta > 0) return "#f5222d";
	if (delta < 0) return "#52c41a";
	return "#8c8c8c";
}

// RATIONALE_KEYS is the display order for StructuredRationale fields.
const RATIONALE_KEYS: (keyof StructuredRationale)[] = [
	"triggerReason",
	"positionReason",
	"precondition",
	"fallback",
	"timeWindow",
];

// RationaleBlock renders a StructuredRationale as labeled rows, hiding
// empty fields. When rationaleTemplate is provided, text is resolved from
// the i18n bundle instead of the rationale fields (which are empty for
// rules-engine fallback steps). For legacy string rationale, the text is
// rendered as-is.
function RationaleBlock({
	rationale,
	rationaleTemplate,
	stepOrder,
}: {
	rationale: StructuredRationale | string;
	rationaleTemplate?: string;
	stepOrder: number;
}) {
	const { t } = useTranslation("app");

	// Rules-engine fallback: resolve rationale text from i18n bundle.
	// Dynamic key paths cannot be statically typed by the i18n plugin, so the
	// constructed key is cast to satisfy the overloaded t() signature.
	if (rationaleTemplate) {
		const prefix = `decisionCard.executionPlan.fallbackRationale.${rationaleTemplate}`;
		const tDyn = t as (key: string, opts?: Record<string, unknown>) => string;
		const entries = RATIONALE_KEYS.filter(
			(k) => tDyn(`${prefix}.${k}`, { defaultValue: "" }) !== "",
		);
		if (entries.length === 0) return null;
		return (
			<div data-testid={`plan-full-rationale-${stepOrder}`}>
				{entries.map((key) => (
					<div key={key} style={{ marginBottom: 2 }}>
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t(`decisionCard.executionPlan.rationale.${key}`)}:
						</Text>{" "}
						<Text style={{ fontSize: 12 }}>{tDyn(`${prefix}.${key}`)}</Text>
					</div>
				))}
			</div>
		);
	}

	// Legacy string rationale (pre-v2 cards).
	if (typeof rationale === "string") {
		if (!rationale) return null;
		return (
			<Paragraph
				type="secondary"
				style={{ margin: 0, whiteSpace: "pre-wrap" }}
				data-testid={`plan-full-rationale-${stepOrder}`}
			>
				{rationale}
			</Paragraph>
		);
	}

	// LLM-generated structured rationale.
	if (!isStructuredRationale(rationale)) return null;

	const entries = RATIONALE_KEYS.filter((k) => rationale[k]);
	if (entries.length === 0) return null;

	return (
		<div data-testid={`plan-full-rationale-${stepOrder}`}>
			{entries.map((key) => (
				<div key={key} style={{ marginBottom: 2 }}>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t(`decisionCard.executionPlan.rationale.${key}`)}:
					</Text>{" "}
					<Text style={{ fontSize: 12 }}>{rationale[key]}</Text>
				</div>
			))}
		</div>
	);
}

// StepRow renders one full execution step with its trigger condition, delta,
// optional absolute amount, lotCount, and structured rationale fields.
function StepRow({
	step,
	index,
	isMonitor,
	amountStr,
}: {
	step: Step;
	index: number;
	isMonitor: boolean;
	amountStr?: string | null;
}) {
	const { t } = useTranslation("app");
	const formatTrigger = useFormatTriggerValue();
	return (
		<div
			data-testid={`plan-full-step-${step.order}`}
			style={{ display: "flex", alignItems: "flex-start", gap: 12 }}
		>
			<span style={stepCircleStyle(index)}>{step.order}</span>
			<Space direction="vertical" size={4} style={{ flex: 1 }}>
				<div style={{ display: "flex", justifyContent: "space-between", gap: 8 }}>
					<Space size={4}>
						<Text strong>{formatTrigger(step)}</Text>
						{isMonitor && (
							<Tag color="default">{t("decisionCard.executionPlan.monitorStepLabel")}</Tag>
						)}
					</Space>
					<div style={{ display: "flex", flexDirection: "column", alignItems: "flex-end" }}>
						<Text
							strong
							style={{
								fontSize: 18,
								lineHeight: 1.2,
								color: deltaPctColor(step.deltaPct),
							}}
						>
							{formatDeltaPct(step.deltaPct)}
						</Text>
						{amountStr != null && (
							<Text type="secondary" style={{ fontSize: 11, lineHeight: 1.3 }}>
								{amountStr}
							</Text>
						)}
					</div>
				</div>
				{step.lotCount != null && step.lotCount > 0 && (
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t("decisionCard.executionPlan.lotCount")}: {step.lotCount}{" "}
						{t("decisionCard.executionPlan.lotUnit")}
					</Text>
				)}
				<RationaleBlock
					rationale={step.rationale}
					rationaleTemplate={step.rationaleTemplate}
					stepOrder={step.order}
				/>
			</Space>
		</div>
	);
}

// ExecutionPlanFull renders the full execution plan block of the decision
// card detail per PRD section 5. Unlike ExecutionPlanStrip it never truncates
// and surfaces every step's rationale so the user can understand "why" at
// each trigger. Monitor plans render as two lines (stop-loss / take-profit);
// staged and one-shot plans render numbered step rows.
export function ExecutionPlanFull({
	execution,
	positionAmountCny,
	positionRatioPct,
}: ExecutionPlanFullProps) {
	const { t } = useTranslation("app");
	const money = useMoney();
	const validDaysText = t("decisionCard.executionPlan.validDays", { days: execution.validDays });
	const steps = execution.steps ?? [];
	const isMonitor = execution.type === "monitor";

	// Derive total capital to show absolute change amounts per step.
	// Only computed when the user has configured capital (hasCapital).
	const totalCapitalCny =
		money.hasCapital &&
		positionAmountCny != null &&
		positionRatioPct != null &&
		positionRatioPct > 0
			? positionAmountCny / (positionRatioPct / 100)
			: null;

	function stepAmountStr(step: Step): string | null {
		if (totalCapitalCny == null || totalCapitalCny <= 0 || step.deltaPct === 0) return null;
		const amountCny = Math.round(Math.abs((totalCapitalCny * step.deltaPct) / 100));
		return money.formatAmountOnly(amountCny);
	}

	// Legacy monitor cards without steps: render stop-loss / take-profit only.
	if (isMonitor && steps.length === 0) {
		return (
			<Card
				data-testid="plan-full"
				title={<Title level={5}>{t("decisionCard.executionPlan.title")}</Title>}
			>
				<Space direction="vertical" size={8} style={{ width: "100%" }}>
					<Text data-testid="plan-full-stop-loss">
						{t("decisionCard.executionPlan.stopLoss")}:{" "}
						{execution.stopLoss != null
							? execution.stopLoss.toFixed(2)
							: t("decisionCard.executionPlan.notSet")}
					</Text>
					<Text data-testid="plan-full-take-profit">
						{t("decisionCard.executionPlan.takeProfit")}:{" "}
						{execution.takeProfit != null
							? execution.takeProfit.toFixed(2)
							: t("decisionCard.executionPlan.notSet")}
					</Text>
					<Alert type="warning" showIcon message={validDaysText} style={{ marginTop: 8 }} />
				</Space>
			</Card>
		);
	}

	return (
		<Card
			data-testid="plan-full"
			title={<Title level={5}>{t("decisionCard.executionPlan.title")}</Title>}
		>
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				{steps.map((step, idx) => (
					<StepRow
						key={step.order}
						step={step}
						index={idx}
						isMonitor={isMonitor}
						amountStr={stepAmountStr(step)}
					/>
				))}
				{isMonitor && (execution.stopLoss != null || execution.takeProfit != null) && (
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t("decisionCard.executionPlan.stopLoss")}:{" "}
						{execution.stopLoss?.toFixed(2) ?? t("decisionCard.executionPlan.notSet")}
						{" / "}
						{t("decisionCard.executionPlan.takeProfit")}:{" "}
						{execution.takeProfit?.toFixed(2) ?? t("decisionCard.executionPlan.notSet")}
					</Text>
				)}
				<Alert type="warning" showIcon message={validDaysText} />
			</Space>
		</Card>
	);
}
