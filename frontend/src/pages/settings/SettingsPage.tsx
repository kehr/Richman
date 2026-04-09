import { PageContainer } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
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
	const { t } = useTranslation("settings");
	const rawTab = searchParams.get("tab");
	const activeKey: TabKey = isTabKey(rawTab) ? rawTab : "account";

	const items = useMemo<SettingsTabItem[]>(
		() => [
			{ key: "account", label: t("tabs.account"), content: <AccountTab /> },
			{ key: "ai", label: t("tabs.ai"), content: <AITab /> },
			{ key: "channels", label: t("tabs.channels"), content: <ChannelsTab /> },
			{ key: "preferences", label: t("tabs.preferences"), content: <PreferencesTab /> },
			{ key: "subscription", label: t("tabs.subscription"), content: <SubscriptionTab /> },
		],
		[t],
	);

	const handleChange = (key: string) => {
		const next = new URLSearchParams(searchParams);
		next.set("tab", key);
		setSearchParams(next, { replace: true });
	};

	return (
		<PageContainer title={t("title")} data-testid="settings-page">
			<SettingsTabsLayout items={items} activeKey={activeKey} onChange={handleChange} />
		</PageContainer>
	);
}
