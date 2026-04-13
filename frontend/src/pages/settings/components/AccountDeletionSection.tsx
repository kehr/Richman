import { getToken } from "@/domain/auth/storage";
import { API_V1_BASE, ApiError } from "@/domain/http/client";
import { useLogout } from "@/features/auth";
import { App, Button, Flex, Form, Input, LockOutlined, Modal, Typography } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";

// deleteAccount sends DELETE /api/v1/auth/account and expects 204 No Content.
// We cannot use requestV1() here because handleResponse() calls response.json()
// which fails on 204 empty body — so we fetch directly.
async function deleteAccount(password: string): Promise<void> {
	const token = getToken();
	const response = await fetch(`${API_V1_BASE}/auth/account`, {
		method: "DELETE",
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: `Bearer ${token}` } : {}),
		},
		body: JSON.stringify({ password }),
	});
	if (!response.ok) {
		const body = await response.json().catch(() => ({}));
		throw new ApiError(
			response.status,
			body?.error?.code || "UNKNOWN",
			body?.error?.message || response.statusText,
		);
	}
}

interface DeleteFormValues {
	password: string;
}

// AccountDeletionSection renders a danger-zone block that lets users permanently
// delete their account after confirming their password.
export function AccountDeletionSection() {
	const { t } = useTranslation("settings");
	const { message } = App.useApp();
	const logout = useLogout();

	const [open, setOpen] = useState(false);
	const [loading, setLoading] = useState(false);
	const [form] = Form.useForm<DeleteFormValues>();

	const handleOpen = () => {
		form.resetFields();
		setOpen(true);
	};

	const handleCancel = () => {
		setOpen(false);
	};

	const handleConfirm = async () => {
		try {
			const values = await form.validateFields();
			setLoading(true);
			await deleteAccount(values.password);
			message.success(t("deleteAccount.success"));
			// Give the success message a moment to display before logging out.
			setTimeout(() => {
				logout();
			}, 800);
		} catch (err) {
			// antd validation rejection — ignore (per-field messages shown inline).
			if (err && typeof err === "object" && "errorFields" in err) return;
			message.error(t("deleteAccount.error"));
		} finally {
			setLoading(false);
		}
	};

	return (
		<>
			<Flex
				vertical
				gap={8}
				style={{
					padding: "16px",
					borderRadius: 8,
					border: "1px solid #ffccc7",
					background: "#fff2f0",
				}}
				data-testid="account-deletion-section"
			>
				<Flex align="center" gap={8}>
					<LockOutlined style={{ color: "#cf1322", fontSize: 16 }} />
					<Typography.Text strong style={{ color: "#cf1322" }}>
						{t("deleteAccount.title")}
					</Typography.Text>
				</Flex>
				<Typography.Text type="secondary" style={{ fontSize: 13 }}>
					{t("deleteAccount.description")}
				</Typography.Text>
				<Button
					danger
					size="small"
					onClick={handleOpen}
					style={{ alignSelf: "flex-start", marginTop: 4 }}
					data-testid="account-deletion-open-btn"
				>
					{t("deleteAccount.openButton")}
				</Button>
			</Flex>

			<Modal
				title={t("deleteAccount.modal.title")}
				open={open}
				onCancel={handleCancel}
				footer={null}
				destroyOnClose
				data-testid="account-deletion-modal"
			>
				<Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
					{t("deleteAccount.modal.warning")}
				</Typography.Paragraph>

				<Form<DeleteFormValues> form={form} layout="vertical" onFinish={handleConfirm}>
					<Form.Item
						name="password"
						label={t("deleteAccount.modal.passwordLabel")}
						rules={[{ required: true, message: t("deleteAccount.modal.passwordRequired") }]}
					>
						<Input.Password
							placeholder={t("deleteAccount.modal.passwordPlaceholder")}
							size="large"
						/>
					</Form.Item>

					<Flex gap={8} justify="flex-end">
						<Button onClick={handleCancel}>{t("deleteAccount.modal.cancel")}</Button>
						<Button
							danger
							type="primary"
							loading={loading}
							htmlType="submit"
							data-testid="account-deletion-confirm-btn"
						>
							{t("deleteAccount.modal.confirm")}
						</Button>
					</Flex>
				</Form>
			</Modal>
		</>
	);
}
