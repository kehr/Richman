import { Divider, Modal } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import type { AnalysisTaskStatus, HoldingProgress } from "../types";
import { useAnalysisTask } from "../use-analysis-task";
import { AnalysisLogPanel } from "./AnalysisLogPanel";
import { AnalysisStepTimeline } from "./AnalysisStepTimeline";

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
	return "#d9d9d9";
}

// Determine whether a completed task has any degraded holdings
function isDegraded(holdings: HoldingProgress[]): boolean {
	return holdings.some((h) => h.synthesisSource === "template" || h.synthesisSource === "mixed");
}

// Derive modal title content from task status
function resolveTitle(
	status: AnalysisTaskStatus | undefined,
	holdings: HoldingProgress[],
	title: string,
	doneClean: string,
	doneDegraded: string,
	failed: string,
): { label: string; color?: string } {
	if (status === "completed") {
		if (isDegraded(holdings)) {
			return { label: doneDegraded, color: "#fa8c16" };
		}
		return { label: doneClean, color: "#52c41a" };
	}
	if (status === "failed") {
		return { label: failed, color: "#ff4d4f" };
	}
	return { label: title };
}

export function AnalysisProgressDrawer({ taskId, open, onClose }: AnalysisProgressDrawerProps) {
	const { t } = useTranslation("app");
	const { task } = useAnalysisTask(taskId);

	const holdings = task?.holdings ?? [];
	const doneCount = holdings.filter((h) => h.status === "done" || h.status === "failed").length;

	const titleMeta = resolveTitle(
		task?.status,
		holdings,
		t("analysisProgress.title"),
		t("analysisProgress.doneClean"),
		t("analysisProgress.doneDegraded"),
		t("analysisProgress.failed"),
	);

	const title = titleMeta.color ? (
		<span style={{ color: titleMeta.color }}>{titleMeta.label}</span>
	) : (
		titleMeta.label
	);

	return (
		<Modal
			open={open}
			onCancel={onClose}
			footer={null}
			width={600}
			title={title}
			destroyOnClose={false}
			styles={{
				body: {
					maxHeight: "calc(80vh - 110px)",
					overflowY: "auto",
					paddingTop: 8,
				},
			}}
		>
			{task ? (
				<div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
					{/* Progress bar + card count */}
					<div>
						<div
							style={{
								height: 4,
								background: "#f0f0f0",
								borderRadius: 2,
								overflow: "hidden",
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
						<div style={{ fontSize: 12, color: "#888", marginTop: 6 }}>
							{t("analysisProgress.cardCount", {
								done: doneCount,
								total: task.holdings.length,
							})}
						</div>
					</div>

					{/* Holdings list */}
					<div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
						{task.holdings.map((h) => (
							<div key={h.symbol} style={{ display: "flex", alignItems: "center", gap: 8 }}>
								<div
									style={{
										width: 8,
										height: 8,
										borderRadius: "50%",
										flexShrink: 0,
										background: holdingDotColor(h.status),
										animation: h.status === "running" ? "analysis-pulse 1.5s infinite" : undefined,
									}}
								/>
								<span
									style={{
										fontSize: 13,
										color: "#333",
										flex: 1,
										overflow: "hidden",
										textOverflow: "ellipsis",
										whiteSpace: "nowrap",
									}}
								>
									{h.name}
								</span>
								{h.status === "done" &&
									h.synthesisSource !== null &&
									h.synthesisSource !== "unknown" && (
										<span style={{ fontSize: 12, color: "#bbb", flexShrink: 0 }}>
											{t(
												`analysisProgress.source.${h.synthesisSource as "llm" | "template" | "mixed"}`,
											)}
										</span>
									)}
								{h.durationMs !== null && h.status === "done" && (
									<span style={{ fontSize: 11, color: "#d9d9d9", flexShrink: 0 }}>
										{h.durationMs}ms
									</span>
								)}
							</div>
						))}
					</div>

					{/* Step timeline */}
					{task.steps.length > 0 && (
						<>
							<Divider style={{ margin: 0 }} />
							<AnalysisStepTimeline steps={task.steps} currentHolding={task.currentHolding} />
						</>
					)}

					{/* Execution logs */}
					<>
						<Divider style={{ margin: 0 }} />
						<div>
							<div style={{ fontSize: 12, color: "#888", marginBottom: 6 }}>
								{t("analysisProgress.logs")}
							</div>
							<div
								style={{
									height: 200,
									display: "flex",
									flexDirection: "column",
									background: "#fafafa",
									borderRadius: 6,
									overflow: "hidden",
								}}
							>
								<AnalysisLogPanel logs={task.logs} />
							</div>
						</div>
					</>
				</div>
			) : (
				<div style={{ textAlign: "center", color: "#888", padding: "24px 0" }}>
					{t("analysisProgress.updating")}
				</div>
			)}
		</Modal>
	);
}
