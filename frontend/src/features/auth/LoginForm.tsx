import { Alert, Button, Form, Input } from "@/ui-kit/eat";
import { Link } from "react-router";
import type { LoginInput } from "./api";
import { useLogin } from "./useAuth";

interface LoginFormProps {
	// Optional override for the post-login redirect target. Pages parse
	// ?returnTo= from the URL, validate it, and pass the cleaned value here.
	redirectTo?: string;
}

// LoginForm fills whatever width its parent provides. AuthSplitLayout is the
// single source of truth for form width and horizontal anchoring; see
// .auth-split-layout__form-wrapper.
export function LoginForm({ redirectTo }: LoginFormProps) {
	const { mutate, isPending, error } = useLogin({ redirectTo });

	const handleSubmit = (values: LoginInput) => {
		mutate(values);
	};

	return (
		<div style={{ width: "100%" }}>
			<div style={{ marginBottom: 28 }}>
				<h2
					style={{
						margin: 0,
						fontSize: 30,
						fontWeight: 600,
						letterSpacing: "-0.01em",
						color: "#0b0b0d",
						lineHeight: 1.2,
					}}
				>
					Sign In
				</h2>
				<p
					style={{
						margin: "8px 0 0",
						fontSize: 14,
						color: "#6b6b70",
						lineHeight: 1.6,
					}}
				>
					输入邮箱和密码继续你的决策。
				</p>
			</div>

			{error && (
				<Alert message={error.message} type="error" showIcon style={{ marginBottom: 16 }} />
			)}

			<Form layout="vertical" onFinish={handleSubmit} autoComplete="off">
				<Form.Item
					name="email"
					label="Email"
					rules={[
						{ required: true, message: "Please enter your email" },
						{ type: "email", message: "Invalid email format" },
					]}
				>
					<Input placeholder="Email" size="large" />
				</Form.Item>

				<Form.Item
					name="password"
					label="Password"
					rules={[{ required: true, message: "Please enter your password" }]}
				>
					<Input.Password placeholder="Password" size="large" />
				</Form.Item>

				<Form.Item style={{ marginBottom: 16 }}>
					<Button type="primary" htmlType="submit" block size="large" loading={isPending}>
						Sign In
					</Button>
				</Form.Item>

				<div style={{ fontSize: 14, color: "#6b6b70" }}>
					Don&apos;t have an account? <Link to="/register">Register</Link>
				</div>
			</Form>
		</div>
	);
}
