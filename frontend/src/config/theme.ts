import { theme } from "@/ui-kit/eat";
import type { ThemeConfig } from "@/ui-kit/eat";

// Neutral palette tuned for a black primary color.
// Keeps backgrounds near-white and uses grayscale for menu states so the
// primary black reads as a sharp accent instead of a flat background.
const palette = {
	black: "#000000",
	ink: "#141414",
	text: "#1F1F1F",
	textSecondary: "#595959",
	textTertiary: "#8C8C8C",
	border: "#E5E7EB",
	borderSecondary: "#F0F0F0",
	bgLayout: "#F5F5F5",
	bgContainer: "#FFFFFF",
	bgElevated: "#FFFFFF",
	bgHover: "#F5F5F5",
	bgSelected: "#EBEBEB",
	success: "#10B981",
	warning: "#F59E0B",
	error: "#EF4444",
	info: "#000000",
};

export function getThemeConfig(mode: "light" | "dark"): ThemeConfig {
	return {
		token: {
			colorPrimary: palette.black,
			colorInfo: palette.info,
			colorSuccess: palette.success,
			colorWarning: palette.warning,
			colorError: palette.error,
			colorLink: palette.black,
			colorTextBase: palette.text,
			colorBgBase: palette.bgContainer,
			colorBgLayout: palette.bgLayout,
			colorBorder: palette.border,
			colorBorderSecondary: palette.borderSecondary,
			borderRadius: 6,
			wireframe: false,
		},
		components: {
			Button: {
				colorPrimary: palette.black,
				colorPrimaryHover: palette.ink,
				colorPrimaryActive: palette.black,
				primaryShadow: "none",
			},
			Menu: {
				itemSelectedBg: palette.bgSelected,
				itemSelectedColor: palette.black,
				itemHoverBg: palette.bgHover,
				itemHoverColor: palette.black,
				itemActiveBg: palette.bgSelected,
			},
			Tabs: {
				itemSelectedColor: palette.black,
				itemHoverColor: palette.ink,
				inkBarColor: palette.black,
			},
			Switch: {
				colorPrimary: palette.black,
				colorPrimaryHover: palette.ink,
			},
			Checkbox: {
				colorPrimary: palette.black,
				colorPrimaryHover: palette.ink,
			},
			Radio: {
				colorPrimary: palette.black,
				colorPrimaryHover: palette.ink,
			},
			Slider: {
				colorPrimary: palette.black,
				colorPrimaryBorder: palette.black,
			},
		},
		algorithm: mode === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm,
	};
}

// ProLayout-specific tokens. Applied via the `token` prop on ProLayout so the
// sider, header, and page container pick up the same black-neutral palette as
// the global antd theme. See https://procomponents.ant.design/components/layout#token
export const layoutToken = {
	bgLayout: palette.bgLayout,
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
		paddingInlinePageContainerContent: 24,
		paddingBlockPageContainerContent: 16,
	},
} as const;
