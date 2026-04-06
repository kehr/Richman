import { RegisterForm } from "@/features/auth";

export default function RegisterPage() {
	return (
		<div
			style={{
				display: "flex",
				justifyContent: "center",
				alignItems: "center",
				minHeight: "100vh",
			}}
		>
			<RegisterForm />
		</div>
	);
}
