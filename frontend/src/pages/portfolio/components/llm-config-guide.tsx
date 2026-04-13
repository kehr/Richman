import { Alert, Button } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

interface LLMConfigGuideProps {
	onDismiss?: () => void;
}

// LLMConfigGuide is shown when the user has added their first holding but has
// no LLM configuration set up yet (TRD SS7.5).
// It prompts the user to configure an LLM provider to unlock AI-powered analysis.
export function LLMConfigGuide({ onDismiss }: LLMConfigGuideProps) {
	const { t } = useTranslation("app");
	const navigate = useNavigate();

	return (
		<Alert
			type="info"
			showIcon
			closable={!!onDismiss}
			onClose={onDismiss}
			message={t("portfolio.llmConfigGuide.title")}
			description={t("portfolio.llmConfigGuide.description")}
			action={
				<Button size="small" type="primary" onClick={() => navigate("/settings")}>
					{t("portfolio.llmConfigGuide.configureButton")}
				</Button>
			}
			style={{ marginBottom: 12 }}
		/>
	);
}
