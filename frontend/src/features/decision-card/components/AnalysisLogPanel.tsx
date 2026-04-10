import { useEffect, useRef } from "react";
import type { AnalysisTaskLog } from "../types";

interface AnalysisLogPanelProps {
	logs: AnalysisTaskLog[];
}

function parseTime(ts: string): string {
	const d = new Date(ts);
	const hh = String(d.getHours()).padStart(2, "0");
	const mm = String(d.getMinutes()).padStart(2, "0");
	const ss = String(d.getSeconds()).padStart(2, "0");
	return `${hh}:${mm}:${ss}`;
}

function levelColor(level: AnalysisTaskLog["level"]): string {
	if (level === "warn") return "#fa8c16";
	if (level === "error") return "#ff4d4f";
	return "#555";
}

export function AnalysisLogPanel({ logs }: AnalysisLogPanelProps) {
	const containerRef = useRef<HTMLDivElement>(null);

	// biome-ignore lint/correctness/useExhaustiveDependencies: logs triggers scroll-to-bottom on new entries; containerRef.current mutation is intentional
	useEffect(() => {
		if (containerRef.current) {
			containerRef.current.scrollTop = containerRef.current.scrollHeight;
		}
	}, [logs]);

	return (
		<div
			ref={containerRef}
			style={{
				flex: "1 1 0",
				overflowY: "auto",
				padding: "8px 14px",
				fontFamily: "monospace",
				fontSize: 12,
				lineHeight: 1.7,
			}}
		>
			{logs.map((log, i) => (
				// index-based key is intentional: logs are append-only and never reordered
				// biome-ignore lint/suspicious/noArrayIndexKey: append-only log list
				<div key={i} style={{ color: levelColor(log.level) }}>
					<span style={{ color: "#bbb", marginRight: 6 }}>{parseTime(log.ts)}</span>
					{log.msg}
				</div>
			))}
		</div>
	);
}
