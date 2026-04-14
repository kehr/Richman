import type { AssetDetailDto } from "@/features/asset-detail";
import { DimensionPanelList } from "./dimension-panel-list";
import { InterpretationCard } from "./interpretation-card";
import { OhlcvChart } from "./ohlcv-chart";
import { ScoreTrendChart } from "./score-trend-chart";

interface Props {
	detail: AssetDetailDto;
}

// AnalysisTab is the default tab. Every block tolerates absent fields: OHLCV
// overlays default to empty arrays, InterpretationCard hides when no text is
// available, and the dimension list handles an empty dimensions array itself.
export function AnalysisTab({ detail }: Props) {
	return (
		<div style={{ padding: "16px 0" }}>
			<OhlcvChart
				code={detail.code}
				sma200={detail.sma200 ?? null}
				supports={detail.supports ?? []}
				resistances={detail.resistances ?? []}
			/>
			{detail.marketInterpretation && <InterpretationCard text={detail.marketInterpretation} />}
			<DimensionPanelList dimensions={detail.dimensions} />
			<ScoreTrendChart code={detail.code} enabled />
		</div>
	);
}
