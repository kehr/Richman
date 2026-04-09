import { theme } from "@/ui-kit/eat";
import type { ThemeConfig } from "@/ui-kit/eat";

// Palette for a monochrome financial-editorial theme.
//
// Background scale (light → dark):
//   bgContainer  #FFFFFF  — card / form surfaces
//   bgElevated   #FAFAFA  — modals, dropdowns, popovers (sits above layout)
//   bgLayout     #F5F5F5  — page canvas
//   bgHover      #EFEFEF  — hover states (must differ from bgLayout)
//   bgSelected   #E8E8E8  — selected / active states
//
// In a monochrome theme there is no distinct "info blue" — info maps to the
// same black scale as primary. colorInfoBg is pinned explicitly because the
// algorithm generates near-black for black inputs; the correct derived value
// for info backgrounds is the neutral gray scale.
const palette = {
	// Primaries
	black: "#000000",
	ink: "#141414",
	// Text hierarchy
	text: "#1A1A1A",
	textSecondary: "#5C5C5C",
	textTertiary: "#8C8C8C",
	textDisabled: "#BFBFBF",
	// Borders
	border: "#E4E4E4",
	borderSecondary: "#F0F0F0",
	// Backgrounds — each step is intentionally distinct
	bgContainer: "#FFFFFF",
	bgElevated: "#FAFAFA",
	bgLayout: "#F5F5F5",
	bgHover: "#EFEFEF",
	bgSelected: "#E8E8E8",
	// Semantic — info maps to black (monochrome; no distinct "info blue")
	info: "#000000",
	success: "#10B981",
	warning: "#F59E0B",
	error: "#EF4444",
} as const;

export function getThemeConfig(mode: "light" | "dark"): ThemeConfig {
	return {
		token: {
			// --- Brand ---
			colorPrimary: palette.black,
			colorLink: palette.black,

			// --- Semantic ---
			// info maps to black (same as primary) so the monochrome theme stays
			// fully achromatic. colorInfoBg is pinned below to a neutral gray so
			// "processing" tags get a visible background; Alert overrides it back
			// to white at the component level so alert surfaces stay card-like.
			colorInfo: palette.info,
			colorSuccess: palette.success,
			colorWarning: palette.warning,
			colorError: palette.error,

			// --- Typography ---
			// Tabular figures (tnum) keep numbers aligned in data tables.
			// PingFang SC / Hiragino / Microsoft YaHei for CJK fallback.
			fontFamily:
				"'SF Pro Text', -apple-system, BlinkMacSystemFont, 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei', sans-serif",
			fontFamilyCode: "'SF Mono', 'JetBrains Mono', 'Fira Code', Menlo, Consolas, monospace",
			fontSize: 13,
			lineHeight: 1.6,

			// --- Backgrounds ---
			colorBgBase: palette.bgContainer,
			colorBgContainer: palette.bgContainer,
			colorBgElevated: palette.bgElevated,
			colorBgLayout: palette.bgLayout,

			// --- Borders ---
			colorBorder: palette.border,
			colorBorderSecondary: palette.borderSecondary,

			// --- Text ---
			colorTextBase: palette.text,

			// --- Shape ---
			borderRadius: 4,
			borderRadiusSM: 3,
			borderRadiusLG: 6,
			wireframe: false,

			// --- Motion ---
			motionDurationFast: "0.1s",
			motionDurationMid: "0.18s",
			motionDurationSlow: "0.25s",

			// --- Intercept algorithm bg-scale for primary ---
			// colorPrimary = #000000 makes the algorithm generate near-black for
			// colorPrimaryBg (the "selected state" tint). Pin to our neutral scale
			// so every component inherits the correct light-hover behavior without
			// needing per-component patches.
			colorPrimaryBg: palette.bgSelected,
			colorPrimaryBgHover: palette.bgHover,
			// colorInfo = #000000 makes the algorithm generate near-black for
			// colorInfoBg. Pin to bgSelected (#E8E8E8) so "processing" tags render
			// with a clear gray swatch. Alert overrides this back to bgContainer
			// at the component level so info alerts remain card-like on the page.
			colorInfoBg: palette.bgSelected,
			colorInfoBgHover: palette.bgHover,
			colorInfoBorder: palette.border,
			colorInfoBorderHover: palette.border,
		},
		components: {
			Button: {
				// Primary button stays solid black with ink hover.
				colorPrimary: palette.black,
				colorPrimaryHover: palette.ink,
				colorPrimaryActive: palette.black,
				primaryShadow: "none",
				defaultShadow: "none",
				dangerShadow: "none",
			},
			// Horizontal top-nav menu: selection is indicated by the ink bar
			// (colorPrimary = black) + text color only. Background fills create
			// a boxy look in horizontal mode; side nav bg is handled separately
			// via layoutToken.sider which is not affected by this component token.
			// Horizontal top-nav menu: selection is indicated by the ink bar
			// (colorPrimary = black) + text color only. Background fills create
			// a boxy look in horizontal mode; side nav bg is handled separately
			// via layoutToken.sider which is not affected by this component token.
			// itemPaddingInline widens each item internally; itemMarginInline adds
			// breathing room between adjacent items.
			Menu: {
				itemSelectedBg: "transparent",
				itemSelectedColor: palette.black,
				itemHoverBg: "transparent",
				itemHoverColor: palette.black,
				itemActiveBg: "transparent",
				itemPaddingInline: 14,
				itemMarginInline: 4,
			},
			Tabs: {
				itemSelectedColor: palette.black,
				itemHoverColor: palette.ink,
				inkBarColor: palette.black,
			},
			// Tag default color uses colorFillTertiary (~#F0F0F0) which is nearly
			// invisible on a white Card surface. Bump the default background to
			// bgSelected (#E8E8E8) so neutral tags have clear visual weight.
			// Tag default: light gray bg + secondary text keeps tags in the same
			// tonal range — avoids the high-contrast clash of dark-on-gray that
			// reads as muddy. Semantic tags (bullish/bearish) use preset colors.
			Tag: {
				defaultBg: palette.bgSelected,
				defaultColor: palette.textSecondary,
			},
			// Alert info type: override colorInfoBg back to white so the alert
			// surface floats as a card above the #F5F5F5 page canvas. The global
			// colorInfoBg (#E8E8E8) is intentionally left for Tag "processing".
			Alert: {
				colorInfoBg: palette.bgContainer,
				colorInfoBorder: "#CFCFCF",
			},
			// Switch / Checkbox / Radio / Slider only need the hover override;
			// colorPrimary is inherited from the global token automatically.
			Switch: { colorPrimaryHover: palette.ink },
			Checkbox: { colorPrimaryHover: palette.ink },
			Radio: { colorPrimaryHover: palette.ink },
			Slider: { colorPrimaryBorder: palette.black },
		},
		algorithm: mode === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm,
	};
}

