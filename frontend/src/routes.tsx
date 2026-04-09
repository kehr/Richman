import { AuthGuard } from "@/domain/auth/auth-guard";
import { OnboardingGuard } from "@/domain/auth/onboarding-guard";
import { MainLayout } from "@/layouts/MainLayout";
import { OnboardingStateProvider } from "@/pages/onboarding/state";
import { Spin } from "@/ui-kit/eat";
import { Suspense, lazy } from "react";
import { Navigate, Outlet, Route, Routes } from "react-router";

const LoginPage = lazy(() => import("@/pages/auth/LoginPage"));
const RegisterPage = lazy(() => import("@/pages/auth/RegisterPage"));
const WelcomePage = lazy(() => import("@/pages/onboarding/WelcomePage"));
const CategoriesPage = lazy(() => import("@/pages/onboarding/CategoriesPage"));
const FirstHoldingPage = lazy(() => import("@/pages/onboarding/FirstHoldingPage"));
const LLMConsentPage = lazy(() => import("@/pages/onboarding/LLMConsentPage"));
const FirstAnalysisPage = lazy(() => import("@/pages/onboarding/FirstAnalysisPage"));
const DashboardPage = lazy(() => import("@/pages/dashboard/DashboardPage"));
const PortfolioListPage = lazy(() => import("@/pages/portfolio/PortfolioListPage"));
const PortfolioEditPage = lazy(() => import("@/pages/portfolio/PortfolioEditPage"));
const PortfolioTransactionsPage = lazy(() => import("@/pages/portfolio/PortfolioTransactionsPage"));
const DecisionCardDetailPage = lazy(() => import("@/pages/decision-cards/DecisionCardDetailPage"));
const SettingsPage = lazy(() => import("@/pages/settings/SettingsPage"));
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

// OnboardingShell renders the onboarding branch outside of MainLayout so users
// do not see the primary navigation shell until they finish the flow. The
// OnboardingStateProvider is mounted INSIDE the auth + onboarding guards so
// unauthenticated users never allocate the provider; every onboarding page
// and every helper hook (useOnboardingState / useOnboardingNav) shares the
// same provider instance for the full /onboarding/* branch.
function OnboardingShell() {
	return (
		<AuthGuard>
			<OnboardingGuard>
				<OnboardingStateProvider>
					<Outlet />
				</OnboardingStateProvider>
			</OnboardingGuard>
		</AuthGuard>
	);
}

function AppShell() {
	return (
		<AuthGuard>
			<OnboardingGuard>
				<MainLayout />
			</OnboardingGuard>
		</AuthGuard>
	);
}

export function AppRoutes() {
	return (
		<Suspense fallback={<PageLoading />}>
			<Routes>
				{/* Public routes */}
				<Route path="/login" element={<LoginPage />} />
				<Route path="/register" element={<RegisterPage />} />

				{/* Onboarding routes (authenticated, no main shell). */}
				<Route element={<OnboardingShell />}>
					<Route path="/onboarding/welcome" element={<WelcomePage />} />
					<Route path="/onboarding/categories" element={<CategoriesPage />} />
					<Route path="/onboarding/first-holding" element={<FirstHoldingPage />} />
					<Route path="/onboarding/llm-consent" element={<LLMConsentPage />} />
					<Route path="/onboarding/first-analysis" element={<FirstAnalysisPage />} />
				</Route>

				{/* Main app routes. */}
				<Route element={<AppShell />}>
					<Route path="/" element={<Navigate to="/dashboard" replace />} />
					<Route path="/dashboard" element={<DashboardPage />} />
					<Route path="/portfolio" element={<PortfolioListPage />} />
					<Route path="/portfolio/:id" element={<PortfolioEditPage />} />
					<Route path="/portfolio/:id/transactions" element={<PortfolioTransactionsPage />} />
					<Route path="/decision-cards/:id" element={<DecisionCardDetailPage />} />
					<Route path="/settings" element={<SettingsPage />} />
					<Route path="/help" element={<HelpPage />} />
				</Route>

				{/* Catch-all */}
				<Route path="*" element={<Navigate to="/dashboard" replace />} />
			</Routes>
		</Suspense>
	);
}
