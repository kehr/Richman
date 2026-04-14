import type { AssetGroupDto } from "@/features/market-overview";
import { Typography, theme } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { AssetCard } from "./asset-card";

const { Title } = Typography;
const { useToken } = theme;

interface AssetGroupSectionProps {
	group: AssetGroupDto;
}

// AssetGroupSection renders a category header followed by a responsive grid of AssetCard tiles.
// The section header label is resolved from i18n using `group.assetType` (see
// overview.assetType.<key> in src/i18n/locales/{zh,en}/market.json). When a new
// asset_type appears on the backend without a matching translation key, the
// raw assetType string is rendered as a graceful fallback.
export function AssetGroupSection({ group }: AssetGroupSectionProps) {
	const { t } = useTranslation("market");
	const { token } = useToken();

	const label = t(`overview.assetType.${group.assetType}`, group.assetType);

	return (
		<div style={{ marginBottom: 24 }}>
			<Title
				level={5}
				style={{
					marginBottom: 12,
					marginTop: 0,
					fontSize: 13,
					fontWeight: 600,
					color: token.colorTextSecondary,
					textTransform: "uppercase",
					letterSpacing: "0.04em",
				}}
			>
				{label}
			</Title>

			<div
				style={{
					display: "grid",
					gridTemplateColumns: "repeat(auto-fill, minmax(160px, 1fr))",
					gap: 12,
				}}
			>
				{group.assets.map((asset) => (
					<AssetCard key={asset.code} asset={asset} />
				))}
			</div>
		</div>
	);
}
