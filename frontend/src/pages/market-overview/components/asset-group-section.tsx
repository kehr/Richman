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
export function AssetGroupSection({ group }: AssetGroupSectionProps) {
	const { i18n } = useTranslation();
	const { token } = useToken();

	// categoryLabel from API is already localized by the backend;
	// fall back to the group.category key if empty.
	const label = group.categoryLabel || group.category;

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
