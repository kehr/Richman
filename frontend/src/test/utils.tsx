import { I18nProvider } from "@/domain/i18n/provider";
import type { ReactElement } from "react";
import { App as AntApp, ConfigProvider } from "@/ui-kit/eat";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";

export function renderWithProviders(ui: ReactElement) {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				retry: false,
			},
		},
	});

	return {
		queryClient,
		...render(
			<QueryClientProvider client={queryClient}>
				<ConfigProvider>
					<AntApp>
						<I18nProvider>{ui}</I18nProvider>
					</AntApp>
				</ConfigProvider>
			</QueryClientProvider>,
		),
	};
}
