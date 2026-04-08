import { layoutToken } from "@/config/theme";
import { clearAuth } from "@/domain/auth/storage";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import {
	Avatar,
	DashboardOutlined,
	Dropdown,
	LogoutOutlined,
	PieChartOutlined,
	ProLayout,
	QuestionCircleOutlined,
	SettingOutlined,
	Space,
	UserOutlined,
} from "@/ui-kit/eat";
import { Outlet, useLocation, useNavigate } from "react-router";

const menuRoutes = {
	path: "/",
	routes: [
		{ path: "/dashboard", name: "Dashboard", icon: <DashboardOutlined /> },
		{ path: "/portfolio", name: "Portfolio", icon: <PieChartOutlined /> },
		{ path: "/settings", name: "Settings", icon: <SettingOutlined /> },
	],
};

export function MainLayout() {
	const navigate = useNavigate();
	const location = useLocation();
	const { data: user } = useCurrentUser();

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
			menuFooterRender={() => (
				// Sidebar footer is a single horizontal row: avatar + user label
				// pinned to the left, help icon pinned to the right. Clicking the
				// avatar block opens the logout dropdown; clicking the help icon
				// navigates to /help. Using a flex row with space-between avoids
				// ProLayout's default stacked layout for actionsRender +
				// menuFooterRender so the two pieces sit on the same line.
				<div
					style={{
						display: "flex",
						alignItems: "center",
						justifyContent: "space-between",
						gap: 8,
						padding: "12px 16px",
					}}
				>
					<Dropdown
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
						placement="topLeft"
					>
						<Space style={{ cursor: "pointer", minWidth: 0 }}>
							<Avatar size="small" icon={<UserOutlined />} />
							<span
								style={{
									overflow: "hidden",
									textOverflow: "ellipsis",
									whiteSpace: "nowrap",
								}}
							>
								{displayName}
							</span>
						</Space>
					</Dropdown>
					<a
						href="/help"
						onClick={(e) => {
							e.preventDefault();
							navigate("/help");
						}}
						aria-label="Help"
						style={{
							display: "inline-flex",
							alignItems: "center",
							gap: 4,
							color: "inherit",
							flexShrink: 0,
						}}
					>
						<QuestionCircleOutlined />
						<span>Help</span>
					</a>
				</div>
			)}
		>
			<Outlet />
		</ProLayout>
	);
}
