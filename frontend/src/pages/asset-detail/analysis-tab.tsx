import type { AssetDetailDto } from "@/features/asset-detail";
import { DimensionPanelList } from "./dimension-panel-list";
import { InterpretationCard } from "./interpretation-card";
import { OhlcvChart } from "./ohlcv-chart";
import { ScoreTrendChart } from "./score-trend-chart";

interface Props {
	detail: AssetDetailDto;
}

// AnalysisTab is the default tab. Data is preloaded; no extra guard needed.
export function AnalysisTab({ detail }: Props) {
	return (
		<div style={{ padding: "16px 0" }}>
			<OhlcvChart
				code={detail.code}
				sma200={detail.sma200}
				supports={detail.supports}
				resistances={detail.resistances}
			/>
			<InterpretationCard text={detail.marketInterpretation} />
			<DimensionPanelList dimensions={detail.dimensions} />
			<ScoreTrendChart code={detail.code} enabled />
		</div>
	);
}
