import { formatDuration } from "@/domain/time/format-duration";
import { useTranslation } from "react-i18next";
import type { AnalysisTaskStep } from "../types";

interface AnalysisStepTimelineProps {
	steps: AnalysisTaskStep[];
	currentHolding: string;
}

// Step status dot sizes: 10x10px circle
const DOT_SIZE = 10;

// Status dot with appropriate color and animation
function StatusDot({ status }: { status: AnalysisTaskStep["status"] }) {
	const baseStyle: React.CSSProperties = {
		width: DOT_SIZE,
		height: DOT_SIZE,
		borderRadius: "50%",
		flexShrink: 0,
		display: "flex",
		alignItems: "center",
		justifyContent: "center",
		fontSize: 7,
		color: "#fff",
		fontWeight: "bold",
	};

	if (status === "pending") {
		return <div style={{ ...baseStyle, background: "#f0f0f0" }} />;
	}
	if (status === "running") {
		return (
			<div
				style={{
					...baseStyle,
					background: "#1677ff",
					animation: "analysis-pulse 1.5s infinite",
					position: "relative",
				}}
			>
				<div
					style={{
						width: 4,
						height: 4,
						borderRadius: "50%",
						background: "#fff",
					}}
				/>
			</div>
		);
	}
	if (status === "done") {
		return <div style={{ ...baseStyle, background: "#52c41a" }}>&#10003;</div>;
	}
	// failed
	return <div style={{ ...baseStyle, background: "#ff4d4f" }}>&#10007;</div>;
}

export function AnalysisStepTimeline({ steps, currentHolding }: AnalysisStepTimelineProps) {
	const { t } = useTranslation("app");

	return (
		<div style={{ padding: "8px 14px" }}>
			<div style={{ fontSize: 12, color: "#888", marginBottom: 8 }}>
				{t("analysisProgress.currentSteps", { name: currentHolding })}
			</div>
			<div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
				{steps.map((step) => (
					<div
						key={step.key}
						style={{
							display: "flex",
							alignItems: "center",
							gap: 8,
							padding: "3px 6px",
							borderRadius: 4,
							background: step.status === "running" ? "#e6f4ff" : "transparent",
						}}
					>
						<StatusDot status={step.status} />
						<span style={{ fontSize: 12, flex: 1, color: "#333" }}>
							{t(`analysisProgress.step.${step.key}`)}
						</span>
						{(step.status === "done" || step.status === "failed") && step.durationMs !== null && (
							<span style={{ fontSize: 11, color: "#bbb" }}>{formatDuration(step.durationMs)}</span>
						)}
					</div>
				))}
			</div>
		</div>
	);
}
