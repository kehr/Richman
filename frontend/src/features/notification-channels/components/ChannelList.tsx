import {
	BellOutlined,
	Button,
	DeleteOutlined,
	Empty,
	Flex,
	List,
	Popconfirm,
	Space,
	Switch,
	Tag,
	Typography,
	message,
} from "@/ui-kit/eat";
import type { ChannelDto, ChannelType } from "../types";
import { useDeleteChannel, useUpdateChannel } from "../use-channels";
import { ChannelTestButton } from "./ChannelTestButton";

interface ChannelListProps {
	channels: ChannelDto[];
	loading?: boolean;
}

const TYPE_LABEL: Record<ChannelType, string> = {
	email: "邮件",
	feishu: "飞书机器人",
	wechat: "微信公众号",
};

// summarizeConfig produces a one-line description of the channel's config
// for the row body. Because the backend stores config as a free-form JSON
// object we narrow it locally before reading the per-type fields.
function summarizeConfig(channel: ChannelDto): string {
	const cfg = (channel.config ?? {}) as Record<string, unknown>;
	switch (channel.channelType) {
		case "email":
			return typeof cfg.to === "string" && cfg.to.length > 0 ? cfg.to : "默认收件人";
		case "feishu":
			return typeof cfg.webhookUrl === "string" ? cfg.webhookUrl : "未配置 webhook";
		case "wechat": {
			const openId = typeof cfg.openId === "string" ? cfg.openId : "";
			return openId.length > 0 ? `OpenID ${openId}` : "未配置 OpenID";
		}
		default:
			return "";
	}
}

export function ChannelList({ channels, loading }: ChannelListProps) {
	const updateMutation = useUpdateChannel();
	const deleteMutation = useDeleteChannel();

	const handleToggle = async (channel: ChannelDto, enabled: boolean) => {
		try {
			await updateMutation.mutateAsync({
				channelId: channel.channelId,
				input: { enabled },
			});
			message.success(enabled ? "渠道已启用" : "渠道已停用");
		} catch {
			message.error("更新渠道失败");
		}
	};

	const handleDelete = async (channel: ChannelDto) => {
		try {
			await deleteMutation.mutateAsync(channel.channelId);
			message.success("渠道已删除");
		} catch {
			message.error("删除渠道失败");
		}
	};

	if (!loading && channels.length === 0) {
		return (
			<Empty
				description="尚未配置任何推送渠道"
				data-testid="channel-list-empty"
				style={{ padding: "32px 0" }}
			/>
		);
	}

	return (
		<List
			data-testid="channel-list"
			loading={loading}
			itemLayout="horizontal"
			dataSource={channels}
			renderItem={(channel) => (
				<List.Item
					key={channel.channelId}
					actions={[
						<Switch
							key="enabled"
							checked={channel.enabled}
							onChange={(checked) => handleToggle(channel, checked)}
							data-testid={`channel-enable-${channel.channelId}`}
						/>,
						<ChannelTestButton key="test" />,
						<Popconfirm
							key="delete"
							title="删除渠道"
							description="确认删除此推送渠道？"
							okText="删除"
							cancelText="取消"
							onConfirm={() => handleDelete(channel)}
						>
							<Button
								size="small"
								danger
								type="text"
								icon={<DeleteOutlined />}
								data-testid={`channel-delete-${channel.channelId}`}
							>
								删除
							</Button>
						</Popconfirm>,
					]}
				>
					<List.Item.Meta
						avatar={<BellOutlined style={{ fontSize: 20 }} />}
						title={
							<Space>
								<Typography.Text strong>{TYPE_LABEL[channel.channelType]}</Typography.Text>
								{!channel.enabled && <Tag color="default">已停用</Tag>}
							</Space>
						}
						description={
							<Flex vertical gap={2}>
								<Typography.Text type="secondary">{summarizeConfig(channel)}</Typography.Text>
							</Flex>
						}
					/>
				</List.Item>
			)}
		/>
	);
}
