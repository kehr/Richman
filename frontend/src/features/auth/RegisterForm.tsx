import { Alert, Button, Form, Input, Typography } from "@/ui-kit/eat";
import { Link } from "react-router";
import type { RegisterInput } from "./api";
import { useRegister } from "./useAuth";

const { Title } = Typography;

export function RegisterForm() {
	const { mutate, isPending, error } = useRegister();

	const handleSubmit = (values: RegisterInput) => {
		mutate(values);
	};

	return (
		<div style={{ width: 360 }}>
			<Title level={3} style={{ textAlign: "center", marginBottom: 32 }}>
				Create Account
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

				<Form.Item>
					<Button type="primary" htmlType="submit" block size="large" loading={isPending}>
						Register
					</Button>
				</Form.Item>

				<div style={{ textAlign: "center" }}>
					Already have an account? <Link to="/login">Sign In</Link>
				</div>
			</Form>
		</div>
	);
}
