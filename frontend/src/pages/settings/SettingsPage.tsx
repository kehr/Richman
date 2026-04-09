import { PageContainer } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useSearchParams } from "react-router";
import { type SettingsTabItem, SettingsTabsLayout } from "./components/SettingsTabsLayout";
import { AITab } from "./tabs/AITab";
import { AccountTab } from "./tabs/AccountTab";
import { ChannelsTab } from "./tabs/ChannelsTab";
import { PreferencesTab } from "./tabs/PreferencesTab";
import { SubscriptionTab } from "./tabs/SubscriptionTab";

const TAB_KEYS = ["account", "ai", "channels", "preferences", "subscription"] as const;
type TabKey = (typeof TAB_KEYS)[number];

function isTabKey(value: string | null): value is TabKey {
	return value != null && (TAB_KEYS as readonly string[]).includes(value);
}

// SettingsPage is the composition root for the five-tab settings UI. Tab
// selection is mirrored to the URL via the ?tab= query param so external
// links can deep-link into a specific tab. The "ai" tab was added by the
// LLM degraded contract work and hosts the LLMSection feature.
export default function SettingsPage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const rawTab = searchParams.get("tab");
	const activeKey: TabKey = isTabKey(rawTab) ? rawTab : "account";

	const items = useMemo<SettingsTabItem[]>(
		() => [
			{ key: "account", label: "账户", content: <AccountTab /> },
			{ key: "ai", label: "AI 解读", content: <AITab /> },
			{ key: "channels", label: "推送渠道", content: <ChannelsTab /> },
			{ key: "preferences", label: "偏好", content: <PreferencesTab /> },
			{ key: "subscription", label: "订阅与额度", content: <SubscriptionTab /> },
		],
		[],
	);

	const handleChange = (key: string) => {
		const next = new URLSearchParams(searchParams);
		next.set("tab", key);
		setSearchParams(next, { replace: true });
	};

	return (
		<PageContainer title="设置" data-testid="settings-page">
			<SettingsTabsLayout items={items} activeKey={activeKey} onChange={handleChange} />
		</PageContainer>
	);
}
