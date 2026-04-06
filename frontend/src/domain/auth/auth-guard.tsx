import { Spin } from "@/ui-kit/eat";
import { type ReactNode, useEffect } from "react";
import { useNavigate } from "react-router";
import { getToken } from "./storage";
import { useCurrentUser } from "./use-current-user";

interface AuthGuardProps {
	children: ReactNode;
}

export function AuthGuard({ children }: AuthGuardProps) {
	const navigate = useNavigate();
	const token = getToken();
	const { isLoading, isError } = useCurrentUser();

	useEffect(() => {
		if (!token || isError) {
			navigate("/login", { replace: true });
		}
	}, [token, isError, navigate]);

	if (!token) {
		return null;
	}

	if (isLoading) {
		return (
			<div
				style={{
					display: "flex",
					justifyContent: "center",
					alignItems: "center",
					height: "100vh",
				}}
			>
				<Spin size="large" />
			</div>
		);
	}

	return <>{children}</>;
}
