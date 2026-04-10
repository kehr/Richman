import { useTranslation } from "react-i18next";
import { Drawer } from "@/ui-kit/eat";
import { useAnalysisTask } from "../use-analysis-task";
import type { AnalysisTaskStatus, HoldingProgress } from "../types";
import { AnalysisStepTimeline } from "./AnalysisStepTimeline";
import { AnalysisLogPanel } from "./AnalysisLogPanel";

interface AnalysisProgressDrawerProps {
	taskId: string | null;
	open: boolean;
	onClose: () => void;
}

// Holding status dot color mapping
function holdingDotColor(status: HoldingProgress["status"]): string {
	if (status === "running") return "#1677ff";
	if (status === "done") return "#52c41a";
	if (status === "failed") return "#ff4d4f";
	return "#f0f0f0";
}

// Determine whether a completed task has any degraded holdings
function isDegraded(holdings: HoldingProgress[]): boolean {
	return holdings.some(
		(h) => h.synthesisSource === "template" || h.synthesisSource === "mixed",
	);
}

// Header bar rendered for running/pending state
function RunningHeader({
	onClose,
	label,
}: {
	onClose: () => void;
	label: string;
}) {
	const { t } = useTranslation("app");

	return (
		<div
			style={{
				flex: "0 0 auto",
				padding: "12px 14px",
				display: "flex",
				alignItems: "center",
				justifyContent: "space-between",
				borderBottom: "1px solid #f0f0f0",
			}}
		>
			<span style={{ fontSize: 13, fontWeight: 500, color: "#333" }}>{label}</span>
			<button
				type="button"
				onClick={onClose}
				style={{
					background: "none",
					border: "none",
					cursor: "pointer",
					fontSize: 12,
					color: "#888",
					padding: "2px 6px",
				}}
			>
				{t("analysisProgress.collapse")}
			</button>
		</div>
	);
}

// Header bar rendered for terminal states (completed / failed)
function TerminalHeader({
	onClose,
	label,
	bg,
	textColor,
	btnColor,
}: {
	onClose: () => void;
	label: string;
	bg: string;
	textColor: string;
	btnColor: string;
}) {
	const { t } = useTranslation("app");

	return (
		<div
			style={{
				flex: "0 0 auto",
				padding: "12px 14px",
				display: "flex",
				alignItems: "center",
				justifyContent: "space-between",
				background: bg,
			}}
		>
			<span style={{ fontSize: 13, fontWeight: 500, color: textColor }}>{label}</span>
			<button
				type="button"
				onClick={onClose}
				style={{
					background: "none",
					border: "none",
					cursor: "pointer",
					fontSize: 12,
					color: btnColor,
					padding: "2px 6px",
				}}
			>
				{t("analysisProgress.close")}
			</button>
		</div>
	);
}

// Derive header props from task status
function resolveHeaderVariant(
	status: AnalysisTaskStatus | undefined,
	holdings: HoldingProgress[],
	title: string,
	doneClean: string,
	doneDegraded: string,
	failed: string,
):
	| { kind: "running"; label: string }
	| { kind: "terminal"; label: string; bg: string; textColor: string; btnColor: string } {
	if (status === "completed") {
		if (isDegraded(holdings)) {
			return {
				kind: "terminal",
				label: doneDegraded,
				bg: "#fff7e6",
				textColor: "#fa8c16",
				btnColor: "#fa8c16",
			};
		}
		return {
			kind: "terminal",
			label: doneClean,
			bg: "#f6ffed",
			textColor: "#52c41a",
			btnColor: "#52c41a",
		};
	}
	if (status === "failed") {
		return {
			kind: "terminal",
			label: failed,
			bg: "#fff1f0",
			textColor: "#ff4d4f",
			btnColor: "#ff4d4f",
		};
	}
	// pending / running / undefined
	return { kind: "running", label: title };
}

