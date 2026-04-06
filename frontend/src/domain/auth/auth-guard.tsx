"use client";

import { Spin } from "@/ui-kit/eat";
import { useRouter } from "next/navigation";
import { type ReactNode, useEffect } from "react";
import { getToken } from "./storage";
import { useCurrentUser } from "./use-current-user";

interface AuthGuardProps {
	children: ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
	const router = useRouter();
	const token = getToken();
	const { isLoading, isError } = useCurrentUser();

	useEffect(() => {
		if (!token || isError) {
			router.replace("/login");
		}
	}, [token, isError, router]);

	if (!token) {
		return null;
	}

	if (isLoading) {
		return (
			<div
				style={{ display: "flex", justifyContent: "center", alignItems: "center", height: "100vh" }}
			>
				<Spin size="large" />
			</div>
		);
	}

	return <>{children}</>;
}
