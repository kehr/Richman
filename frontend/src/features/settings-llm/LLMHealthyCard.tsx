import { App, Button, Divider, Space, Switch, theme } from "@/ui-kit/eat";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { LLMProbeButton } from "./LLMProbeButton";
import { ProviderCardLayout } from "./ProviderCardLayout";
import { useDeleteLLMSettings, useUpsertLLMSettings } from "./hooks";
import type { LLMSettingsDTO } from "./types";

interface LLMHealthyCardProps {
	config: LLMSettingsDTO;
	onEdit: () => void;
}

// LLMHealthyCard is the "configured + healthy" variant. It delegates layout
// to ProviderCardLayout and provides body/footer slot content.
export function LLMHealthyCard({ config, onEdit }: LLMHealthyCardProps) {
	const { t } = useTranslation("settings");
	const { token } = theme.useToken();
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

	const fallbackText = config.fallbackToSystemDefaultOnFailure
		? t("llm.healthyCard.fallbackOn")
		: t("llm.healthyCard.fallbackOff");

	// Info grid items: filter out baseUrl row if empty
	const infoItems: { label: string; value: string; muted?: boolean }[] = [
		{ label: t("llm.healthyCard.modelLabel"), value: config.model ?? "-" },
		{ label: t("llm.healthyCard.apiKeyLabel"), value: config.apiKeyHint ?? "..****", muted: true },
		...(config.baseUrl
			? [{ label: t("llm.healthyCard.baseUrlLabel"), value: config.baseUrl }]
			: []),
		{
			label: t("llm.healthyCard.fallbackLabel"),
			value: fallbackText,
			muted: true,
		},
	];

	const bodyContent: ReactNode = (
		<>
			<div
				style={{
					display: "grid",
					gridTemplateColumns: "1fr 1fr",
					gap: "12px 24px",
					marginBottom: 14,
				}}
			>
				{infoItems.map(({ label, value, muted }) => (
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
						<span
							style={{
								fontSize: 13,
								color: muted ? token.colorTextTertiary : token.colorText,
							}}
						>
							{value}
						</span>
					</div>
				))}
			</div>
			<Divider style={{ margin: "14px 0" }} />
			<div
				style={{
					display: "flex",
					alignItems: "flex-start",
					justifyContent: "space-between",
					gap: 12,
				}}
			>
				<div>
					<div style={{ fontSize: 13, color: token.colorText }}>
						{t("llm.healthyCard.fallbackToggle")}
					</div>
					<div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 3 }}>
						{t("llm.healthyCard.fallbackHint")}
					</div>
				</div>
				<Switch
					checked={config.fallbackToSystemDefaultOnFailure}
					loading={upsertMutation.isPending}
					onChange={handleToggleFallback}
					data-testid="llm-fallback-switch"
				/>
			</div>
		</>
	);

	const footerContent: ReactNode = (
		<Space>
			<LLMProbeButton />
			<Button onClick={onEdit} data-testid="llm-edit-button">
				{t("llm.healthyCard.editButton")}
			</Button>
		</Space>
	);

	return (
		<ProviderCardLayout
			providerType={config.providerType}
			lastProbeAt={config.lastProbeAt}
			healthStatus="healthy"
			onDelete={handleDelete}
			isDeleting={deleteMutation.isPending}
			bodyContent={bodyContent}
			footerContent={footerContent}
			data-testid="llm-healthy-card"
		/>
	);
}
