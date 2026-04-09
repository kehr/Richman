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
	Tooltip,
	UserOutlined,
} from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Outlet, useLocation, useNavigate } from "react-router";

export function MainLayout() {
	const navigate = useNavigate();
	const location = useLocation();
	const { data: user } = useCurrentUser();
	const { t, i18n } = useTranslation();

	const displayName = user?.email?.split("@")[0] || "User";

	const handleLogout = () => {
		clearAuth();
		navigate("/login", { replace: true });
	};

	const menuRoutes = useMemo(
		() => ({
			path: "/",
			routes: [
				{ path: "/dashboard", name: t("nav.dashboard"), icon: <DashboardOutlined /> },
				{ path: "/portfolio", name: t("nav.portfolio"), icon: <PieChartOutlined /> },
				{ path: "/settings", name: t("nav.settings"), icon: <SettingOutlined /> },
			],
		}),
		[t],
	);

	const [langDropdownOpen, setLangDropdownOpen] = useState(false);

	const languageMenu = useMemo(
		() => ({
			items: [
				{ key: "en", label: "English" },
				{ key: "zh", label: "中文" },
			],
			selectedKeys: [i18n.language],
			onClick: ({ key }: { key: string }) => i18n.changeLanguage(key),
		}),
		[i18n],
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
									label: t("nav.logout"),
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
					<Tooltip title={t("nav.switchLanguage")} open={langDropdownOpen ? false : undefined}>
						<Dropdown menu={languageMenu} placement="topLeft" onOpenChange={setLangDropdownOpen}>
							<button
								type="button"
								style={{
									cursor: "pointer",
									background: "none",
									border: "none",
									padding: 0,
									color: "inherit",
									display: "inline-flex",
									alignItems: "center",
									gap: 8,
								}}
							>
								<GlobalOutlined />
								<span>{i18n.language === "zh" ? "中文" : "EN"}</span>
							</button>
						</Dropdown>
					</Tooltip>
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
						<span>{t("nav.help")}</span>
					</a>
				</div>
			)}
		>
			<Outlet />
		</ProLayout>
	);
}
