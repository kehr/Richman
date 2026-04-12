import { type IChartApi, type ISeriesApi, LineStyle, createChart } from "lightweight-charts";
import { useEffect, useRef } from "react";
import type { PriceLine, QuoteHistoryPoint, TimeMarker } from "../types";

interface MarketQuoteChartProps {
	history: QuoteHistoryPoint[];
	priceLines: PriceLine[];
	timeMarkers: TimeMarker[];
	height?: number;
}

// MarketQuoteChart renders a lightweight-charts line chart with price overlays.
// It is a pure visual component with no business logic.
export function MarketQuoteChart({
	history,
	priceLines,
	timeMarkers,
	height = 160,
}: MarketQuoteChartProps) {
	const containerRef = useRef<HTMLDivElement>(null);
	const chartRef = useRef<IChartApi | null>(null);
	const seriesRef = useRef<ISeriesApi<"Line"> | null>(null);

	// Initialize chart on mount, destroy on unmount.
	useEffect(() => {
		const container = containerRef.current;
		if (!container) return;

		const chart = createChart(container, {
			height,
			layout: {
				background: { color: "transparent" },
				textColor: "#666",
				fontSize: 11,
			},
			grid: {
				vertLines: { visible: false },
				horzLines: { color: "#f0f0f0" },
			},
			rightPriceScale: {
				borderVisible: false,
			},
			timeScale: {
				borderVisible: false,
				fixLeftEdge: true,
				fixRightEdge: true,
			},
			crosshair: {
				horzLine: { visible: false },
			},
			handleScroll: false,
			handleScale: false,
		});

		// v4 API: chart.addLineSeries(options)
		const series = chart.addLineSeries({
			color: "#1677ff",
			lineWidth: 2,
			priceLineVisible: false,
			lastValueVisible: true,
		});

		chartRef.current = chart;
		seriesRef.current = series;

		// Resize observer for responsive width.
		const ro = new ResizeObserver((entries) => {
			const entry = entries[0];
			if (entry) {
				chart.applyOptions({ width: entry.contentRect.width });
			}
		});
		ro.observe(container);

		return () => {
			ro.disconnect();
			chart.remove();
			chartRef.current = null;
			seriesRef.current = null;
		};
	}, [height]);

	// Update data when history changes.
	useEffect(() => {
		const series = seriesRef.current;
		if (!series || history.length === 0) return;

		const data = history.map((p) => ({
			time: p.date.slice(0, 10) as string,
			value: p.close,
		}));

		series.setData(data);
		chartRef.current?.timeScale().fitContent();
	}, [history]);

	// Update price lines when overlays change.
	useEffect(() => {
		const series = seriesRef.current;
		if (!series) return;

		// Track created price line instances so they can be removed on cleanup.
		const lineInstances: ReturnType<typeof series.createPriceLine>[] = [];

		for (const pl of priceLines) {
			const instance = series.createPriceLine({
				price: pl.price,
				color: pl.color,
				lineWidth: 1,
				lineStyle: pl.lineStyle === "dashed" ? LineStyle.Dashed : LineStyle.Solid,
				axisLabelVisible: true,
				title: pl.label,
			});
			lineInstances.push(instance);
		}

		return () => {
			for (const inst of lineInstances) {
				try {
					series.removePriceLine(inst);
				} catch {
					// series may have been removed already
				}
			}
		};
	}, [priceLines]);

	// Update time markers when they change.
	useEffect(() => {
		const series = seriesRef.current;
		if (!series || timeMarkers.length === 0) return;

		// v4 API: series.setMarkers(markers)
		const markers = timeMarkers.map((tm) => ({
			time: tm.time.slice(0, 10) as string,
			position: "aboveBar" as const,
			color: tm.color,
			shape: "circle" as const,
			text: tm.label,
		}));

		series.setMarkers(markers);

		return () => {
			series.setMarkers([]);
		};
	}, [timeMarkers]);

	return <div ref={containerRef} style={{ width: "100%" }} />;
}
