import { PageContainer, Typography } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useSearchParams } from "react-router";
import { type SettingsTabItem, SettingsTabsLayout } from "./components/SettingsTabsLayout";
import { AccountTab } from "./tabs/AccountTab";

const TAB_KEYS = ["account", "channels", "preferences", "subscription"] as const;
type TabKey = (typeof TAB_KEYS)[number];

function isTabKey(value: string | null): value is TabKey {
	return value != null && (TAB_KEYS as readonly string[]).includes(value);
}

// Placeholder used by tabs that have not been implemented yet. Replaced by
// dedicated tab components in subsequent commits within Step 18.
function ComingSoon({ label }: { label: string }) {
	return <Typography.Text type="secondary">{label} 即将上线</Typography.Text>;
}

// SettingsPage is the composition root for the four-tab settings UI in
// PRD §6. Tab selection is mirrored to the URL via the ?tab= query param so
// external links can deep-link into a specific tab.
export default function SettingsPage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const rawTab = searchParams.get("tab");
	const activeKey: TabKey = isTabKey(rawTab) ? rawTab : "account";

	const items = useMemo<SettingsTabItem[]>(
		() => [
			{ key: "account", label: "账户", content: <AccountTab /> },
			{ key: "channels", label: "推送渠道", content: <ComingSoon label="推送渠道" /> },
			{ key: "preferences", label: "偏好", content: <ComingSoon label="偏好" /> },
			{ key: "subscription", label: "订阅与额度", content: <ComingSoon label="订阅与额度" /> },
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
