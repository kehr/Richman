import "./i18n/config";
import { createQueryClient } from "@/config/query-client";
import { getThemeConfig } from "@/config/theme";
import { useThemeMode } from "@/domain/ui/use-theme";
import { antdLocaleMap } from "@/i18n/antd-locale";
import { App as AntApp, ConfigProvider } from "@/ui-kit/eat";
import { QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { BrowserRouter } from "react-router";
import { AppRoutes } from "./routes";

export function App() {
	const [queryClient] = useState(() => createQueryClient());
	const { mode } = useThemeMode();
	const { i18n } = useTranslation();

	return (
		<QueryClientProvider client={queryClient}>
			<ConfigProvider
				theme={getThemeConfig(mode)}
				locale={antdLocaleMap[i18n.language as "en" | "zh"]}
			>
				<AntApp>
					<BrowserRouter>
						<AppRoutes />
					</BrowserRouter>
				</AntApp>
			</ConfigProvider>
		</QueryClientProvider>
	);
}
