// Canonical asset type identifiers shared with the backend seed data in
// backend/db/seed/asset_catalog.sql. These four values also double as the
// onboarding category keys (PRD §2.3 step 2).
//
// Display copy (label / description / examples) lives in the i18n layer under
// `common:assetCategory.{key}.*` — it is intentionally NOT duplicated here so
// that switching the app language updates every caller uniformly. See
// CategoriesPage, FirstHoldingPage and AssetTypeStep for usage; they read the
// strings via t("assetCategory.${key}.label", { ns: "common" }).
export type AssetCategory = "gold_etf" | "a_share_broad" | "a_share_industry" | "us_stock";

export const ASSET_CATEGORIES: AssetCategory[] = [
	"gold_etf",
	"a_share_broad",
	"a_share_industry",
	"us_stock",
];
