import { layoutToken } from "@/config/theme";
import { clearAuth } from "@/domain/auth/storage";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import {
	Avatar,
	DashboardOutlined,
	Dropdown,
	GlobalOutlined,
	LogoutOutlined,
	PieChartOutlined,
	ProLayout,
	QuestionCircleOutlined,
	SettingOutlined,
	Space,
	UserOutlined,
} from "@/ui-kit/eat";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Outlet, useLocation, useNavigate } from "react-router";

export function MainLayout() {
	const navigate = useNavigate();
	const location = useLocation();
	const { data: user } = useCurrentUser();
	const { t, i18n } = useTranslation();

	const displayName = user?.email?.split("@")[0] || "User";

	const menuRoutes = useMemo(
		() => ({
			path: "/",
			routes: [
				{ path: "/dashboard", name: t("nav.dashboard"), icon: <DashboardOutlined /> },
				{ path: "/portfolio", name: t("nav.portfolio"), icon: <PieChartOutlined /> },
			],
		}),
		[t],
	);

	const userMenu = useMemo(
		() => ({
			selectedKeys: [`lang-${i18n.language}`],
			onClick: ({ key }: { key: string }) => {
				if (key === "settings") navigate("/settings");
				else if (key === "lang-en") i18n.changeLanguage("en");
				else if (key === "lang-zh") i18n.changeLanguage("zh");
				else if (key === "logout") {
					clearAuth();
					navigate("/login", { replace: true });
				}
			},
			items: [
				{
					key: "settings",
					icon: <SettingOutlined />,
					label: t("nav.settings"),
				},
				{
					key: "language",
					icon: <GlobalOutlined />,
					label: t("nav.language"),
					// popupClassName: "lang-submenu-popup",
					children: [
						{ key: "lang-en", label: "English" },
						{ key: "lang-zh", label: "中文" },
					],
				},
				{ type: "divider" as const },
				{
					key: "logout",
					icon: <LogoutOutlined />,
					label: t("nav.logout"),
					danger: true,
				},
			],
		}),
		[t, i18n, navigate],
	);

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
				<div
					style={{
						display: "flex",
						alignItems: "center",
						justifyContent: "space-between",
						gap: 8,
						padding: "12px 16px",
					}}
				>
					<Dropdown menu={userMenu} placement="topLeft" arrow>
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
						aria-label={t("nav.help")}
						style={{
							display: "inline-flex",
							alignItems: "center",
							color: "inherit",
							flexShrink: 0,
						}}
					>
						<QuestionCircleOutlined />
					</a>
				</div>
			)}
		>
			<Outlet />
		</ProLayout>
	);
}
