import { layoutToken } from "@/config/theme";
import { gravatarUrl } from "@/domain/auth/gravatar";
import { clearAuth } from "@/domain/auth/storage";
import { useCurrentUser } from "@/domain/auth/use-current-user";
import {
	Avatar,
	Dropdown,
	GlobalOutlined,
	LogoutOutlined,
	ProLayout,
	QuestionCircleOutlined,
	SettingOutlined,
	Space,
	UserOutlined,
} from "@/ui-kit/eat";
import { Briefcase, LineChart, TrendingUp } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useLocation, useNavigate } from "react-router";

export function MainLayout() {
	const navigate = useNavigate();
	const location = useLocation();
	const { data: user } = useCurrentUser();
	const { t, i18n } = useTranslation();

	const isAuthenticated = !!user;
	const displayName = user?.email?.split("@")[0] || "—";

	// Authenticated menu: Market + Portfolio + Briefing
	const authenticatedRoutes = useMemo(
		() => ({
			path: "/",
			routes: [
				{ path: "/market", name: t("nav.market"), icon: <LineChart size={14} /> },
				{ path: "/portfolio", name: t("nav.portfolio"), icon: <Briefcase size={14} /> },
				{ path: "/briefing", name: t("nav.briefing"), icon: <TrendingUp size={14} /> },
			],
		}),
		[t],
	);

	// Unauthenticated menu: Market only
	const publicRoutes = useMemo(
		() => ({
			path: "/",
			routes: [{ path: "/market", name: t("nav.market"), icon: <LineChart size={14} /> }],
		}),
		[t],
	);

	const menuRoutes = isAuthenticated ? authenticatedRoutes : publicRoutes;

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
				else if (key === "help") navigate("/help");
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
					key: "help",
					icon: <QuestionCircleOutlined />,
					label: t("nav.help"),
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

	// Actions rendered in the top-right area of the navigation bar.
	// Authenticated: language switcher + user avatar dropdown.
	// Unauthenticated: language switcher + Login link + Register link.
	const actionsRender = () => {
		const langSwitcher = (
			<Dropdown key="lang" menu={langMenu} placement="bottom">
				<GlobalOutlined aria-label={t("nav.language")} style={{ fontSize: 14 }} />
			</Dropdown>
		);

		if (isAuthenticated) {
			return [
				langSwitcher,
				<Dropdown key="user" menu={userMenu} placement="bottom">
					<Space style={{ paddingBlock: 0 }}>
						<Avatar src={gravatarUrl(user?.email ?? "", 32)} icon={<UserOutlined />} size={24} />
						<span style={{ fontSize: 14 }}>{displayName}</span>
					</Space>
				</Dropdown>,
			];
		}

		return [
			langSwitcher,
			<Link
				key="login"
				to="/login"
				style={{ fontSize: 14, color: "inherit", textDecoration: "none" }}
			>
				{t("nav.login")}
			</Link>,
			<Link
				key="register"
				to="/register"
				style={{ fontSize: 14, color: "inherit", textDecoration: "none" }}
			>
				{t("nav.register")}
			</Link>,
		];
	};

	return (
		<ProLayout
			title="Richman"
			logo="/logo.svg"
			layout="top"
			contentWidth="Fixed"
			token={layoutToken}
			location={{ pathname: location.pathname }}
			route={menuRoutes}
			menuHeaderRender={(logo, title) => (
				<Link to="/" style={{ display: "flex", alignItems: "center" }}>
					{logo}
					{title}
				</Link>
			)}
			menuItemRender={(item, dom) => <Link to={item.path || "#"}>{dom}</Link>}
			actionsRender={actionsRender}
		>
			<Outlet />
		</ProLayout>
	);
}
