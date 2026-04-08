import { Alert, Button, Form, Input, Typography } from "@/ui-kit/eat";
import { Link } from "react-router";
import type { LoginInput } from "./api";
import { useLogin } from "./useAuth";

const { Title } = Typography;

interface LoginFormProps {
	// Optional override for the post-login redirect target. Pages parse
	// ?returnTo= from the URL, validate it, and pass the cleaned value here.
	redirectTo?: string;
}

export function LoginForm({ redirectTo }: LoginFormProps = {}) {
	const { mutate, isPending, error } = useLogin({ redirectTo });

	const handleSubmit = (values: LoginInput) => {
		mutate(values);
	};

	return (
		<div style={{ width: 360 }}>
			<Title level={3} style={{ textAlign: "center", marginBottom: 32 }}>
				Sign In
			</Title>

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

				<Form.Item>
					<Button type="primary" htmlType="submit" block size="large" loading={isPending}>
						Sign In
					</Button>
				</Form.Item>

				<div style={{ textAlign: "center" }}>
					Don&apos;t have an account? <Link to="/register">Register</Link>
				</div>
			</Form>
		</div>
	);
}
