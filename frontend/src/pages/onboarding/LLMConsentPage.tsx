import { useDashboardSummary } from "@/features/dashboard-summary";
import { useLLMConsent } from "@/features/settings-llm";
import { Alert, App, Button, Card, Space, Typography } from "@/ui-kit/eat";
import { useState } from "react";
import { useNavigate } from "react-router";
import { OnboardingLayout } from "./components/OnboardingLayout";
import { useOnboardingNav } from "./use-onboarding-nav";

const { Text, Title } = Typography;

// LLMConsentPage is the fourth onboarding step introduced by the LLM
// degraded contract work. The user is presented with two choices:
//
//   - Skip: use the rules engine, cards will show the "Rules" pill. The
//     useSystemDefaultWhenUnconfigured consent is set to false.
//   - Try AI: if the system-default provider is reachable, consent is
//     set to true and the user continues to step 5. Otherwise the user
//     is redirected to /settings?tab=ai&from=onboarding so they can
//     configure their own provider before advancing.
//
// Either choice calls POST /onboarding/llm-consent with the chosen
// boolean so the backend writes users.use_system_default_llm_consent
// atomically.
export default function LLMConsentPage() {
	const { message } = App.useApp();
	const navigate = useNavigate();
	const nav = useOnboardingNav();
	const dashboardQuery = useDashboardSummary();
	const consentMutation = useLLMConsent();
	const [pendingChoice, setPendingChoice] = useState<"skip" | "try" | null>(null);

	const systemDefaultAvailable = dashboardQuery.data?.llmStatus.systemDefaultAvailable ?? false;

	const handleSkip = async () => {
		setPendingChoice("skip");
		try {
			await consentMutation.mutateAsync({ useSystemDefault: false });
			message.success("已跳过 AI 解读，当前使用规则引擎。");
			await nav.next();
		} catch (err) {
			const msg = err instanceof Error ? err.message : "请稍后再试";
			message.error(`保存失败：${msg}`);
		} finally {
			setPendingChoice(null);
		}
	};

	const handleTryAI = async () => {
		setPendingChoice("try");
		try {
			if (systemDefaultAvailable) {
				await consentMutation.mutateAsync({ useSystemDefault: true });
				message.success("已启用系统默认 AI Provider。");
				await nav.next();
			} else {
				// System default is unavailable — route the user to the settings
				// page so they can configure their own provider. The from=onboarding
				// query tells the settings tab to return the user to onboarding
				// after a successful save (future enhancement; link today is
				// one-way).
				await consentMutation.mutateAsync({ useSystemDefault: false });
				message.info("系统默认 AI Provider 不可用，请配置你自己的 Provider。");
				navigate("/settings?tab=ai&from=onboarding");
			}
		} catch (err) {
			const msg = err instanceof Error ? err.message : "请稍后再试";
			message.error(`保存失败：${msg}`);
		} finally {
			setPendingChoice(null);
		}
	};

	const busy = consentMutation.isPending || pendingChoice !== null;

	return (
		<OnboardingLayout
			currentStep={4}
			title="AI 解读 Provider"
			description="Richman 可以用大语言模型为你的持仓生成更自然的解读。你可以选择跳过、使用系统默认或稍后配置自己的 Provider。"
		>
			<Space
				direction="vertical"
				size={20}
				style={{ width: "100%" }}
				data-testid="llm-consent-page"
			>
				<Card
					title={
						<Title level={5} style={{ margin: 0 }}>
							跳过，使用规则引擎
						</Title>
					}
					data-testid="llm-consent-skip-card"
				>
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						<Text type="secondary">
							分析卡片会以"Rules"角标呈现。你可以随时回到设置页打开 AI 解读。
						</Text>
						<Button
							onClick={handleSkip}
							loading={pendingChoice === "skip"}
							disabled={busy && pendingChoice !== "skip"}
							data-testid="llm-consent-skip-button"
						>
							跳过
						</Button>
					</Space>
				</Card>

				<Card
					title={
						<Title level={5} style={{ margin: 0 }}>
							我想试试 AI 解读
						</Title>
					}
					data-testid="llm-consent-try-card"
				>
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						{systemDefaultAvailable ? (
							<Alert
								type="info"
								showIcon
								message="使用 Richman 默认 AI Provider"
								description="你的持仓数据将以加密传输方式发给 Richman 的默认 AI Provider 做分析。勾选同意后生效，你可以随时在设置里关闭或改配自己的 Provider。"
							/>
						) : (
							<Alert
								type="warning"
								showIcon
								message="系统默认 Provider 当前不可用"
								description="你需要配置自己的 LLM Provider 才能启用 AI 解读。点击下方按钮会带你到设置页面。"
							/>
						)}
						<Button
							type="primary"
							onClick={handleTryAI}
							loading={pendingChoice === "try"}
							disabled={busy && pendingChoice !== "try"}
							data-testid="llm-consent-try-button"
						>
							{systemDefaultAvailable ? "同意并启用" : "去配置我的 Provider"}
						</Button>
					</Space>
				</Card>
			</Space>
		</OnboardingLayout>
	);
}