export function AnalysisProgressDrawer({
	taskId,
	open,
	onClose,
}: AnalysisProgressDrawerProps) {
	const { t } = useTranslation("app");
	const { task } = useAnalysisTask(taskId);

	const holdings = task?.holdings ?? [];
	const doneCount = holdings.filter(
		(h) => h.status === "done" || h.status === "failed",
	).length;

	const headerVariant = resolveHeaderVariant(
		task?.status,
		holdings,
		t("analysisProgress.title"),
		t("analysisProgress.doneClean"),
		t("analysisProgress.doneDegraded"),
		t("analysisProgress.failed"),
	);

	return (
		<Drawer
			open={open}
			placement="right"
			mask={false}
			width={280}
			closable={false}
			styles={{
				body: {
					padding: 0,
					display: "flex",
					flexDirection: "column",
					height: "100%",
				},
			}}
		>
			{/* Inject pulse animation for running step dots */}
			<style>{`
				@keyframes analysis-pulse {
					0%   { box-shadow: 0 0 0 0 rgba(22, 119, 255, 0.4); }
					70%  { box-shadow: 0 0 0 6px rgba(22, 119, 255, 0); }
					100% { box-shadow: 0 0 0 0 rgba(22, 119, 255, 0); }
				}
			`}</style>

			{/* Region 1: Header */}
			{headerVariant.kind === "running" ? (
				<RunningHeader onClose={onClose} label={headerVariant.label} />
			) : (
				<TerminalHeader
					onClose={onClose}
					label={headerVariant.label}
					bg={headerVariant.bg}
					textColor={headerVariant.textColor}
					btnColor={headerVariant.btnColor}
				/>
			)}

			{/* Region 2: Overall progress */}
			{task && (
				<div style={{ flex: "0 0 auto", padding: "0 14px 12px" }}>
					{/* Progress bar */}
					<div
						style={{
							height: 4,
							background: "#f0f0f0",
							borderRadius: 2,
							overflow: "hidden",
							margin: "10px 0 8px",
						}}
					>
						<div
							style={{
								height: "100%",
								background: "#1677ff",
								width: `${Math.round((task.progress ?? 0) * 100)}%`,
								transition: "width 0.5s ease",
								borderRadius: 2,
							}}
						/>
					</div>

					{/* Holdings count */}
					<div style={{ fontSize: 12, color: "#888", marginBottom: 8 }}>
						{t("analysisProgress.cardCount", {
							done: doneCount,
							total: task.holdings.length,
						})}
					</div>

					{/* Holdings list */}
					<div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
						{task.holdings.map((h) => (
							<div
								key={h.symbol}
								style={{
									display: "flex",
									alignItems: "center",
									gap: 8,
								}}
							>
								{/* Status dot */}
								<div
									style={{
										width: 8,
										height: 8,
										borderRadius: "50%",
										flexShrink: 0,
										background: holdingDotColor(h.status),
									}}
								/>
								{/* Holding name */}
								<span
									style={{
										fontSize: 12,
										color: "#333",
										flex: 1,
										overflow: "hidden",
										textOverflow: "ellipsis",
										whiteSpace: "nowrap",
									}}
								>
									{h.name}
								</span>
								{/* Source label — only show when done */}
								{h.status === "done" &&
									h.synthesisSource !== null &&
									h.synthesisSource !== "unknown" && (
										<span style={{ fontSize: 11, color: "#bbb", flexShrink: 0 }}>
											{t(
												`analysisProgress.source.${h.synthesisSource as "llm" | "template" | "mixed"}`,
											)}
										</span>
									)}
							</div>
						))}
					</div>
				</div>
			)}

			{/* Region 3: Step timeline */}
			{task && task.steps.length > 0 && (
				<div style={{ flex: "0 0 auto" }}>
					<AnalysisStepTimeline
						steps={task.steps}
						currentHolding={task.currentHolding}
					/>
				</div>
			)}

			{/* Region 4: Log panel */}
			<div
				style={{
					flex: "1 1 0",
					overflow: "hidden",
					display: "flex",
					flexDirection: "column",
				}}
			>
				<div
					style={{
						fontSize: 11,
						color: "#888",
						padding: "6px 14px 2px",
						flexShrink: 0,
					}}
				>
					{t("analysisProgress.logs")}
				</div>
				<AnalysisLogPanel logs={task?.logs ?? []} />
			</div>
		</Drawer>
	);
}
