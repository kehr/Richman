import { formatDuration } from "@/domain/time/format-duration";
import {
	Button,
	CheckCircleOutlined,
	CloseCircleOutlined,
	Divider,
	LoadingOutlined,
	Modal,
	WarningOutlined,
} from "@/ui-kit/eat";
import type React from "react";
import { useTranslation } from "react-i18next";
import type { AnalysisTaskStatus, HoldingProgress } from "../types";
import { useAnalysisTask } from "../use-analysis-task";
import { AnalysisLogPanel } from "./AnalysisLogPanel";
import { AnalysisStepTimeline } from "./AnalysisStepTimeline";

interface AnalysisProgressDrawerProps {
	taskId: string | null;
	open: boolean;
	onClose: () => void;
	// onClear removes the task reference from localStorage and closes the drawer.
	// Provided when the parent wants to offer the user an escape from a stuck task.
	onClear: () => void;
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

// Derive modal title node from task status, with an appropriate icon.
function resolveTitle(
	status: AnalysisTaskStatus | undefined,
	holdings: HoldingProgress[],
	title: string,
	doneClean: string,
	doneDegraded: string,
	failed: string,
): React.ReactNode {
	const iconStyle: React.CSSProperties = { marginRight: 8, fontSize: 16 };
	if (status === "completed") {
		if (isDegraded(holdings)) {
			return (
				<span style={{ color: "#fa8c16" }}>
					<WarningOutlined style={iconStyle} />
					{doneDegraded}
				</span>
			);
		}
		return (
			<span style={{ color: "#52c41a" }}>
				<CheckCircleOutlined style={iconStyle} />
				{doneClean}
			</span>
		);
	}
	if (status === "failed") {
		return (
			<span style={{ color: "#ff4d4f" }}>
				<CloseCircleOutlined style={iconStyle} />
				{failed}
			</span>
		);
	}
	// running / pending
	return (
		<span>
			<LoadingOutlined style={{ ...iconStyle, color: "#1677ff" }} spin />
			{title}
		</span>
	);
}

export function AnalysisProgressDrawer({
	taskId,
	open,
	onClose,
	onClear,
}: AnalysisProgressDrawerProps) {
	const { t } = useTranslation("app");
	const { task } = useAnalysisTask(taskId);

	const holdings = task?.holdings ?? [];
	const steps = task?.steps ?? [];
	const logs = task?.logs ?? [];
	const doneCount = holdings.filter((h) => h.status === "done" || h.status === "failed").length;

	// A task is "orphaned" when the backend restored it from DB after a restart:
	// status is running/pending but holdings list is empty (goroutine is gone).
	const isOrphaned =
		(task?.status === "running" || task?.status === "pending") && holdings.length === 0;

	const title = resolveTitle(
		task?.status,
		holdings,
		t("analysisProgress.title"),
		t("analysisProgress.doneClean"),
		t("analysisProgress.doneDegraded"),
		t("analysisProgress.failed"),
	);

	// Show "放弃追踪" when the task is running/pending so users can always escape.
	const footer =
		task?.status === "running" || task?.status === "pending" ? (
			<div style={{ textAlign: "center", paddingBottom: 4 }}>
				<Button type="link" size="small" danger onClick={onClear}>
					{t("analysisProgress.abandon")}
				</Button>
			</div>
		) : null;

	return (
		<Modal
			open={open}
			onCancel={onClose}
			footer={footer}
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
					{/* Orphaned task warning */}
					{isOrphaned && (
						<div
							style={{
								padding: "10px 14px",
								background: "#fff7e6",
								border: "1px solid #ffd591",
								borderRadius: 6,
								fontSize: 13,
								color: "#d46b08",
							}}
						>
							{t("analysisProgress.interrupted")}
						</div>
					)}

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
								total: holdings.length,
							})}
						</div>
					</div>

					{/* Holdings list */}
					{holdings.length > 0 && (
						<div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
							{holdings.map((h) => (
								<div key={h.symbol} style={{ display: "flex", alignItems: "center", gap: 8 }}>
									<div
										style={{
											width: 8,
											height: 8,
											borderRadius: "50%",
											flexShrink: 0,
											background: holdingDotColor(h.status),
											animation:
												h.status === "running" ? "analysis-pulse 1.5s infinite" : undefined,
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
											{formatDuration(h.durationMs)}
										</span>
									)}
								</div>
							))}
						</div>
					)}

					{/* Step timeline */}
					{steps.length > 0 && (
						<>
							<Divider style={{ margin: 0 }} />
							<AnalysisStepTimeline steps={steps} currentHolding={task.currentHolding} />
						</>
					)}

					{/* Execution logs */}
					{logs.length > 0 && (
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
									<AnalysisLogPanel logs={logs} />
								</div>
							</div>
						</>
					)}
				</div>
			) : (
				<div style={{ textAlign: "center", color: "#888", padding: "24px 0" }}>
					{t("analysisProgress.updating")}
				</div>
			)}
		</Modal>
	);
}
