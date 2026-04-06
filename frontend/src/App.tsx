import { createQueryClient } from "@/config/query-client";
import { getThemeConfig } from "@/config/theme";
import { I18nProvider } from "@/domain/i18n/provider";
import { useThemeMode } from "@/domain/ui/use-theme";
import { App as AntApp, ConfigProvider } from "@/ui-kit/eat";
import { QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { BrowserRouter } from "react-router";
import { AppRoutes } from "./routes";

export function App() {
	const [queryClient] = useState(() => createQueryClient());
	const { mode } = useThemeMode();

	return (
		<QueryClientProvider client={queryClient}>
			<ConfigProvider theme={getThemeConfig(mode)}>
				<AntApp>
					<I18nProvider>
						<BrowserRouter>
							<AppRoutes />
						</BrowserRouter>
					</I18nProvider>
				</AntApp>
			</ConfigProvider>
		</QueryClientProvider>
	);
}
