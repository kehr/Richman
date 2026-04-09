import { AddChannelDrawer, ChannelList, useChannels } from "@/features/notification-channels";
import { Alert, Button, Divider, Flex, PlusOutlined, Typography } from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

// ChannelsTab is the PRD §6.3 channel management view: header counter,
// channel list (with toggle / test / delete actions), an "add" button that
// opens the drawer, and a footer pointer to the help anchor explaining
// push windows.
export function ChannelsTab() {
	const { t } = useTranslation("settings");
	const channelsQuery = useChannels();
	const [drawerOpen, setDrawerOpen] = useState(false);

	const channels = channelsQuery.data ?? [];
	const enabledCount = useMemo(() => channels.filter((c) => c.enabled).length, [channels]);

	return (
		<Flex vertical gap={16} data-testid="channels-tab">
			<Flex align="center" justify="space-between">
				<Typography.Text type="secondary" data-testid="channels-counter">
					{t("channels.enabledCount", { count: enabledCount })}
				</Typography.Text>
				<Button
					type="primary"
					icon={<PlusOutlined />}
					onClick={() => setDrawerOpen(true)}
					data-testid="channels-add-button"
				>
					{t("channels.addButton")}
				</Button>
			</Flex>

			{channelsQuery.isError && (
				<Alert
					type="error"
					showIcon
					message={t("channels.loadError")}
					description={t("channels.loadErrorDesc")}
					data-testid="channels-load-error"
				/>
			)}

			<ChannelList channels={channels} loading={channelsQuery.isLoading} />

			<Divider style={{ margin: "8px 0" }} />

			<Typography.Text type="secondary" style={{ fontSize: 12 }}>
				{t("channels.pushSchedule")}
				<Link to="/help#push" style={{ marginLeft: 8 }}>
					{t("channels.learnMore")}
				</Link>
			</Typography.Text>

			<AddChannelDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />
		</Flex>
	);
}
