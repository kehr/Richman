import { layoutToken } from "@/config/theme";
import { clearAuth } from "@/domain/auth/storage";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import {
	Avatar,
	Dropdown,
	GlobalOutlined,
	LineChartOutlined,
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
import { Link, Outlet, useLocation, useNavigate } from "react-router";

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
				{ path: "/briefing", name: t("nav.briefing"), icon: <LineChartOutlined /> },
				{ path: "/portfolio", name: t("nav.portfolio"), icon: <PieChartOutlined /> },
			],
		}),
		[t],
	);

	const langMenu = useMemo(
		() => ({
			selectedKeys: [i18n.language],
			onClick: ({ key }: { key: string }) => i18n.changeLanguage(key),
			items: [
				{ key: "en", label: "English" },
				{ key: "zh", label: "中文" },
			],
		}),
		[i18n],
	);

	const userMenu = useMemo(
		() => ({
			onClick: ({ key }: { key: string }) => {
				if (key === "settings") navigate("/settings");
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
				{ type: "divider" as const },
				{
					key: "logout",
					icon: <LogoutOutlined />,
					label: t("nav.logout"),
					danger: true,
				},
			],
		}),
		[t, navigate],
	);

	return (
		<ProLayout
			title="Richman"
			logo="/logo.svg"
			layout="top"
			contentWidth="Fixed"
			token={layoutToken}
			location={{ pathname: location.pathname }}
			route={menuRoutes}
			menuItemRender={(item, dom) => <Link to={item.path || "#"}>{dom}</Link>}
			actionsRender={() => [
				<Link
					key="help"
					to="/help"
					aria-label={t("nav.help")}
					style={{ display: "inline-flex", alignItems: "center", color: "inherit" }}
				>
					<QuestionCircleOutlined style={{ fontSize: 14 }} />
				</Link>,
				<Dropdown key="lang" menu={langMenu} placement="bottom">
					<GlobalOutlined aria-label={t("nav.language")} style={{ fontSize: 14 }} />
				</Dropdown>,

				<Dropdown key="user" menu={userMenu} placement="bottom">
					<Space
						style={{
							paddingBlock: 0,
						}}
					>
						<Avatar icon={<UserOutlined />} />
						<span>{displayName}</span>
					</Space>
				</Dropdown>,
			]}
		>
			<Outlet />
		</ProLayout>
	);
}
