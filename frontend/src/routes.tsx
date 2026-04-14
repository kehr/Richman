import { AuthGuard } from "@/domain/auth/auth-guard";
import { MainLayout } from "@/layouts/MainLayout";
import { Spin } from "@/ui-kit/eat";
import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router";

const LoginPage = lazy(() => import("@/pages/auth/LoginPage"));
const RegisterPage = lazy(() => import("@/pages/auth/RegisterPage"));
const MarketOverviewPage = lazy(() => import("@/pages/market-overview/market-overview-page"));
const AssetDetailPage = lazy(() => import("@/pages/asset-detail"));
const BriefingPage = lazy(() => import("@/pages/briefing/briefing-page"));
const PortfolioListPage = lazy(() => import("@/pages/portfolio/PortfolioListPage"));
const PortfolioEditPage = lazy(() => import("@/pages/portfolio/PortfolioEditPage"));
const PortfolioTransactionsPage = lazy(() => import("@/pages/portfolio/PortfolioTransactionsPage"));
const SettingsPage = lazy(() => import("@/pages/settings/SettingsPage"));
const RiskPreferenceSubPage = lazy(() => import("@/pages/settings/RiskPreferenceSubPage"));
const HelpPage = lazy(() => import("@/pages/help/HelpPage"));

function PageLoading() {
	return (
		<div
			style={{
				display: "flex",
				justifyContent: "center",
				alignItems: "center",
				height: "100vh",
			}}
		>
			<Spin size="large" />
		</div>
	);
}

// AppShell wraps protected routes with JWT authentication and the main layout.
// OnboardingGuard has been removed in v2: users go directly to /market after
// registration with no wizard flow.
function AppShell() {
	return (
		<AuthGuard>
			<MainLayout />
		</AuthGuard>
	);
}

// PublicShell renders pages within MainLayout but without the AuthGuard,
// allowing unauthenticated visitors to browse market data.
function PublicShell() {
	return <MainLayout />;
}

export function AppRoutes() {
	return (
		<Suspense fallback={<PageLoading />}>
			<Routes>
				{/* Auth routes (unauthenticated only, no layout shell) */}
				<Route path="/login" element={<LoginPage />} />
				<Route path="/register" element={<RegisterPage />} />

				{/* Public routes — accessible without JWT, rendered inside MainLayout */}
				<Route element={<PublicShell />}>
					<Route path="/market" element={<MarketOverviewPage />} />
					<Route path="/market/:code" element={<AssetDetailPage />} />
					<Route path="/help" element={<HelpPage />} />
				</Route>

				{/* Protected routes — require valid JWT */}
				<Route element={<AppShell />}>
					<Route path="/briefing" element={<BriefingPage />} />
					<Route path="/portfolio" element={<PortfolioListPage />} />
					<Route path="/portfolio/:id" element={<PortfolioEditPage />} />
					<Route path="/portfolio/:id/transactions" element={<PortfolioTransactionsPage />} />
					<Route path="/settings" element={<SettingsPage />} />
					<Route path="/settings/risk-preference" element={<RiskPreferenceSubPage />} />
				</Route>

				{/* Root redirect and catch-all */}
				<Route path="/" element={<Navigate to="/market" replace />} />
				<Route path="*" element={<Navigate to="/market" replace />} />
			</Routes>
		</Suspense>
	);
}