// ProLayout-specific tokens. Applied via the `token` prop on ProLayout.
// See https://procomponents.ant.design/components/layout#token
export const layoutToken = {
	bgLayout: palette.bgLayout,
	contentWidth: "fixed",
	sider: {
		colorMenuBackground: palette.bgContainer,
		colorMenuItemDivider: palette.borderSecondary,
		colorBgMenuItemHover: palette.bgHover,
		colorBgMenuItemSelected: palette.bgSelected,
		colorBgMenuItemActive: palette.bgSelected,
		colorBgMenuItemCollapsedElevated: palette.bgContainer,
		colorTextMenu: palette.textSecondary,
		colorTextMenuSelected: palette.black,
		colorTextMenuActive: palette.black,
		colorTextMenuItemHover: palette.black,
		colorTextMenuTitle: palette.black,
		colorTextSubMenuSelected: palette.black,
		colorBgCollapsedButton: palette.bgContainer,
		colorTextCollapsedButton: palette.textSecondary,
		colorTextCollapsedButtonHover: palette.black,
	},
	header: {
		colorBgHeader: palette.bgContainer,
		colorHeaderTitle: palette.black,
		colorBgMenuItemHover: palette.bgHover,
		colorBgMenuItemSelected: palette.bgSelected,
		colorTextMenu: palette.textSecondary,
		colorTextMenuSelected: palette.black,
		colorTextMenuActive: palette.black,
		colorTextRightActionsItem: palette.textSecondary,
		heightLayoutHeader: 56,
	},
	pageContainer: {
		colorBgPageContainer: palette.bgLayout,
		paddingInlinePageContainerContent: 16,
		paddingBlockPageContainerContent: 16,
	},
} as const;
