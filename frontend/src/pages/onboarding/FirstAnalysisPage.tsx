import { useRerunAnalysis } from "@/features/decision-card";
import { useMarkOnboardingCompleted } from "@/features/user-settings";
import { Alert, Button, Space, Spin, Typography } from "@/ui-kit/eat";
import { motion, useReducedMotion } from "framer-motion";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { OnboardingLayout } from "./components/OnboardingLayout";
import { useOnboardingState } from "./state";

const { Text } = Typography;

const ANALYSIS_STEPS = ["拉取行情数据", "计算三维信号", "LLM 增强催化剂", "生成决策卡"];

const STEP_INTERVAL_MS = 4000;

type StepStatus = "pending" | "active" | "done";

// FirstAnalysisPage drives the visible progress of the mandatory first
// analysis. The backend call is fire-and-forget (no task polling yet), so we
// advance a local timer through four canned steps while the mutation runs in
// the background. Once both the timer and the mutation have settled we mark
// onboarding complete and redirect to /dashboard.
//
// The one-shot analysis trigger is gated by `state.analysisFired` from the
// onboarding state provider (persisted in sessionStorage) so that navigating
// back to step 3 and forward again does NOT re-dispatch the mutation. The
// 4-step visual animation still plays on every mount so returning users see
// consistent feedback; only the underlying mutation is suppressed.
export default function FirstAnalysisPage() {
	const navigate = useNavigate();
	const rerunAnalysis = useRerunAnalysis();
	const markCompleted = useMarkOnboardingCompleted();
	const { state, update, clear } = useOnboardingState();
	const reducedMotion = useReducedMotion();

	const [currentStep, setCurrentStep] = useState(0);
	const [error, setError] = useState<string | null>(null);
	const completedRef = useRef(false);

	// Kick off the analysis mutation exactly once per onboarding session. The
	// guard reads `state.analysisFired` on first render only — subsequent
	// mounts (e.g. user navigates back to step 3 and returns) will see the
	// flag already true and skip the trigger. The empty dependency array is
	// intentional: adding `state.analysisFired` would re-run the effect every
	// time the state updates, which would either re-fire the mutation or be
	// a no-op depending on ordering. The current pattern is explicit and
	// mirrors the "mount-once seed" convention used on FirstHoldingPage.
	// biome-ignore lint/correctness/useExhaustiveDependencies: mount-once guard
	useEffect(() => {
		if (state.analysisFired) return;
		update({ analysisFired: true });
		rerunAnalysis.mutateAsync().catch((err: unknown) => {
			// The analysis call is fire-and-forget per the backend contract
			// (202 Accepted, the real work runs in a detached goroutine). A
			// rejection here means the HTTP round-trip itself failed, which we
			// surface as an error so the user can retry rather than silently
			// hanging on the progress screen.
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
	// `markCompleted`, `navigate` and `clear` are stable references from
	// TanStack Query, React Router v7, and the state provider respectively,
	// so we only react to the currentStep counter and the error state.
	// biome-ignore lint/correctness/useExhaustiveDependencies: markCompleted/navigate/clear are stable
	useEffect(() => {
		if (completedRef.current) return;
		if (error) return;
		if (currentStep < ANALYSIS_STEPS.length) return;
		completedRef.current = true;
		markCompleted
			.mutateAsync()
			.then(() => {
				// Clear the onboarding draft so the next session (if the user
				// ever re-enters the flow via Settings) starts fresh. The state
				// provider's own effect also wipes it once the status query
				// reports completed=true, but clearing here removes the race
				// where the user reaches /dashboard before the status refetch
				// has landed.
				clear();
				navigate("/dashboard", { replace: true });
			})
			.catch(() => {
				// Even if the mark-complete call fails, free the user from the
				// progress screen so they are not stuck. OnboardingGuard will
				// simply put them back here on the next hard reload, which is a
				// better failure mode than an infinite spinner. We deliberately
				// do NOT show an error toast here — a failed markCompleted at
				// the tail of the wizard should not ruin the "final step"
				// moment.
				navigate("/dashboard", { replace: true });
			});
	}, [currentStep, error]);

	const handleRetry = () => {
		// Reset UI state and re-fire the mutation directly. The mount-only
		// effect above will not re-run (empty deps), and `state.analysisFired`
		// is already true, so we bypass both guards by calling mutateAsync
		// directly. This is the one documented escape hatch from the
		// fire-once invariant: a user-initiated retry after an observed error.
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
			clear();
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
								{status === "done" ? (
									// framer-motion pathLength draw-in: animates the checkmark
									// stroke from 0 to 1 over 0.4s on transition to "done".
									// When the user prefers reduced motion we render the path
									// fully drawn so the visual payload is preserved without
									// the motion.
									<motion.svg
										width="16"
										height="16"
										viewBox="0 0 20 20"
										aria-hidden="true"
										data-testid={`analysis-step-check-${index}`}
									>
										<motion.path
											d="M4 10.5 L8 14.5 L16 6"
											fill="none"
											stroke="currentColor"
											strokeWidth="2.5"
											strokeLinecap="round"
											strokeLinejoin="round"
											initial={{ pathLength: reducedMotion ? 1 : 0 }}
											animate={{ pathLength: 1 }}
											transition={{ duration: reducedMotion ? 0 : 0.4, ease: "easeOut" }}
										/>
									</motion.svg>
								) : (
									index + 1
								)}
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
