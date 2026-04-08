import { Alert, Button, Form, Input } from "@/ui-kit/eat";
import { Link } from "react-router";
import type { RegisterInput } from "./api";
import { useRegister } from "./useAuth";

interface RegisterFormProps {
	// Optional override for the post-register redirect target, mirroring
	// LoginForm so deep links survive the register pivot.
	redirectTo?: string;
}

// RegisterForm mirrors LoginForm layout: width comes from the parent
// (AuthSplitLayout's form-wrapper), title is left-aligned display style.
export function RegisterForm({ redirectTo }: RegisterFormProps) {
	const { mutate, isPending, error } = useRegister({ redirectTo });

	const handleSubmit = (values: RegisterInput) => {
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
					Create Account
				</h2>
				<p
					style={{
						margin: "8px 0 0",
						fontSize: 14,
						color: "#6b6b70",
						lineHeight: 1.6,
					}}
				>
					只需邀请码即可加入，立即开始你的第一张决策卡。
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
					rules={[
						{ required: true, message: "Please enter your password" },
						{ min: 8, message: "Password must be at least 8 characters" },
					]}
				>
					<Input.Password placeholder="Password" size="large" />
				</Form.Item>

				<Form.Item
					name="inviteCode"
					label="Invite Code"
					rules={[{ required: true, message: "Please enter your invite code" }]}
				>
					<Input placeholder="Invite Code" size="large" />
				</Form.Item>

				<Form.Item style={{ marginBottom: 16 }}>
					<Button type="primary" htmlType="submit" block size="large" loading={isPending}>
						Register
					</Button>
				</Form.Item>

				<div style={{ fontSize: 14, color: "#6b6b70" }}>
					Already have an account? <Link to="/login">Sign In</Link>
				</div>
			</Form>
		</div>
	);
}
