import type { MarketOverviewDto } from "@/features/market-overview";
import { Skeleton } from "@/ui-kit/eat";
import { AssetGroupSection } from "./asset-group-section";

interface AssetCardWallProps {
	data: MarketOverviewDto | undefined;
	isLoading: boolean;
}

// AssetCardWall renders grouped asset cards in order defined by the API response
// (commodity -> equity -> fixed income -> digital assets per TRD SS4.3).
export function AssetCardWall({ data, isLoading }: AssetCardWallProps) {
	if (isLoading) {
		return (
			<div style={{ marginBottom: 24 }}>
				<Skeleton active paragraph={{ rows: 4 }} />
			</div>
		);
	}

	if (!data || data.groups.length === 0) {
		return null;
	}

	return (
		<div>
			{data.groups.map((group) => (
				<AssetGroupSection key={group.assetType} group={group} />
			))}
		</div>
	);
}
