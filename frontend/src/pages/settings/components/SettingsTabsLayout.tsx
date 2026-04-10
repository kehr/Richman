import { Card, Flex, Typography } from "@/ui-kit/eat";
import type { ReactNode } from "react";

export interface SettingsTabItem {
	key: string;
	label: string;
	icon?: ReactNode;
	content: ReactNode;
}

interface SettingsTabsLayoutProps {
	items: SettingsTabItem[];
	activeKey: string;
	onChange: (key: string) => void;
}

// SettingsTabsLayout renders the PRD §6.1 layout: a fixed 200px left tab
// rail with the active tab marked by a 3px black border on the left and a
// light grey background, plus the right-hand content area for whichever tab
// is currently selected. We deliberately do not use antd Tabs because the
// PRD calls for a custom rail style that does not match the default tab UI.
export function SettingsTabsLayout({ items, activeKey, onChange }: SettingsTabsLayoutProps) {
	const active = items.find((item) => item.key === activeKey) ?? items[0];

	return (
		<Flex gap={24} align="flex-start" data-testid="settings-tabs-layout">
			<div style={{ width: 200, flexShrink: 0 }} data-testid="settings-tabs-rail">
				{items.map((item) => {
					const isActive = item.key === active.key;
					return (
						<button
							key={item.key}
							type="button"
							onClick={() => onChange(item.key)}
							data-testid={`settings-tab-${item.key}`}
							aria-current={isActive ? "page" : undefined}
							style={{
								position: "relative",
								display: "flex",
								alignItems: "center",
								gap: 10,
								width: "100%",
								textAlign: "left",
								padding: "10px 16px 10px 19px",
								marginBottom: 4,
								border: "none",
								background: isActive ? "#f5f5f5" : "transparent",
								cursor: "pointer",
								fontSize: 14,
								fontWeight: isActive ? 600 : 400,
								color: isActive ? "#000" : "#5C5C5C",
							}}
						>
							{isActive && (
								<span
									style={{
										position: "absolute",
										left: 0,
										top: "50%",
										transform: "translateY(-50%)",
										width: 3,
										height: 16,
										background: "#000",
										borderRadius: 2,
									}}
								/>
							)}
							{item.icon}
							{item.label}
						</button>
					);
				})}
			</div>
			<Card style={{ flex: 1 }} data-testid="settings-tab-content">
				<Typography.Title level={4} style={{ marginTop: 0 }}>
					{active.label}
				</Typography.Title>
				{active.content}
			</Card>
		</Flex>
	);
}
