"use client";

import { AuthGuard } from "@/domain/auth/auth-guard";
import { MainLayout } from "@/layouts/MainLayout";
import type { ReactNode } from "react";

export default function MainGroupLayout({ children }: { children: ReactNode }) {
	return (
		<AuthGuard>
			<MainLayout>{children}</MainLayout>
		</AuthGuard>
	);
}
