import { LoginForm } from "@/features/auth";

export default function LoginPage() {
	return (
		<div
			style={{
				display: "flex",
				justifyContent: "center",
				alignItems: "center",
				minHeight: "100vh",
			}}
		>
			<LoginForm />
		</div>
	);
}
