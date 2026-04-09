import { useRerunAnalysis } from "@/features/decision-card";
import { useMarkOnboardingCompleted } from "@/features/user-settings";
import { Alert, Button, Space, Spin, Typography } from "@/ui-kit/eat";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { OnboardingLayout } from "./components/OnboardingLayout";

const { Text } = Typography;

const ANALYSIS_STEPS = ["拉取行情数据", "计算三维信号", "LLM 增强催化剂", "生成决策卡"];

const STEP_INTERVAL_MS = 4000;

type StepStatus = "pending" | "active" | "done";

// FirstAnalysisPage drives the visible progress of the mandatory first
// analysis. The backend call is fire-and-forget (no task polling yet), so we
// advance a local timer through four canned steps while the mutation runs in
// the background. Once both the timer and the mutation have settled we mark
// onboarding complete and redirect to /dashboard.
export default function FirstAnalysisPage() {
	const navigate = useNavigate();
	const rerunAnalysis = useRerunAnalysis();
	const markCompleted = useMarkOnboardingCompleted();

	const [currentStep, setCurrentStep] = useState(0);
	const [error, setError] = useState<string | null>(null);
	const startedRef = useRef(false);
	const completedRef = useRef(false);

	// Kick off the analysis mutation exactly once on mount. StrictMode would
	// otherwise double-fire and produce two /analysis/trigger requests. The
	// rerunAnalysis mutation object is stable across renders so we intentionally
	// omit it from the dependency array via the biome ignore comment below.
	// biome-ignore lint/correctness/useExhaustiveDependencies: mount-only effect
	useEffect(() => {
		if (startedRef.current) return;
		startedRef.current = true;
		rerunAnalysis.mutateAsync().catch((err: unknown) => {
			const msg = err instanceof Error ? err.message : "分析触发失败";
			setError(msg);
		});
	}, []);

	// Advance the visible step indicator on a fixed cadence. The interval is
	// cleared when the component unmounts or once all steps are visible.
	useEffect(() => {
		if (error) return;
		if (currentStep >= ANALYSIS_STEPS.length) return;
		const timer = setTimeout(() => {
			setCurrentStep((prev) => prev + 1);
		}, STEP_INTERVAL_MS);
		return () => clearTimeout(timer);
	}, [currentStep, error]);

	// Finalise onboarding once every step has been shown. We intentionally do
	// NOT gate on rerunAnalysis.isPending: the backend accepts the trigger as
	// fire-and-forget (202 Accepted) and the real analysis work runs in a
	// detached goroutine, so the mutation's pending flag is only meaningful
	// for the HTTP round-trip. Blocking the UI on it previously caused users
	// to hang on the "all done" screen when the mutation state failed to
	// settle for any reason (flaky network, stale reference, data source
	// failure mid-request). completedRef guards against double-firing.
	// `markCompleted` and `navigate` are stable references from TanStack Query
	// and React Router v7, so we only react to the currentStep counter and
	// the error state.
	// biome-ignore lint/correctness/useExhaustiveDependencies: markCompleted and navigate are stable
	useEffect(() => {
		if (completedRef.current) return;
		if (error) return;
		if (currentStep < ANALYSIS_STEPS.length) return;
		completedRef.current = true;
		markCompleted
			.mutateAsync()
			.then(() => {
				navigate("/dashboard", { replace: true });
			})
			.catch(() => {
				// Even if the mark-complete call fails, free the user from the
				// progress screen so they are not stuck. OnboardingGuard will
				// simply put them back here on the next hard reload, which is a
				// better failure mode than an infinite spinner.
				navigate("/dashboard", { replace: true });
			});
	}, [currentStep, error]);

	const handleRetry = () => {
		// Reset UI state and re-fire the mutation directly; the mount-only
		// effect above will not re-run (empty deps), so startedRef stays true.
		setError(null);
		setCurrentStep(0);
		completedRef.current = false;
		rerunAnalysis.mutateAsync().catch((err: unknown) => {
			const msg = err instanceof Error ? err.message : "分析触发失败";
			setError(msg);
		});
	};

	const handleSkip = () => {
		if (completedRef.current) return;
		completedRef.current = true;
		markCompleted.mutateAsync().finally(() => {
			navigate("/dashboard", { replace: true });
		});
	};

	const statusOf = (index: number): StepStatus => {
		if (index < currentStep) return "done";
		if (index === currentStep) return "active";
		return "pending";
	};

	// Show an always-available skip link once every step has ticked through.
	// In the happy path the finalise effect auto-advances, but if anything
	// further downstream (markCompleted, navigate) hangs the user can bail
	// out manually instead of being trapped on the progress screen.
	const showSkipEscape = currentStep >= ANALYSIS_STEPS.length && !error;

	return (
		<OnboardingLayout
			currentStep={5}
			title="正在为你生成第一张决策卡"
			description="这一步只需十几秒，Richman 会扫描你的持仓并跑一遍三维分析。"
		>
			{error ? (
				<Alert
					type="error"
					showIcon
					message="首次分析失败"
					description={error}
					style={{ marginBottom: 16 }}
					action={
						<Space>
							<Button size="small" onClick={handleRetry} data-testid="onboarding-analysis-retry">
								重试
							</Button>
							<Button
								size="small"
								type="primary"
								onClick={handleSkip}
								data-testid="onboarding-analysis-skip"
							>
								跳过先看 Dashboard
							</Button>
						</Space>
					}
				/>
			) : null}

			<ol
				data-testid="onboarding-analysis-steps"
				style={{
					listStyle: "none",
					padding: 0,
					margin: 0,
					display: "flex",
					flexDirection: "column",
					gap: 16,
				}}
			>
				{ANALYSIS_STEPS.map((label, index) => {
					const status = statusOf(index);
					return (
						<li
							key={label}
							data-testid={`analysis-step-${index}`}
							data-status={status}
							style={{
								display: "flex",
								alignItems: "center",
								gap: 12,
								padding: 16,
								borderRadius: 8,
								border: "1px solid",
								borderColor: status === "active" ? "#000" : "#f0f0f0",
								backgroundColor: status === "done" ? "#f6ffed" : "#fff",
								opacity: status === "pending" ? 0.5 : 1,
							}}
						>
							<div
								style={{
									width: 28,
									height: 28,
									borderRadius: 999,
									display: "flex",
									alignItems: "center",
									justifyContent: "center",
									backgroundColor:
										status === "done" ? "#52c41a" : status === "active" ? "#000" : "#d9d9d9",
									color: "#fff",
									fontSize: 14,
									fontWeight: 600,
								}}
							>
								{status === "done" ? "✓" : index + 1}
							</div>
							<Text style={{ flex: 1, fontSize: 15 }}>{label}</Text>
							{status === "active" ? <Spin size="small" /> : null}
						</li>
					);
				})}
			</ol>

			{showSkipEscape ? (
				<div style={{ marginTop: 16, textAlign: "center" }}>
					<Button
						type="link"
						onClick={handleSkip}
						data-testid="onboarding-analysis-manual-continue"
					>
						看起来卡住了？直接进 Dashboard
					</Button>
				</div>
			) : null}
		</OnboardingLayout>
	);
}
