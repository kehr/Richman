import { Button, Drawer, Flex, Form, Input, Radio, Typography, message } from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type {
	ChannelType,
	CreateChannelInput,
	EmailChannelConfig,
	FeishuChannelConfig,
	WechatChannelConfig,
} from "../types";
import { useCreateChannel } from "../use-channels";

interface AddChannelDrawerProps {
	open: boolean;
	onClose: () => void;
}

interface EmailFormValues {
	to: string;
}

interface FeishuFormValues {
	webhookUrl: string;
}

interface WechatFormValues {
	openId: string;
	templateId: string;
}

// AddChannelDrawer guides the user through a two-step flow: pick a channel
// type, then fill in the per-type configuration form. The form fields mirror
// the backend adapter config structs verbatim (see types.ts for citations).
export function AddChannelDrawer({ open, onClose }: AddChannelDrawerProps) {
	const { t } = useTranslation("settings");
	const [channelType, setChannelType] = useState<ChannelType>("email");
	const [emailForm] = Form.useForm<EmailFormValues>();
	const [feishuForm] = Form.useForm<FeishuFormValues>();
	const [wechatForm] = Form.useForm<WechatFormValues>();
	const createMutation = useCreateChannel();

	const handleClose = () => {
		emailForm.resetFields();
		feishuForm.resetFields();
		wechatForm.resetFields();
		setChannelType("email");
		onClose();
	};

	// Memoize validation rules to stay reactive to locale changes.
	const emailRules = useMemo(
		() => [
			{ required: true, message: t("channels.emailForm.validation.required") },
			{ type: "email" as const, message: t("channels.emailForm.validation.invalid") },
		],
		[t],
	);

	const feishuRules = useMemo(
		() => [
			{ required: true, message: t("channels.feishuForm.validation.required") },
			{ type: "url" as const, message: t("channels.feishuForm.validation.invalid") },
		],
		[t],
	);

	const wechatOpenIdRules = useMemo(
		() => [{ required: true, message: t("channels.wechatForm.validation.openIdRequired") }],
		[t],
	);

	const wechatTemplateIdRules = useMemo(
		() => [{ required: true, message: t("channels.wechatForm.validation.templateIdRequired") }],
		[t],
	);

	const buildPayload = async (): Promise<CreateChannelInput | null> => {
		switch (channelType) {
			case "email": {
				const values = await emailForm.validateFields();
				const config: EmailChannelConfig = { to: values.to };
				return { channelType: "email", config };
			}
			case "feishu": {
				const values = await feishuForm.validateFields();
				const config: FeishuChannelConfig = { webhookUrl: values.webhookUrl };
				return { channelType: "feishu", config };
			}
			case "wechat": {
				const values = await wechatForm.validateFields();
				const config: WechatChannelConfig = {
					openId: values.openId,
					templateId: values.templateId,
				};
				return { channelType: "wechat", config };
			}
			default:
				return null;
		}
	};

	const handleSubmit = async () => {
		try {
			const payload = await buildPayload();
			if (!payload) return;
			await createMutation.mutateAsync(payload);
			message.success(t("channels.drawer.saveSuccess"));
			handleClose();
		} catch (err) {
			// validateFields rejects with an error list; treat any other thrown
			// value as a network error.
			if (err && typeof err === "object" && "errorFields" in err) {
				return;
			}
			message.error(t("channels.drawer.saveError"));
		}
	};

	return (
		<Drawer
			title={t("channels.drawer.title")}
			open={open}
			onClose={handleClose}
			placement="right"
			width={480}
			data-testid="add-channel-drawer"
			footer={
				<Flex justify="flex-end" gap={8}>
					<Button onClick={handleClose}>{t("action.cancel", { ns: "common" })}</Button>
					<Button
						type="primary"
						loading={createMutation.isPending}
						onClick={handleSubmit}
						data-testid="add-channel-save"
					>
						{t("action.save", { ns: "common" })}
					</Button>
				</Flex>
			}
		>
			<Form.Item label={t("channels.drawer.channelType")}>
				<Radio.Group
					value={channelType}
					onChange={(e) => setChannelType(e.target.value as ChannelType)}
					data-testid="channel-type-picker"
				>
					<Radio.Button value="email">{t("channels.drawer.email")}</Radio.Button>
					<Radio.Button value="feishu">{t("channels.drawer.feishu")}</Radio.Button>
					<Radio.Button value="wechat">{t("channels.drawer.wechat")}</Radio.Button>
				</Radio.Group>
			</Form.Item>

			{channelType === "email" && (
				<Form<EmailFormValues> form={emailForm} layout="vertical" data-testid="channel-form-email">
					<Form.Item label={t("channels.emailForm.recipient")} name="to" rules={emailRules}>
						<Input placeholder={t("channels.emailForm.recipientPlaceholder")} />
					</Form.Item>
					<Typography.Text type="secondary">{t("channels.emailForm.defaultHint")}</Typography.Text>
				</Form>
			)}

			{channelType === "feishu" && (
				<Form<FeishuFormValues>
					form={feishuForm}
					layout="vertical"
					data-testid="channel-form-feishu"
				>
					<Form.Item
						label={t("channels.feishuForm.webhookUrl")}
						name="webhookUrl"
						rules={feishuRules}
					>
						<Input placeholder={t("channels.feishuForm.webhookPlaceholder")} />
					</Form.Item>
				</Form>
			)}

			{channelType === "wechat" && (
				<Form<WechatFormValues>
					form={wechatForm}
					layout="vertical"
					data-testid="channel-form-wechat"
				>
					<Form.Item
						label={t("channels.wechatForm.openId")}
						name="openId"
						rules={wechatOpenIdRules}
					>
						<Input placeholder={t("channels.wechatForm.openIdPlaceholder")} />
					</Form.Item>
					<Form.Item
						label={t("channels.wechatForm.templateId")}
						name="templateId"
						rules={wechatTemplateIdRules}
					>
						<Input placeholder={t("channels.wechatForm.templateIdPlaceholder")} />
					</Form.Item>
				</Form>
			)}
		</Drawer>
	);
}
