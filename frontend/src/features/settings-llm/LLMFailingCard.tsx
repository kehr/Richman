import { Alert, App, Button, Card, Popconfirm, Space, Tag, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { LLMProbeButton } from "./LLMProbeButton";
import { useDeleteLLMSettings } from "./hooks";
import type { LLMSettingsDTO } from "./types";

const { Text, Title } = Typography;

interface LLMFailingCardProps {
	config: LLMSettingsDTO;
	systemDefaultAvailable: boolean;
	onEdit: () => void;
}

// providerLabel maps the provider enum to a human-readable brand label.
// These are proper nouns / brand names and intentionally not translated.
function providerLabel(providerType: LLMSettingsDTO["providerType"]): string {
	switch (providerType) {
		case "claude":
			return "Claude (Anthropic)";
		case "openai":
			return "OpenAI";
		case "openai_compatible":
			return "OpenAI Compatible";
		default:
			return "Unknown";
	}
}

// LLMFailingCard is the "configured + failing" variant. It surfaces the
// probe error message and tells the user which fallback layer will be used
// in the meantime, then offers the three standard action buttons.
export function LLMFailingCard({ config, systemDefaultAvailable, onEdit }: LLMFailingCardProps) {
	const { t } = useTranslation("settings");
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

	const probeTime = config.lastProbeAt
		? new Date(config.lastProbeAt).toLocaleString()
		: t("llm.failingCard.unknown");

	const fallbackCopy = (() => {
		if (config.fallbackToSystemDefaultOnFailure && systemDefaultAvailable) {
			return t("llm.failingCard.fallback.systemAvailable");
		}
		if (config.fallbackToSystemDefaultOnFailure && !systemDefaultAvailable) {
			return t("llm.failingCard.fallback.systemUnavailable");
		}
		return t("llm.failingCard.fallback.noFallback");
	})();

	return (
		<Card data-testid="llm-failing-card">
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Space align="center" wrap>
					<Title level={5} style={{ margin: 0 }}>
						{providerLabel(config.providerType)}
					</Title>
					<Tag color="error" data-testid="llm-health-tag">
						{t("llm.failingCard.failing")}
					</Tag>
				</Space>

				<Alert
					type="error"
					showIcon
					message={t("llm.failingCard.connectivityFailed")}
					description={config.lastProbeError ?? t("llm.failingCard.unknown")}
					data-testid="llm-failing-alert"
				/>

				<Space direction="vertical" size={4}>
					<Text>
						{t("llm.failingCard.modelLabel")}: <Text code>{config.model ?? "-"}</Text>
					</Text>
					<Text>
						{t("llm.failingCard.apiKeyLabel")}: <Text code>{config.apiKeyHint ?? "..****"}</Text>
					</Text>
					{config.baseUrl && (
						<Text>
							{t("llm.failingCard.baseUrlLabel")}: <Text code>{config.baseUrl}</Text>
						</Text>
					)}
					<Text type="secondary" style={{ fontSize: 12 }}>
						{probeTime}
					</Text>
				</Space>

				<Text type="secondary">{fallbackCopy}</Text>

				<Space wrap>
					<LLMProbeButton label={t("llm.failingCard.retestButton")} />
					<Button onClick={onEdit} data-testid="llm-edit-button">
						{t("llm.failingCard.editButton")}
					</Button>
					<Popconfirm
						title={t("llm.failingCard.deleteConfirm.title")}
						description={t("llm.failingCard.deleteConfirm.description")}
						okText={t("llm.failingCard.deleteConfirm.ok")}
						cancelText={t("llm.healthyCard.deleteConfirm.cancel")}
						okButtonProps={{ danger: true }}
						onConfirm={handleDelete}
					>
						<Button danger loading={deleteMutation.isPending} data-testid="llm-delete-button">
							{t("llm.failingCard.deleteButton")}
						</Button>
					</Popconfirm>
				</Space>
			</Space>
		</Card>
	);
}
