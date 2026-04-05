import type { ReactNode } from "react";

export const metadata = {
	title: "Richman",
	description: "Personal finance management platform",
};

export default function RootLayout({ children }: { children: ReactNode }) {
	return (
		<html lang="en">
			<body>{children}</body>
		</html>
	);
}
