// ohlcv-chart.tsx — candlestick chart using lightweight-charts v4.
// lazy-loaded to avoid blocking initial render.

import { useAssetOhlcv } from "@/features/asset-detail";
import type { OhlcvPeriod } from "@/features/asset-detail";
import { Alert, Segmented, Skeleton } from "@/ui-kit/eat";
import { LineStyle, createChart } from "lightweight-charts";
import type { IChartApi } from "lightweight-charts";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

interface Props {
	code: string;
	sma200: number | null;
	supports: number[];
	resistances: number[];
}

const PERIODS: OhlcvPeriod[] = ["1D", "1W", "1M", "3M", "1Y"];

export function OhlcvChart({ code, sma200, supports, resistances }: Props) {
	const { t } = useTranslation("app");
	const [period, setPeriod] = useState<OhlcvPeriod>("3M");
	const containerRef = useRef<HTMLDivElement>(null);
	const chartRef = useRef<IChartApi | null>(null);

	const { data, isLoading, isError } = useAssetOhlcv(code, period);

	// Initialize chart on mount, clean up on unmount.
	useEffect(() => {
		if (!containerRef.current) return;
		const chart = createChart(containerRef.current, {
			autoSize: true,
			layout: { background: { color: "transparent" }, textColor: "#595959" },
			grid: { vertLines: { color: "#f0f0f0" }, horzLines: { color: "#f0f0f0" } },
			rightPriceScale: { borderColor: "#d9d9d9" },
			timeScale: { borderColor: "#d9d9d9", timeVisible: true },
		});
		chartRef.current = chart;
		return () => {
			chart.remove();
			chartRef.current = null;
		};
	}, []);

	// Render data whenever it changes.
	useEffect(() => {
		const chart = chartRef.current;
		if (!chart || !data?.bars?.length) return;

		// Remove all existing series by recreating; lightweight-charts v4 does
		// not expose a clearSeries() helper, so we track series externally.
		const candleSeries = chart.addCandlestickSeries({
			upColor: "#52c41a",
			downColor: "#f5222d",
			borderUpColor: "#52c41a",
			borderDownColor: "#f5222d",
			wickUpColor: "#52c41a",
			wickDownColor: "#f5222d",
		});

		const bars = data.bars.map((b) => ({
			time: b.time as `${string}-${string}-${string}`,
			open: b.open,
			high: b.high,
			low: b.low,
			close: b.close,
		}));
		candleSeries.setData(bars);

		// SMA-200 line overlay
		if (sma200 !== null) {
			candleSeries.createPriceLine({
				price: sma200,
				color: "#1890ff",
				lineWidth: 1,
				lineStyle: LineStyle.Dashed,
				axisLabelVisible: true,
				title: t("assetDetail.ohlcvChart.sma200"),
			});
		}

		// Support levels
		for (const s of supports) {
			candleSeries.createPriceLine({
				price: s,
				color: "#52c41a",
				lineWidth: 1,
				lineStyle: LineStyle.Dotted,
				axisLabelVisible: false,
				title: t("assetDetail.ohlcvChart.support"),
			});
		}

		// Resistance levels
		for (const r of resistances) {
			candleSeries.createPriceLine({
				price: r,
				color: "#f5222d",
				lineWidth: 1,
				lineStyle: LineStyle.Dotted,
				axisLabelVisible: false,
				title: t("assetDetail.ohlcvChart.resistance"),
			});
		}

		chart.timeScale().fitContent();

		// Cleanup series before next render cycle.
		return () => {
			chart.removeSeries(candleSeries);
		};
	}, [data, sma200, supports, resistances, t]);

	const periodOptions = PERIODS.map((p) => ({
		value: p,
		label: t(`assetDetail.ohlcvChart.period.${p}`),
	}));

	return (
		<div>
			<div style={{ marginBottom: 8 }}>
				<Segmented
					options={periodOptions}
					value={period}
					onChange={(v) => setPeriod(v as OhlcvPeriod)}
					size="small"
				/>
			</div>
			{isLoading && <Skeleton active style={{ height: 300 }} />}
			{isError && <Alert type="error" message={t("assetDetail.ohlcvChart.error")} />}
			<div
				ref={containerRef}
				style={{
					height: 300,
					display: isLoading || isError ? "none" : "block",
				}}
			/>
		</div>
	);
}
