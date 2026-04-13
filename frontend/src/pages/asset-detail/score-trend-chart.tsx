// score-trend-chart.tsx — echarts line chart for score history over time.

import { useScoreHistory } from "@/features/asset-detail";
import type { ScoreHistoryDays } from "@/features/asset-detail";
import { Alert, Segmented, Skeleton } from "@/ui-kit/eat";
import { Suspense, lazy, useState } from "react";
import { useTranslation } from "react-i18next";

// Lazy load echarts-for-react to avoid bloating the initial bundle.
const ReactECharts = lazy(() => import("echarts-for-react"));

interface Props {
	code: string;
	enabled: boolean;
}

const DAY_OPTIONS: ScoreHistoryDays[] = [30, 90, 180, 240];

export function ScoreTrendChart({ code, enabled }: Props) {
	const { t } = useTranslation("app");
	const [days, setDays] = useState<ScoreHistoryDays>(90);
	const { data, isLoading, isError } = useScoreHistory(code, days, enabled);

	const dayOptions = DAY_OPTIONS.map((d) => ({
		value: d,
		label: t(`assetDetail.scoreTrend.days.${d}`),
	}));

	const points = data?.points ?? [];
	const dates = points.map((p) => p.date);
	const scores = points.map((p) => p.score);

	// Mark version change dates for vertical line annotation.
	const versionChangeIndices = points
		.map((p, i) => (p.versionChange ? i : -1))
		.filter((i) => i >= 0);

	const markLines = versionChangeIndices.map((idx) => ({
		xAxis: dates[idx],
		label: {
			formatter: t("assetDetail.scoreTrend.versionChange"),
			position: "start",
			fontSize: 10,
		},
	}));

	const option = {
		grid: { top: 20, right: 20, bottom: 40, left: 40, containLabel: true },
		xAxis: {
			type: "category",
			data: dates,
			axisLabel: { fontSize: 11, rotate: 30 },
		},
		yAxis: {
			type: "value",
			min: 0,
			max: 100,
			splitLine: { lineStyle: { color: "#f0f0f0" } },
		},
		series: [
			{
				type: "line",
				data: scores,
				smooth: true,
				lineStyle: { color: "#1890ff", width: 2 },
				areaStyle: { color: "rgba(24,144,255,0.08)" },
				symbol: "none",
				markLine: markLines.length
					? {
							data: markLines.map((ml) => [
								{ xAxis: ml.xAxis, label: ml.label },
								{ xAxis: ml.xAxis },
							]),
							lineStyle: { color: "#fa8c16", type: "dashed" },
							symbol: ["none", "none"],
						}
					: undefined,
			},
		],
		tooltip: {
			trigger: "axis",
			formatter: (params: { name: string; value: number }[]) => {
				if (!params.length) return "";
				return `${params[0].name}<br/>Score: ${params[0].value}`;
			},
		},
	};

	return (
		<div style={{ marginTop: 16 }}>
			<div
				style={{
					display: "flex",
					justifyContent: "space-between",
					alignItems: "center",
					marginBottom: 8,
				}}
			>
				<span style={{ fontWeight: 600, fontSize: 14 }}>{t("assetDetail.scoreTrend.title")}</span>
				<Segmented
					options={dayOptions}
					value={days}
					onChange={(v) => setDays(v as ScoreHistoryDays)}
					size="small"
				/>
			</div>
			{isLoading && <Skeleton active style={{ height: 200 }} />}
			{isError && <Alert type="error" message={t("assetDetail.scoreTrend.error")} />}
			{!isLoading && !isError && points.length > 0 && (
				<Suspense fallback={<Skeleton active style={{ height: 200 }} />}>
					<ReactECharts option={option} style={{ height: 200 }} notMerge />
				</Suspense>
			)}
		</div>
	);
}
