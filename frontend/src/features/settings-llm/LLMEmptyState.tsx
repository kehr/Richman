import { Alert, Button, Card, Typography, theme } from "@/ui-kit/eat";
import { Bot } from "lucide-react";
import { useTranslation } from "react-i18next";

const { Text, Title } = Typography;

interface LLMEmptyStateProps {
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
	const { token } = theme.useToken();

	const calloutNode = (() => {
		if (!systemDefaultAvailable) return null;
		if (useSystemDefaultConsent) {
			return (
				<Alert
					type="info"
					showIcon
					message={t("llm.emptyState.callout.systemConsentGiven")}
					style={{ borderTop: "none", borderRadius: "0 0 8px 8px" }}
				/>
			);
		}
		return (
			<Alert
				type="warning"
				showIcon
				message={t("llm.emptyState.callout.systemNoConsent")}
				style={{ borderTop: "none", borderRadius: "0 0 8px 8px" }}
			/>
		);
	})();

	return (
		<Card data-testid="llm-empty-state">
			<div style={{ textAlign: "center", padding: "24px 20px" }}>
				<Bot size={32} color={token.colorTextQuaternary} style={{ marginBottom: 12 }} />
				<Title level={5} style={{ margin: "0 0 6px" }}>
					{t("llm.emptyState.title")}
				</Title>
				<Text
					type="secondary"
					style={{ display: "block", maxWidth: 360, margin: "0 auto 16px", fontSize: 13 }}
				>
					{t("llm.emptyState.description")}
				</Text>
				<Button type="primary" onClick={onAddProvider} data-testid="llm-add-provider-button">
					{t("llm.emptyState.addButton")}
				</Button>
			</div>
			{calloutNode}
		</Card>
	);
}
