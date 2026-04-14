import type { AssetDetailDto } from "@/features/asset-detail";
import { Divider, theme } from "@/ui-kit/eat";
import { AssetIdentity } from "./asset-identity";
import { ChangeSummary } from "./change-summary";
import { ConflictWarning } from "./conflict-warning";
import { FreshnessIndicator } from "./freshness-indicator";
import { MajorChangeRecap } from "./major-change-recap";
import { ScoreBar } from "./score-bar";
import { ScoreSummary } from "./score-summary";
import { computePriceDriftPercent } from "./utils";

const { useToken } = theme;

interface Props {
	detail: AssetDetailDto;
}

export function StickyHeader({ detail }: Props) {
	const { token } = useToken();
	const drift = computePriceDriftPercent(detail.currentPrice, detail.priceAtAnalysis);
	const scoreDelta = detail.scoreDelta ?? 0;

	return (
		<div
			style={{
				position: "sticky",
				top: 0,
				zIndex: 10,
				background: token.colorBgContainer,
				borderBottom: `1px solid ${token.colorBorder}`,
				padding: "12px 16px 8px",
				backdropFilter: "blur(8px)",
			}}
		>
			<AssetIdentity
				code={detail.code}
				name={detail.name}
				nameEn={detail.nameEn}
				price={detail.currentPrice}
				currency={detail.currency}
				changePercent={detail.priceChangePercent}
			/>
			<Divider style={{ margin: "6px 0" }} />
			<ScoreSummary
				score={detail.overallScore}
				signal={detail.signalLevel}
				percentileLabel={detail.percentileLabel}
			/>
			<div style={{ marginTop: 6 }}>
				<ScoreBar
					score={detail.overallScore}
					bandLow={detail.scoreBandLow}
					bandHigh={detail.scoreBandHigh}
				/>
			</div>
			{scoreDelta >= 5 && detail.changeSummary && <ChangeSummary text={detail.changeSummary} />}
			{Math.abs(scoreDelta) > 20 && detail.majorChangeRecap && (
				<MajorChangeRecap recap={detail.majorChangeRecap} />
			)}
			{detail.conflictType && detail.conflictMessage && (
				<ConflictWarning type={detail.conflictType} message={detail.conflictMessage} />
			)}
			{drift > 2 && detail.analyzedAt && (
				<FreshnessIndicator drift={drift} analysisTime={detail.analyzedAt} />
			)}
		</div>
	);
}
