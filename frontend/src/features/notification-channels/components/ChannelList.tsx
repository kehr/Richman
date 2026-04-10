import {
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
import { useTranslation } from "react-i18next";
import type { ChannelDto, ChannelType } from "../types";
import { useDeleteChannel, useUpdateChannel } from "../use-channels";
import { ChannelIcon } from "./ChannelIcon";
import { ChannelTestButton } from "./ChannelTestButton";

interface ChannelListProps {
	channels: ChannelDto[];
	loading?: boolean;
}

export function ChannelList({ channels, loading }: ChannelListProps) {
	const { t } = useTranslation("settings");

	// summarizeConfig produces a one-line description of the channel's config
	// for the row body. Because the backend stores config as a free-form JSON
	// object we narrow it locally before reading the per-type fields.
	const summarizeConfig = (channel: ChannelDto): string => {
		const cfg = (channel.config ?? {}) as Record<string, unknown>;
		switch (channel.channelType) {
			case "email":
				return typeof cfg.to === "string" && cfg.to.length > 0
					? cfg.to
					: t("channels.list.defaultRecipient");
			case "feishu":
				return typeof cfg.webhookUrl === "string"
					? cfg.webhookUrl
					: t("channels.list.webhookNotConfigured");
			case "wechat": {
				const openId = typeof cfg.openId === "string" ? cfg.openId : "";
				return openId.length > 0 ? `OpenID ${openId}` : t("channels.list.openIdNotConfigured");
			}
			default:
				return "";
		}
	};
	const updateMutation = useUpdateChannel();
	const deleteMutation = useDeleteChannel();

	const typeLabel: Record<ChannelType, string> = {
		email: t("channels.list.typeLabel.email"),
		feishu: t("channels.list.typeLabel.feishu"),
		wechat: t("channels.list.typeLabel.wechat"),
	};

	const handleToggle = async (channel: ChannelDto, enabled: boolean) => {
		try {
			await updateMutation.mutateAsync({
				channelId: channel.channelId,
				input: { enabled },
			});
			message.success(
				enabled ? t("channels.list.enableSuccess") : t("channels.list.disableSuccess"),
			);
		} catch {
			message.error(t("channels.list.updateError"));
		}
	};

	const handleDelete = async (channel: ChannelDto) => {
		try {
			await deleteMutation.mutateAsync(channel.channelId);
			message.success(t("channels.list.deleteSuccess"));
		} catch {
			message.error(t("channels.list.deleteError"));
		}
	};

	if (!loading && channels.length === 0) {
		return (
			<Empty
				description={t("channels.list.empty")}
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
							title={t("channels.list.deleteConfirm.title")}
							description={t("channels.list.deleteConfirm.description")}
							okText={t("channels.list.deleteConfirm.ok")}
							cancelText={t("channels.list.deleteConfirm.cancel")}
							onConfirm={() => handleDelete(channel)}
						>
							<Button
								size="small"
								danger
								type="text"
								icon={<DeleteOutlined />}
								data-testid={`channel-delete-${channel.channelId}`}
							>
								{t("channels.list.deleteButton")}
							</Button>
						</Popconfirm>,
					]}
				>
					<List.Item.Meta
						avatar={<ChannelIcon channelType={channel.channelType} size={20} />}
						title={
							<Space>
								<Typography.Text strong>{typeLabel[channel.channelType]}</Typography.Text>
								{!channel.enabled && <Tag color="default">{t("channels.list.disabled")}</Tag>}
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
