import { Alert, Button, Form, Input } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
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
	const { t } = useTranslation("auth");
	const { mutate, isPending, error } = useLogin({ redirectTo });

	const validationRules = useMemo(
		() => ({
			email: [
				{ required: true, message: t("validation.emailRequired") },
				{ type: "email" as const, message: t("validation.emailInvalid") },
			],
			password: [{ required: true, message: t("validation.passwordRequired") }],
		}),
		[t],
	);

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
					{t("login.title")}
				</h2>
				<p
					style={{
						margin: "8px 0 0",
						fontSize: 14,
						color: "#6b6b70",
						lineHeight: 1.6,
					}}
				>
					{t("login.subtitle")}
				</p>
			</div>

			{error && (
				<Alert message={error.message} type="error" showIcon style={{ marginBottom: 16 }} />
			)}

			<Form layout="vertical" onFinish={handleSubmit} autoComplete="off">
				<Form.Item name="email" label={t("field.email")} rules={validationRules.email}>
					<Input placeholder={t("field.emailPlaceholder")} size="large" />
				</Form.Item>

				<Form.Item name="password" label={t("field.password")} rules={validationRules.password}>
					<Input.Password placeholder={t("field.passwordPlaceholder")} size="large" />
				</Form.Item>

				<Form.Item style={{ marginBottom: 16 }}>
					<Button type="primary" htmlType="submit" block size="large" loading={isPending}>
						{t("login.submit")}
					</Button>
				</Form.Item>

				<div style={{ fontSize: 14, color: "#6b6b70" }}>
					{t("login.noAccount")} <Link to="/register">{t("login.registerLink")}</Link>
				</div>
			</Form>
		</div>
	);
}
