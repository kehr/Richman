import {
	ASSET_CATEGORIES,
	ASSET_CATEGORY_META,
	type AssetCategory,
	type AssetDto,
	useAssets,
} from "@/features/asset-catalog";
import { Empty, Radio, Select, Space, Spin, Typography } from "@/ui-kit/eat";
import { useMemo, useState } from "react";

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
	const [category, setCategory] = useState<AssetCategory>(ASSET_CATEGORIES[0]);
	const [keyword, setKeyword] = useState("");

	const { data: assets, isLoading } = useAssets({ type: category, keyword });

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
				<Typography.Text strong>选择标的类型</Typography.Text>
			</div>
			<Radio.Group
				value={category}
				onChange={(e) => setCategory(e.target.value as AssetCategory)}
				optionType="button"
				buttonStyle="solid"
				data-testid="asset-type-tabs"
			>
				{ASSET_CATEGORIES.map((key) => (
					<Radio.Button key={key} value={key} data-testid={`asset-type-${key}`}>
						{ASSET_CATEGORY_META[key].label}
					</Radio.Button>
				))}
			</Radio.Group>

			<div>
				<Typography.Text strong>搜索标的</Typography.Text>
			</div>
			<Select
				showSearch
				placeholder="输入代码或名称"
				value={undefined}
				style={{ width: "100%" }}
				filterOption={false}
				notFoundContent={isLoading ? <Spin size="small" /> : <Empty description="未找到标的" />}
				onSearch={setKeyword}
				onChange={handleChange}
				options={options}
				data-testid="asset-select"
			/>

			<Typography.Text type="secondary" style={{ fontSize: 12 }}>
				{ASSET_CATEGORY_META[category].examples}
			</Typography.Text>
		</Space>
	);
}
