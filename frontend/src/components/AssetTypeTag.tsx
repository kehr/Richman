import { Tag } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface AssetTypeTagProps {
	assetType: string;
}

// AssetTypeTag renders the asset type as a localised Tag, falling back to the
// raw value if no translation is found (forward-compatible with new types).
export function AssetTypeTag({ assetType }: AssetTypeTagProps) {
	const { t } = useTranslation("app");
	return <Tag>{t(`portfolio.assetTypes.${assetType}`, { defaultValue: assetType })}</Tag>;
}
