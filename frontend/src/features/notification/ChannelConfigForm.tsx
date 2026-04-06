"use client";

import { Button, Form, Input, Select, Space, Switch, message } from "@/ui-kit/eat";
import type { NotificationChannelDto } from "./api";
import { useCreateChannel, useUpdateChannel } from "./useNotification";

const CHANNEL_TYPES = [
	{ value: "webhook", label: "Webhook" },
	{ value: "email", label: "Email" },
	{ value: "wechat", label: "WeChat" },
];

interface ChannelConfigFormProps {
	initialValues?: NotificationChannelDto;
	onSuccess?: () => void;
}

export function ChannelConfigForm({ initialValues, onSuccess }: ChannelConfigFormProps) {
	const [form] = Form.useForm();
	const createMutation = useCreateChannel();
	const updateMutation = useUpdateChannel();

	const isEdit = !!initialValues;

	const handleSubmit = async (values: Record<string, unknown>) => {
		try {
			const config: Record<string, unknown> = {};
			if (values.url) config.url = values.url;
			if (values.email) config.email = values.email;
			if (values.secret) config.secret = values.secret;

			if (isEdit) {
				await updateMutation.mutateAsync({
					id: initialValues.channelId,
					data: {
						config,
						enabled: values.enabled as boolean,
					},
				});
				message.success("Channel updated");
			} else {
				await createMutation.mutateAsync({
					channelType: values.channelType as string,
					config,
				});
				message.success("Channel created");
			}
			onSuccess?.();
		} catch {
			message.error("Operation failed");
		}
	};

	return (
		<Form
			form={form}
			layout="vertical"
			initialValues={
				isEdit
					? {
							channelType: initialValues.channelType,
							enabled: initialValues.enabled,
							url: initialValues.config.url,
							email: initialValues.config.email,
							secret: initialValues.config.secret,
						}
					: { enabled: true }
			}
			onFinish={handleSubmit}
		>
			<Form.Item
				label="Channel Type"
				name="channelType"
				rules={[{ required: true, message: "Please select channel type" }]}
			>
				<Select options={CHANNEL_TYPES} disabled={isEdit} />
			</Form.Item>

			<Form.Item label="Webhook URL" name="url">
				<Input placeholder="https://example.com/webhook" />
			</Form.Item>

			<Form.Item label="Email" name="email">
				<Input placeholder="user@example.com" />
			</Form.Item>

			<Form.Item label="Secret" name="secret">
				<Input.Password placeholder="Optional secret key" />
			</Form.Item>

			{isEdit && (
				<Form.Item label="Enabled" name="enabled" valuePropName="checked">
					<Switch />
				</Form.Item>
			)}

			<Space>
				<Button
					type="primary"
					htmlType="submit"
					loading={createMutation.isPending || updateMutation.isPending}
				>
					{isEdit ? "Update" : "Create"}
				</Button>
			</Space>
		</Form>
	);
}
