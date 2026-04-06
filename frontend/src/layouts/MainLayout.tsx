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
	const { data: userData } = useCurrentUser();

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
				<a
					href="/help"
					onClick={(e) => {
						e.preventDefault();
						navigate("/help");
					}}
					style={{
						display: "flex",
						alignItems: "center",
						gap: 8,
						padding: "12px 16px",
						color: "inherit",
					}}
				>
					<QuestionCircleOutlined />
					<span>Help</span>
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
