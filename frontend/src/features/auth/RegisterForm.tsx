import { Alert, Button, Form, Input } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
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
	const { t } = useTranslation("auth");
	const { mutate, isPending, error } = useRegister({ redirectTo });

	const validationRules = useMemo(
		() => ({
			email: [
				{ required: true, message: t("validation.emailRequired") },
				{ type: "email" as const, message: t("validation.emailInvalid") },
			],
			password: [
				{ required: true, message: t("validation.passwordRequired") },
				{ min: 8, message: t("validation.passwordMinLength") },
			],
			inviteCode: [{ required: true, message: t("validation.inviteCodeRequired") }],
		}),
		[t],
	);

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
					{t("register.title")}
				</h2>
				<p
					style={{
						margin: "8px 0 0",
						fontSize: 14,
						color: "#6b6b70",
						lineHeight: 1.6,
					}}
				>
					{t("register.subtitle")}
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

				<Form.Item
					name="inviteCode"
					label={t("field.inviteCode")}
					rules={validationRules.inviteCode}
				>
					<Input placeholder={t("field.inviteCodePlaceholder")} size="large" />
				</Form.Item>

				<Form.Item style={{ marginBottom: 16 }}>
					<Button type="primary" htmlType="submit" block size="large" loading={isPending}>
						{t("register.submit")}
					</Button>
				</Form.Item>

				<div style={{ fontSize: 14, color: "#6b6b70" }}>
					{t("register.hasAccount")} <Link to="/login">{t("register.loginLink")}</Link>
				</div>
			</Form>
		</div>
	);
}
