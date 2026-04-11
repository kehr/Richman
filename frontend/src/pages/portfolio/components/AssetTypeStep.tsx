import {
	ASSET_CATEGORIES,
	type AssetCategory,
	type AssetDto,
	useAssets,
} from "@/features/asset-catalog";
import { Empty, Radio, Select, Space, Spin, Typography } from "@/ui-kit/eat";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

// SEARCH_DEBOUNCE_MS is the keystroke-to-fetch delay for the asset search
// box. The backend endpoint is cheap but back-to-back keystrokes still race
// because TanStack Query does not guarantee response ordering, so a short
// debounce both reduces upstream load and prevents stale results from
// flashing in the dropdown.
const SEARCH_DEBOUNCE_MS = 250;

// AssetTypeStep is step 1 of the AddHoldingDrawer (PRD §4.2). The user first
// picks an asset category tab then searches the catalog via a Select with
// server-driven options. Selecting an entry invokes onSelect so the parent
// drawer can move to step 2 with the chosen asset prefilled.

export interface SelectedAssetValue {
	code: string;
	name: string;
	assetType: string;
}

interface AssetTypeStepProps {
	onSelect: (asset: SelectedAssetValue) => void;
}

export function AssetTypeStep({ onSelect }: AssetTypeStepProps) {
	const { t } = useTranslation("app");
	const [category, setCategory] = useState<AssetCategory>(ASSET_CATEGORIES[0]);
	const [keyword, setKeyword] = useState("");
	const [debouncedKeyword, setDebouncedKeyword] = useState(keyword);

	// Debounce the search keyword so rapid typing doesn't fire one fetch
	// per keystroke and the dropdown doesn't flicker with stale results
	// when slower responses arrive after newer ones.
	useEffect(() => {
		const handle = window.setTimeout(() => {
			setDebouncedKeyword(keyword);
		}, SEARCH_DEBOUNCE_MS);
		return () => window.clearTimeout(handle);
	}, [keyword]);

	const { data: assets, isLoading } = useAssets({ type: category, keyword: debouncedKeyword });

	// Build Select options from the latest assets response. useMemo keeps the
	// option array stable across renders when the underlying data does not
	// change, avoiding unnecessary re-renders of the Select internals.
	const options = useMemo(
		() =>
			(assets ?? []).map((a: AssetDto) => ({
				value: a.code,
				label: `${a.code} ${a.name}`,
				asset: a,
			})),
		[assets],
	);

	const handleChange = (code: string) => {
		const picked = options.find((o) => o.value === code)?.asset;
		if (picked) {
			onSelect({
				code: picked.code,
				name: picked.name,
				assetType: picked.assetType,
			});
		}
	};

	return (
		<Space direction="vertical" size="middle" style={{ width: "100%" }}>
			<div>
				<Typography.Text strong>{t("portfolio.assetTypeStep.selectType")}</Typography.Text>
			</div>
			<Radio.Group
				value={category}
				onChange={(e) => setCategory(e.target.value as AssetCategory)}
				// Labels pulled from common:assetCategory.{key}.label — the single
				// source shared with onboarding pages. The explicit ns prefix is
				// required because this component's default useTranslation is "app".
				options={ASSET_CATEGORIES.map((key) => ({
					label: t(`assetCategory.${key}.label`, { ns: "common" }),
					value: key,
					"data-testid": `asset-type-${key}`,
				}))}
				data-testid="asset-type-tabs"
			/>

			<div>
				<Typography.Text strong>{t("portfolio.assetTypeStep.searchAsset")}</Typography.Text>
			</div>
			<Select
				showSearch
				placeholder={t("portfolio.assetTypeStep.searchPlaceholder")}
				value={undefined}
				style={{ width: "100%" }}
				filterOption={false}
				notFoundContent={
					isLoading ? (
						<Spin size="small" />
					) : (
						<Empty description={t("portfolio.assetTypeStep.notFound")} />
					)
				}
				onSearch={setKeyword}
				onChange={handleChange}
				options={options}
				data-testid="asset-select"
			/>

			<Typography.Text type="secondary" style={{ fontSize: 12 }}>
				{t(`assetCategory.${category}.examples`, { ns: "common" })}
			</Typography.Text>
		</Space>
	);
}
