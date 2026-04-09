import { useChannels } from "@/features/notification-channels";
import { useHoldings } from "@/features/portfolio";
import {
	Badge,
	Button,
	Card,
	Col,
	Divider,
	Flex,
	Row,
	Tag,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const HOLDING_LIMIT = 5;
const CHANNEL_LIMIT = 3;

// SubscriptionTab covers PRD §6.5: a single "invite" tier badge, a quota
// usage grid (holdings, daily analyses, channels, model), and a disabled
// upgrade button. All numbers are derived from existing feature hooks; this
// tab does not call any subscription-specific API because the MVP only
// supports the invite tier.
export function SubscriptionTab() {
	const { t } = useTranslation("settings");
	const holdingsQuery = useHoldings();
	const channelsQuery = useChannels();

	const holdingCount = holdingsQuery.data?.length ?? 0;
	// Use the enabled-channel count so this tab stays consistent with the
	// ChannelsTab header. Disabled channels still occupy a row in the list
	// but should not count against the invite-tier quota.
	const channelCount = (channelsQuery.data ?? []).filter((c) => c.enabled).length;

	return (
		<Flex vertical gap={24} data-testid="subscription-tab">
			<Flex align="center" gap={12}>
				<Typography.Text type="secondary">{t("subscription.currentPlan")}</Typography.Text>
				<Badge status="success" />
				<Tag color="blue">invite</Tag>
				<Typography.Text type="secondary">{t("subscription.inviteExclusive")}</Typography.Text>
			</Flex>

			<Divider style={{ margin: 0 }} />

			<Row gutter={[16, 16]}>
				<Col xs={24} sm={12}>
					<Card size="small" data-testid="quota-holdings">
						<Typography.Text type="secondary">{t("subscription.quota.holdings")}</Typography.Text>
						<Typography.Title level={4} style={{ margin: "4px 0 0" }}>
							{holdingCount} / {HOLDING_LIMIT}
						</Typography.Title>
					</Card>
				</Col>
				<Col xs={24} sm={12}>
					<Card size="small" data-testid="quota-analysis">
						<Typography.Text type="secondary">
							{t("subscription.quota.dailyAnalysis")}
						</Typography.Text>
						<Typography.Title level={4} style={{ margin: "4px 0 0" }}>
							{t("subscription.quota.unlimited")}
						</Typography.Title>
					</Card>
				</Col>
				<Col xs={24} sm={12}>
					<Card size="small" data-testid="quota-channels">
						<Typography.Text type="secondary">{t("subscription.quota.channels")}</Typography.Text>
						<Typography.Title level={4} style={{ margin: "4px 0 0" }}>
							{channelCount} / {CHANNEL_LIMIT}
						</Typography.Title>
					</Card>
				</Col>
				<Col xs={24} sm={12}>
					<Card size="small" data-testid="quota-model">
						<Typography.Text type="secondary">{t("subscription.quota.model")}</Typography.Text>
						<Typography.Title level={4} style={{ margin: "4px 0 0" }}>
							Claude Sonnet 4.6
						</Typography.Title>
					</Card>
				</Col>
			</Row>

			<Tooltip title={t("subscription.upgradeTooltip")}>
				{/* antd disabled Button has pointer-events: none which suppresses the
				 * Tooltip hover — wrap in an inline-block span so the parent element
				 * owns the hover region. Same pattern as Step 16 AddHoldingDrawer I4. */}
				<span style={{ display: "inline-block", cursor: "not-allowed" }}>
					<Button disabled style={{ pointerEvents: "none" }} data-testid="subscription-upgrade">
						{t("subscription.upgradeButton")}
					</Button>
				</span>
			</Tooltip>
		</Flex>
	);
}
