import { resources } from "@/i18n/config";
import { App as AntApp, ConfigProvider } from "@/ui-kit/eat";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render } from "@testing-library/react";
import i18n from "i18next";
import type { ReactElement } from "react";
import { I18nextProvider, initReactI18next } from "react-i18next";

// Create a test-only i18n instance (isolated from app singleton)
const testI18n = i18n.createInstance();
testI18n.use(initReactI18next).init({
	resources,
	lng: "en",
	fallbackLng: "en",
	ns: ["common", "auth", "app", "settings"],
	defaultNS: "common",
	interpolation: { escapeValue: false },
	react: { useSuspense: false },
});

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
				<I18nextProvider i18n={testI18n}>
					<ConfigProvider>
						<AntApp>{ui}</AntApp>
					</ConfigProvider>
				</I18nextProvider>
			</QueryClientProvider>,
		),
	};
}
