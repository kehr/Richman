import { Alert, App, Button, Divider, Typography, theme } from "@/ui-kit/eat";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { LLMProbeButton } from "./LLMProbeButton";
import { ProviderCardLayout } from "./ProviderCardLayout";
import { useDeleteLLMSettings } from "./hooks";
import type { LLMSettingsDTO } from "./types";

const { Text } = Typography;

interface LLMFailingCardProps {
	config: LLMSettingsDTO;
	systemDefaultAvailable: boolean;
	onEdit: () => void;
}

// LLMFailingCard is the "configured + failing" variant. It surfaces the
// probe error message and tells the user which fallback layer will be used
// in the meantime, then offers the three standard action buttons.
export function LLMFailingCard({ config, systemDefaultAvailable, onEdit }: LLMFailingCardProps) {
	const { t } = useTranslation("settings");
	const { token } = theme.useToken();
	const { message } = App.useApp();
	const deleteMutation = useDeleteLLMSettings();

	const handleDelete = async () => {
		try {
			await deleteMutation.mutateAsync();
			message.success(t("llm.failingCard.deleteSuccess"));
		} catch (err) {
			const msg = err instanceof Error ? err.message : t("llm.healthyCard.deleteError");
			message.error(msg);
		}
	};

	const fallbackCopy = (() => {
		if (config.fallbackToSystemDefaultOnFailure && systemDefaultAvailable) {
			return t("llm.failingCard.fallback.systemAvailable");
		}
		if (config.fallbackToSystemDefaultOnFailure && !systemDefaultAvailable) {
			return t("llm.failingCard.fallback.systemUnavailable");
		}
		return t("llm.failingCard.fallback.noFallback");
	})();

	// Info grid items: only model and baseUrl (no API key for failing state)
	const infoItems: { label: string; value: string }[] = [
		{ label: t("llm.failingCard.modelLabel"), value: config.model ?? "-" },
		...(config.baseUrl
			? [{ label: t("llm.failingCard.baseUrlLabel"), value: config.baseUrl }]
			: []),
	];

	const bodyContent: ReactNode = (
		<>
			<Alert
				type="error"
				showIcon
				message={t("llm.failingCard.connectivityFailed")}
				description={config.lastProbeError ?? t("llm.failingCard.unknown")}
				data-testid="llm-failing-alert"
				style={{ marginBottom: 14 }}
			/>
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "1fr 1fr",
					gap: "12px 24px",
					marginBottom: 14,
				}}
			>
				{infoItems.map(({ label, value }) => (
					<div key={label} style={{ display: "flex", flexDirection: "column", gap: 2 }}>
						<span
							style={{
								fontSize: 11,
								color: token.colorTextQuaternary,
								textTransform: "uppercase",
								letterSpacing: "0.4px",
							}}
						>
							{label}
						</span>
						<span style={{ fontSize: 13, color: token.colorText }}>{value}</span>
					</div>
				))}
			</div>
			<Divider style={{ margin: "14px 0" }} />
			<Text type="secondary" style={{ fontSize: 12 }}>
				{fallbackCopy}
			</Text>
		</>
	);

	const footerContent: ReactNode = (
		<>
			<LLMProbeButton label={t("llm.failingCard.retestButton")} />
			<Button onClick={onEdit} data-testid="llm-edit-button">
				{t("llm.failingCard.editButton")}
			</Button>
		</>
	);

	return (
		<ProviderCardLayout
			providerType={config.providerType}
			lastProbeAt={config.lastProbeAt}
			healthStatus="failing"
			onDelete={handleDelete}
			isDeleting={deleteMutation.isPending}
			bodyContent={bodyContent}
			footerContent={footerContent}
			data-testid="llm-failing-card"
		/>
	);
}
