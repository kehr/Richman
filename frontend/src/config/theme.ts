import { theme } from "@/ui-kit/eat";
import type { ThemeConfig } from "@/ui-kit/eat";

export function getThemeConfig(mode: "light" | "dark"): ThemeConfig {
	return {
		token: {
			colorPrimary: "#2563EB",
			borderRadius: 4,
		},
		algorithm: mode === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm,
	};
}
