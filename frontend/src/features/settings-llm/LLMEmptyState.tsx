import { Button, Card, Empty, Space, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text, Title } = Typography;

interface LLMEmptyStateProps {
	// systemDefaultAvailable controls the supporting callout: when true we
	// tell the user analyses will fall back to Richman's shared provider
	// (assuming consent); when false we tell them analyses will use the
	// rules engine.
	systemDefaultAvailable: boolean;
	useSystemDefaultConsent: boolean;
	onAddProvider: () => void;
}

// LLMEmptyState is the "not configured" variant of the settings LLM
// section. It presents a single primary CTA and a callout explaining
// which fallback layer will be used in the meantime.
export function LLMEmptyState({
	systemDefaultAvailable,
	useSystemDefaultConsent,
	onAddProvider,
}: LLMEmptyStateProps) {
	const { t } = useTranslation("settings");

	const calloutCopy = (() => {
		if (systemDefaultAvailable && useSystemDefaultConsent) {
			return t("llm.emptyState.callout.systemConsentGiven");
		}
		if (systemDefaultAvailable && !useSystemDefaultConsent) {
			return t("llm.emptyState.callout.systemNoConsent");
		}
		return t("llm.emptyState.callout.noSystem");
	})();

	return (
		<Card data-testid="llm-empty-state">
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Empty
					description={
						<Space direction="vertical" size={4}>
							<Title level={5} style={{ margin: 0 }}>
								{t("llm.emptyState.title")}
							</Title>
							<Text type="secondary">{calloutCopy}</Text>
						</Space>
					}
				>
					<Button type="primary" onClick={onAddProvider} data-testid="llm-add-provider-button">
						{t("llm.emptyState.addButton")}
					</Button>
				</Empty>
			</Space>
		</Card>
	);
}
