import { layoutToken } from "@/config/theme";
import { clearAuth } from "@/domain/auth/storage";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import {
	Avatar,
	BellOutlined,
	DashboardOutlined,
	Dropdown,
	FundOutlined,
	LineChartOutlined,
	LogoutOutlined,
	PieChartOutlined,
	ProLayout,
	SettingOutlined,
	Space,
	UserOutlined,
} from "@/ui-kit/eat";
import { useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";

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

export function MainLayout() {
	const navigate = useNavigate();
	const location = useLocation();
	const { data: userData } = useCurrentUser();
	const [collapsed, setCollapsed] = useState(false);

	const user = userData?.data;
	const displayName = user?.email?.split("@")[0] || "User";

	const handleLogout = () => {
		clearAuth();
		navigate("/login", { replace: true });
	};

	return (
		<ProLayout
			title="Richman"
			logo="/logo.svg"
			layout="side"
			fixSiderbar
			token={layoutToken}
			collapsed={false}
			collapsedButtonRender={false}
			// collapsed={collapsed}
			onCollapse={setCollapsed}
			location={{ pathname: location.pathname }}
			route={menuRoutes}
			menuItemRender={(item, dom) => (
				<a
					href={item.path || "#"}
					onClick={(e) => {
						e.preventDefault();
						if (item.path) navigate(item.path);
					}}
				>
					{dom}
				</a>
			)}
			actionsRender={() => [
				<Dropdown
					key="user"
					menu={{
						items: [
							{
								key: "logout",
								icon: <LogoutOutlined />,
								label: "Logout",
								danger: true,
								onClick: handleLogout,
							},
						],
					}}
					placement="bottomRight"
				>
					<Space style={{ cursor: "pointer", padding: "0 8px" }}>
						<Avatar size="small" icon={<UserOutlined />} />
						<span>{displayName}</span>
					</Space>
				</Dropdown>,
			]}
		>
			<Outlet />
		</ProLayout>
	);
}
