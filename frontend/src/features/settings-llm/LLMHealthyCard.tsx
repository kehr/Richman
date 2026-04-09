import { App, Button, Card, Popconfirm, Space, Switch, Tag, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { LLMProbeButton } from "./LLMProbeButton";
import { useDeleteLLMSettings, useUpsertLLMSettings } from "./hooks";
import type { LLMSettingsDTO } from "./types";

const { Text, Title } = Typography;

interface LLMHealthyCardProps {
	config: LLMSettingsDTO;
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

// LLMHealthyCard is the "configured + healthy" variant. It shows the
// provider brand, model, masked key hint, health tag, fallback toggle and
// the three standard action buttons.
export function LLMHealthyCard({ config, onEdit }: LLMHealthyCardProps) {
	const { t } = useTranslation("settings");
	const { message } = App.useApp();
	const upsertMutation = useUpsertLLMSettings();
	const deleteMutation = useDeleteLLMSettings();

	const handleToggleFallback = async (checked: boolean) => {
		// The backend's PUT handler requires providerType + model even on a
		// toggle-only change. We reuse the stored values and leave apiKey
		// absent so the server keeps the existing encrypted key.
		if (!config.providerType || !config.model) return;
		try {
			await upsertMutation.mutateAsync({
				providerType: config.providerType,
				baseUrl: config.baseUrl ?? undefined,
				model: config.model,
				fallbackToSystemDefaultOnFailure: checked,
				probe: false,
			});
			message.success(t("llm.healthyCard.fallbackUpdated"));
		} catch (err) {
			const msg = err instanceof Error ? err.message : t("llm.healthyCard.fallbackUpdateError");
			message.error(msg);
		}
	};

	const handleDelete = async () => {
		try {
			await deleteMutation.mutateAsync();
			message.success(t("llm.healthyCard.deleteSuccess"));
		} catch (err) {
			const msg = err instanceof Error ? err.message : t("llm.healthyCard.deleteError");
			message.error(msg);
		}
	};

	const probeTime = config.lastProbeAt
		? new Date(config.lastProbeAt).toLocaleString()
		: t("llm.healthyCard.notTested");

	return (
		<Card data-testid="llm-healthy-card">
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Space align="center" wrap>
					<Title level={5} style={{ margin: 0 }}>
						{providerLabel(config.providerType)}
					</Title>
					<Tag color="success" data-testid="llm-health-tag">
						{t("llm.healthyCard.healthy")}
					</Tag>
				</Space>

				<Space direction="vertical" size={4}>
					<Text>
						{t("llm.healthyCard.modelLabel")}: <Text code>{config.model ?? "-"}</Text>
					</Text>
					<Text>
						{t("llm.healthyCard.apiKeyLabel")}: <Text code>{config.apiKeyHint ?? "..****"}</Text>
					</Text>
					{config.baseUrl && (
						<Text>
							{t("llm.healthyCard.baseUrlLabel")}: <Text code>{config.baseUrl}</Text>
						</Text>
					)}
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t("llm.healthyCard.lastProbed")}: {probeTime}
					</Text>
				</Space>

				<Space align="center">
					<Switch
						checked={config.fallbackToSystemDefaultOnFailure}
						loading={upsertMutation.isPending}
						onChange={handleToggleFallback}
						data-testid="llm-fallback-switch"
					/>
					<Text>{t("llm.healthyCard.fallbackToggle")}</Text>
				</Space>
				<Text type="secondary" style={{ fontSize: 12 }}>
					{t("llm.healthyCard.fallbackHint")}
				</Text>

				<Space wrap>
					<LLMProbeButton />
					<Button onClick={onEdit} data-testid="llm-edit-button">
						{t("llm.healthyCard.editButton")}
					</Button>
					<Popconfirm
						title={t("llm.healthyCard.deleteConfirm.title")}
						description={t("llm.healthyCard.deleteConfirm.description")}
						okText={t("llm.healthyCard.deleteConfirm.ok")}
						cancelText={t("llm.healthyCard.deleteConfirm.cancel")}
						okButtonProps={{ danger: true }}
						onConfirm={handleDelete}
					>
						<Button danger loading={deleteMutation.isPending} data-testid="llm-delete-button">
							{t("llm.healthyCard.deleteButton")}
						</Button>
					</Popconfirm>
				</Space>
			</Space>
		</Card>
	);
}
