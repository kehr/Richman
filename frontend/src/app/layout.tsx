import type { ReactNode } from "react";
import { Providers } from "./providers";

export const metadata = {
	title: "Richman",
	description: "Personal finance management platform",
};

export default function RootLayout({ children }: { children: ReactNode }) {
	return (
		<html lang="en" suppressHydrationWarning>
			<body style={{ margin: 0 }}>
				<Providers>{children}</Providers>
			</body>
		</html>
	);
}
