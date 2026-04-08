import { AddChannelDrawer, ChannelList, useChannels } from "@/features/notification-channels";
import { Button, Divider, Flex, PlusOutlined, Typography } from "@/ui-kit/eat";
import { useMemo, useState } from "react";

// ChannelsTab is the PRD §6.3 channel management view: header counter,
// channel list (with toggle / test / delete actions), an "add" button that
// opens the drawer, and a footer pointer to the help anchor explaining
// push windows.
export function ChannelsTab() {
	const channelsQuery = useChannels();
	const [drawerOpen, setDrawerOpen] = useState(false);

	const channels = channelsQuery.data ?? [];
	const enabledCount = useMemo(() => channels.filter((c) => c.enabled).length, [channels]);

	return (
		<Flex vertical gap={16} data-testid="channels-tab">
			<Flex align="center" justify="space-between">
				<Typography.Text type="secondary" data-testid="channels-counter">
					当前已启用 {enabledCount} 个渠道
				</Typography.Text>
				<Button
					type="primary"
					icon={<PlusOutlined />}
					onClick={() => setDrawerOpen(true)}
					data-testid="channels-add-button"
				>
					添加渠道
				</Button>
			</Flex>

			<ChannelList channels={channels} loading={channelsQuery.isLoading} />

			<Divider style={{ margin: "8px 0" }} />

			<Typography.Text type="secondary" style={{ fontSize: 12 }}>
				推送时段：北京时间 08:30 / 15:30 / 次日 06:00，根据持仓自动筛选。
				<a href="/help#push" style={{ marginLeft: 8 }}>
					了解更多
				</a>
			</Typography.Text>

			<AddChannelDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />
		</Flex>
	);
}
