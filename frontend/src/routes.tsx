import { AuthGuard } from "@/domain/auth/auth-guard";
import { MainLayout } from "@/layouts/MainLayout";
import { Spin } from "@/ui-kit/eat";
import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router";

const LoginPage = lazy(() => import("@/pages/auth/LoginPage"));
const RegisterPage = lazy(() => import("@/pages/auth/RegisterPage"));
const DashboardPage = lazy(() => import("@/pages/dashboard/DashboardPage"));
const PortfolioListPage = lazy(() => import("@/pages/portfolio/PortfolioListPage"));
const PortfolioNewPage = lazy(() => import("@/pages/portfolio/PortfolioNewPage"));
const PortfolioEditPage = lazy(() => import("@/pages/portfolio/PortfolioEditPage"));
const AnalysisPage = lazy(() => import("@/pages/analysis/AnalysisPage"));
const DecisionCardListPage = lazy(() => import("@/pages/decision-cards/DecisionCardListPage"));
const DecisionCardDetailPage = lazy(() => import("@/pages/decision-cards/DecisionCardDetailPage"));
const NotificationsPage = lazy(() => import("@/pages/notifications/NotificationsPage"));
const SettingsPage = lazy(() => import("@/pages/settings/SettingsPage"));

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

export function AppRoutes() {
	return (
		<Suspense fallback={<PageLoading />}>
			<Routes>
				{/* Public routes */}
				<Route path="/login" element={<LoginPage />} />
				<Route path="/register" element={<RegisterPage />} />

				{/* Protected routes */}
				<Route
					element={
						<AuthGuard>
							<MainLayout />
						</AuthGuard>
					}
				>
					<Route path="/" element={<Navigate to="/dashboard" replace />} />
					<Route path="/dashboard" element={<DashboardPage />} />
					<Route path="/portfolio" element={<PortfolioListPage />} />
					<Route path="/portfolio/new" element={<PortfolioNewPage />} />
					<Route path="/portfolio/:id" element={<PortfolioEditPage />} />
					<Route path="/analysis" element={<AnalysisPage />} />
					<Route path="/decision-cards" element={<DecisionCardListPage />} />
					<Route path="/decision-cards/:id" element={<DecisionCardDetailPage />} />
					<Route path="/notifications" element={<NotificationsPage />} />
					<Route path="/settings" element={<SettingsPage />} />
				</Route>

				{/* Catch-all */}
				<Route path="*" element={<Navigate to="/dashboard" replace />} />
			</Routes>
		</Suspense>
	);
}
