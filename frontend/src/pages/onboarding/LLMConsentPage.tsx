import { useDashboardSummary } from "@/features/dashboard-summary";
import { useLLMConsent } from "@/features/settings-llm";
import { Alert, App, Button, Card, Space, Typography } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";
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
	const { t } = useTranslation("auth");
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
			message.success(t("onboarding.llmConsent.message.skipped"));
			await nav.next();
		} catch (err) {
			const msg = err instanceof Error ? err.message : "";
			message.error(t("onboarding.llmConsent.message.saveError", { msg }));
		} finally {
			setPendingChoice(null);
		}
	};

	const handleTryAI = async () => {
		setPendingChoice("try");
		try {
			if (systemDefaultAvailable) {
				await consentMutation.mutateAsync({ useSystemDefault: true });
				message.success(t("onboarding.llmConsent.message.enabled"));
				await nav.next();
			} else {
				// System default is unavailable — route the user to the settings
				// page so they can configure their own provider. The from=onboarding
				// query tells the settings tab to return the user to onboarding
				// after a successful save (future enhancement; link today is
				// one-way).
				await consentMutation.mutateAsync({ useSystemDefault: false });
				message.info(t("onboarding.llmConsent.message.unavailable"));
				navigate("/settings?tab=ai&from=onboarding");
			}
		} catch (err) {
			const msg = err instanceof Error ? err.message : "";
			message.error(t("onboarding.llmConsent.message.saveError", { msg }));
		} finally {
			setPendingChoice(null);
		}
	};

	const busy = consentMutation.isPending || pendingChoice !== null;

	return (
		<OnboardingLayout
			currentStep={4}
			title={t("onboarding.llmConsent.title")}
			description={t("onboarding.llmConsent.description")}
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
							{t("onboarding.llmConsent.skipCard.title")}
						</Title>
					}
					data-testid="llm-consent-skip-card"
				>
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						<Text type="secondary">{t("onboarding.llmConsent.skipCard.description")}</Text>
						<Button
							onClick={handleSkip}
							loading={pendingChoice === "skip"}
							disabled={busy && pendingChoice !== "skip"}
							data-testid="llm-consent-skip-button"
						>
							{t("onboarding.llmConsent.skipCard.button")}
						</Button>
					</Space>
				</Card>

				<Card
					title={
						<Title level={5} style={{ margin: 0 }}>
							{t("onboarding.llmConsent.tryCard.title")}
						</Title>
					}
					data-testid="llm-consent-try-card"
				>
					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						{systemDefaultAvailable ? (
							<Alert
								type="info"
								showIcon
								message={t("onboarding.llmConsent.tryCard.systemAvailable.message")}
								description={t("onboarding.llmConsent.tryCard.systemAvailable.description")}
							/>
						) : (
							<Alert
								type="warning"
								showIcon
								message={t("onboarding.llmConsent.tryCard.systemUnavailable.message")}
								description={t("onboarding.llmConsent.tryCard.systemUnavailable.description")}
							/>
						)}
						<Button
							type="primary"
							onClick={handleTryAI}
							loading={pendingChoice === "try"}
							disabled={busy && pendingChoice !== "try"}
							data-testid="llm-consent-try-button"
						>
							{systemDefaultAvailable
								? t("onboarding.llmConsent.tryCard.buttonConsent")
								: t("onboarding.llmConsent.tryCard.buttonConfigure")}
						</Button>
					</Space>
				</Card>
			</Space>
		</OnboardingLayout>
	);
}
