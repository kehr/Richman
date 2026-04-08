import { Button, Drawer, Flex, Form, Input, Radio, Typography, message } from "@/ui-kit/eat";
import { useState } from "react";
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
			message.success("渠道已添加");
			handleClose();
		} catch (err) {
			// validateFields rejects with an error list; treat any other thrown
			// value as a network error.
			if (err && typeof err === "object" && "errorFields" in err) {
				return;
			}
			message.error("添加渠道失败");
		}
	};

	return (
		<Drawer
			title="添加推送渠道"
			open={open}
			onClose={handleClose}
			placement="right"
			width={480}
			data-testid="add-channel-drawer"
			footer={
				<Flex justify="flex-end" gap={8}>
					<Button onClick={handleClose}>取消</Button>
					<Button
						type="primary"
						loading={createMutation.isPending}
						onClick={handleSubmit}
						data-testid="add-channel-save"
					>
						保存
					</Button>
				</Flex>
			}
		>
			<Form.Item label="渠道类型">
				<Radio.Group
					value={channelType}
					onChange={(e) => setChannelType(e.target.value as ChannelType)}
					data-testid="channel-type-picker"
				>
					<Radio.Button value="email">邮件</Radio.Button>
					<Radio.Button value="feishu">飞书机器人</Radio.Button>
					<Radio.Button value="wechat">微信公众号</Radio.Button>
				</Radio.Group>
			</Form.Item>

			{channelType === "email" && (
				<Form<EmailFormValues> form={emailForm} layout="vertical" data-testid="channel-form-email">
					<Form.Item
						label="收件邮箱"
						name="to"
						rules={[
							{ required: true, message: "请输入收件邮箱" },
							{ type: "email", message: "邮箱格式不正确" },
						]}
					>
						<Input placeholder="user@example.com" />
					</Form.Item>
					<Typography.Text type="secondary">留空将默认发送到当前账户邮箱。</Typography.Text>
				</Form>
			)}

			{channelType === "feishu" && (
				<Form<FeishuFormValues>
					form={feishuForm}
					layout="vertical"
					data-testid="channel-form-feishu"
				>
					<Form.Item
						label="Webhook URL"
						name="webhookUrl"
						rules={[
							{ required: true, message: "请输入飞书 Webhook URL" },
							{ type: "url", message: "URL 格式不正确" },
						]}
					>
						<Input placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." />
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
						label="OpenID"
						name="openId"
						rules={[{ required: true, message: "请输入用户 OpenID" }]}
					>
						<Input placeholder="o6_bmjrPTlm6_2sgVt7hMZOPfL2M" />
					</Form.Item>
					<Form.Item
						label="模板 ID"
						name="templateId"
						rules={[{ required: true, message: "请输入模板消息 ID" }]}
					>
						<Input placeholder="模板消息 template_id" />
					</Form.Item>
				</Form>
			)}
		</Drawer>
	);
}
