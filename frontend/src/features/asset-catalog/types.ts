// Canonical asset type identifiers shared with the backend seed data in
// backend/db/seed/asset_catalog.sql. These four values also double as the
// onboarding category keys (PRD §2.3 step 2).
export type AssetCategory = "gold_etf" | "a_share_broad" | "a_share_industry" | "us_stock";

export const ASSET_CATEGORIES: AssetCategory[] = [
	"gold_etf",
	"a_share_broad",
	"a_share_industry",
	"us_stock",
];

export interface AssetCategoryMeta {
	key: AssetCategory;
	label: string;
	description: string;
	examples: string;
}

// Display metadata for each asset category card used by the onboarding
// categories step and the first-holding quick mode form. Copy lives here so
// the page and the feature share a single source of truth.
export const ASSET_CATEGORY_META: Record<AssetCategory, AssetCategoryMeta> = {
	gold_etf: {
		key: "gold_etf",
		label: "黄金 ETF",
		description: "跟踪黄金价格的 ETF 基金",
		examples: "518880 华安黄金、518800 易方达黄金",
	},
	a_share_broad: {
		key: "a_share_broad",
		label: "A 股宽基 ETF",
		description: "跟踪主要指数的宽基 ETF",
		examples: "510300 沪深 300、510500 中证 500",
	},
	a_share_industry: {
		key: "a_share_industry",
		label: "A 股行业 ETF",
		description: "跟踪特定行业指数的 ETF",
		examples: "512880 证券、515790 光伏",
	},
	us_stock: {
		key: "us_stock",
		label: "美股",
		description: "美国市场的个股与 ETF",
		examples: "AAPL 苹果、TSLA 特斯拉",
	},
};
