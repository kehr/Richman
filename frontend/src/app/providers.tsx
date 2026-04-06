"use client";

import { createQueryClient } from "@/config/query-client";
import { getThemeConfig } from "@/config/theme";
import { useThemeMode } from "@/domain/ui/use-theme";
import { App as AntApp, ConfigProvider } from "@/ui-kit/eat";
import { QueryClientProvider } from "@tanstack/react-query";
import { type ReactNode, useState } from "react";

export function Providers({ children }: { children: ReactNode }) {
	const [queryClient] = useState(() => createQueryClient());
	const { mode } = useThemeMode();

	return (
		<QueryClientProvider client={queryClient}>
			<ConfigProvider theme={getThemeConfig(mode)}>
				<AntApp>{children}</AntApp>
			</ConfigProvider>
		</QueryClientProvider>
	);
}
