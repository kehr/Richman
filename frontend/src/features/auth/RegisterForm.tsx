import { Alert, Button, Checkbox, Form, Input } from "@/ui-kit/eat";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import type { RegisterInput } from "./api";
import { useRegister } from "./useAuth";

interface RegisterFormProps {
	// Optional override for the post-register redirect target, mirroring
	// LoginForm so deep links survive the register pivot.
	redirectTo?: string;
	// Optional pre-filled invite code from the ?ref= URL parameter. When
	// provided the invite code field is pre-populated and the user can still
	// override it manually.
	refCode?: string | null;
}

// RegisterFormValues extends RegisterInput with the required disclaimer checkbox.
interface RegisterFormValues extends RegisterInput {
	disclaimerAccepted: boolean;
}

// RegisterForm mirrors LoginForm layout: width comes from the parent
// (AuthSplitLayout's form-wrapper), title is left-aligned display style.
export function RegisterForm({ redirectTo, refCode }: RegisterFormProps) {
	const { t } = useTranslation("auth");
	const { mutate, isPending, error } = useRegister({ redirectTo });
	const [form] = Form.useForm<RegisterFormValues>();

	// Auto-fill the invite code field when a ?ref= code is present in the URL.
	useEffect(() => {
		if (refCode) {
			form.setFieldValue("inviteCode", refCode);
		}
	}, [form, refCode]);

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
			disclaimerAccepted: [
				{
					validator: (_: unknown, value: boolean) =>
						value
							? Promise.resolve()
							: Promise.reject(new Error(t("validation.disclaimerRequired"))),
				},
			],
		}),
		[t],
	);

	const handleSubmit = (values: RegisterFormValues) => {
		// Strip the disclaimer field before sending to the API.
		const { disclaimerAccepted: _accepted, ...payload } = values;
		mutate(payload);
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

			<Form<RegisterFormValues>
				form={form}
				layout="vertical"
				onFinish={handleSubmit}
				autoComplete="off"
			>
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

				<Form.Item
					name="disclaimerAccepted"
					valuePropName="checked"
					rules={validationRules.disclaimerAccepted}
					style={{ marginBottom: 16 }}
				>
					<Checkbox>{t("register.disclaimerText")}</Checkbox>
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
