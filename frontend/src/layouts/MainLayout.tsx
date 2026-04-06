"use client";

import { clearAuth } from "@/domain/auth/storage";
import { useThemeMode } from "@/domain/ui/use-theme";
import {
	BellOutlined,
	DashboardOutlined,
	Dropdown,
	FundOutlined,
	LineChartOutlined,
	LogoutOutlined,
	MoonOutlined,
	PieChartOutlined,
	ProLayout,
	SettingOutlined,
	SunOutlined,
	Switch,
	Typography,
	UserOutlined,
} from "@/ui-kit/eat";
import { usePathname, useRouter } from "next/navigation";
import { type ReactNode, useState } from "react";

const { Text } = Typography;

interface MainLayoutProps {
	children: ReactNode;
}

const menuRoutes = {
	path: "/",
	routes: [
		{ path: "/dashboard", name: "Dashboard", icon: <DashboardOutlined /> },
		{ path: "/portfolio", name: "Portfolio", icon: <PieChartOutlined /> },
		{ path: "/analysis", name: "Analysis", icon: <FundOutlined /> },
		{ path: "/decision-cards", name: "Decision Cards", icon: <LineChartOutlined /> },
		{ path: "/notifications", name: "Notifications", icon: <BellOutlined /> },
		{ path: "/settings", name: "Settings", icon: <SettingOutlined /> },
	],
};

export function MainLayout({ children }: MainLayoutProps) {
	const router = useRouter();
	const pathname = usePathname();
	const { mode, toggle } = useThemeMode();
	const [collapsed, setCollapsed] = useState(false);

	const handleLogout = () => {
		clearAuth();
		router.replace("/login");
	};

	return (
		<ProLayout
			title="Richman"
			layout="mix"
			fixSiderbar
			collapsed={collapsed}
			onCollapse={setCollapsed}
			location={{ pathname }}
			route={menuRoutes}
			menuItemRender={(item, dom) => (
				<a
					href={item.path || "#"}
					onClick={(e) => {
						e.preventDefault();
						if (item.path) router.push(item.path);
					}}
				>
					{dom}
				</a>
			)}
			actionsRender={() => [
				<Switch
					key="theme"
					checkedChildren={<MoonOutlined />}
					unCheckedChildren={<SunOutlined />}
					checked={mode === "dark"}
					onChange={toggle}
				/>,
				<Dropdown
					key="user"
					menu={{
						items: [
							{
								key: "logout",
								icon: <LogoutOutlined />,
								label: "Logout",
								onClick: handleLogout,
							},
						],
					}}
				>
					<span style={{ cursor: "pointer" }}>
						<UserOutlined />
						<Text style={{ marginLeft: 8 }}>User</Text>
					</span>
				</Dropdown>,
			]}
		>
			{children}
		</ProLayout>
	);
}
