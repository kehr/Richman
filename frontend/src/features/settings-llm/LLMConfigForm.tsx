import { App, Button, Form, Input, Modal, Select, Space, Switch, Typography } from "@/ui-kit/eat";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useUpsertLLMSettings } from "./hooks";
import type { LLMProviderType, LLMSettingsDTO, UpsertLLMRequest } from "./types";

const { Text } = Typography;

interface LLMConfigFormProps {
	open: boolean;
	// mode is "create" when there is no existing row (apiKey required),
	// "edit" when editing an existing row (apiKey optional — empty means
	// leave the stored key unchanged).
	mode: "create" | "edit";
	initialValue?: LLMSettingsDTO;
	onClose: () => void;
	onSaved?: () => void;
}

interface FormValues {
	providerType: LLMProviderType;
	baseUrl?: string;
	apiKey?: string;
	model: string;
	fallbackToSystemDefaultOnFailure: boolean;
}

// LLMConfigForm is the modal dialog behind the "add LLM Provider" CTA
// (create) and the "edit" CTA on the Healthy / Failing cards (edit). It
// owns form validation, client-side SSRF rejection (http baseUrl), and
// calls useUpsertLLMSettings on submit. On success the modal closes and
// the caller's onSaved is invoked so the parent can surface a toast from
// the latest probe result.
export function LLMConfigForm({ open, mode, initialValue, onClose, onSaved }: LLMConfigFormProps) {
	const { t } = useTranslation("settings");
	const [form] = Form.useForm<FormValues>();
	const { message } = App.useApp();
	const upsertMutation = useUpsertLLMSettings();

	const providerType = Form.useWatch("providerType", form);
	const requiresBaseUrl = providerType === "openai_compatible";
	const apiKeyOptional = providerType === "openai_compatible";

	// Reset the form whenever the modal opens or the initial value changes.
	// This keeps stale values from the previous edit from leaking into a
	// subsequent "add provider" click.
	useEffect(() => {
		if (!open) return;
		form.resetFields();
		if (initialValue?.configured) {
			form.setFieldsValue({
				providerType: initialValue.providerType ?? "claude",
				baseUrl: initialValue.baseUrl ?? undefined,
				apiKey: undefined,
				model: initialValue.model ?? "",
				fallbackToSystemDefaultOnFailure: initialValue.fallbackToSystemDefaultOnFailure,
			});
		} else {
			form.setFieldsValue({
				providerType: "claude",
				baseUrl: undefined,
				apiKey: undefined,
				model: "",
				fallbackToSystemDefaultOnFailure: false,
			});
		}
	}, [open, initialValue, form]);

	// Memoize validation rules so they stay reactive to locale changes.
	const rules = useMemo(
		() => ({
			providerType: [{ required: true, message: t("llm.configForm.providerTypeRequired") }],
			baseUrl: [
				{ required: true, message: t("llm.configForm.baseUrlRequired") },
				{
					validator: (_rule: unknown, value: string) => {
						if (typeof value !== "string" || value.length === 0) return Promise.resolve();
						if (value.startsWith("https://") || value.startsWith("http://"))
							return Promise.resolve();
						return Promise.reject(new Error(t("llm.configForm.baseUrlProtocolRequired")));
					},
				},
			],
			apiKeyCreate: apiKeyOptional
				? []
				: [{ required: true, message: t("llm.configForm.apiKeyRequired") }],
			model: [{ required: true, message: t("llm.configForm.modelRequired") }],
		}),
		[t, apiKeyOptional],
	);

	const providerOptions = useMemo(
		() => [
			{ label: t("llm.configForm.providerOptions.claude"), value: "claude" as LLMProviderType },
			{ label: t("llm.configForm.providerOptions.openai"), value: "openai" as LLMProviderType },
			{
				label: t("llm.configForm.providerOptions.openai_compatible"),
				value: "openai_compatible" as LLMProviderType,
			},
		],
		[t],
	);

	const handleOk = async () => {
		let values: FormValues;
		try {
			values = await form.validateFields();
		} catch {
			return;
		}

		const body: UpsertLLMRequest = {
			providerType: values.providerType,
			model: values.model.trim(),
			fallbackToSystemDefaultOnFailure: values.fallbackToSystemDefaultOnFailure,
			probe: true,
		};
		if (values.baseUrl && values.baseUrl.trim().length > 0) {
			body.baseUrl = values.baseUrl.trim();
		}
		if (values.apiKey && values.apiKey.length > 0) {
			body.apiKey = values.apiKey;
		}

		try {
			const result = await upsertMutation.mutateAsync(body);
			if (result.healthStatus === "healthy") {
				message.success(t("llm.configForm.message.savedHealthy"));
			} else if (result.healthStatus === "failing") {
				message.warning(
					t("llm.configForm.message.savedFailing", {
						error: result.lastProbeError ?? t("llm.failingCard.unknown"),
					}),
				);
			} else {
				message.success(t("llm.configForm.message.saved"));
			}
			onSaved?.();
			onClose();
		} catch (err) {
			const msg = err instanceof Error ? err.message : t("llm.configForm.message.saveError");
			message.error(msg);
		}
	};

	return (
		<Modal
			open={open}
			title={mode === "create" ? t("llm.configForm.addTitle") : t("llm.configForm.editTitle")}
			onCancel={onClose}
			onOk={handleOk}
			confirmLoading={upsertMutation.isPending}
			okText={t("llm.configForm.saveAndTest")}
			cancelText={t("llm.configForm.cancel")}
			destroyOnClose
			data-testid="llm-config-form-modal"
		>
			<Form<FormValues> form={form} layout="vertical" requiredMark data-testid="llm-config-form">
				<Form.Item<FormValues>
					name="providerType"
					label={t("llm.configForm.providerType")}
					rules={rules.providerType}
				>
					<Select<LLMProviderType>
						options={providerOptions}
						data-testid="llm-config-provider-type"
					/>
				</Form.Item>

				{requiresBaseUrl && (
					<Form.Item<FormValues>
						name="baseUrl"
						label={t("llm.configForm.baseUrl")}
						rules={rules.baseUrl}
					>
						<Input placeholder="https://example.com/v1" data-testid="llm-config-base-url" />
					</Form.Item>
				)}

				<Form.Item<FormValues>
					name="apiKey"
					label={mode === "create" ? t("llm.configForm.apiKey") : t("llm.configForm.apiKeyEdit")}
					rules={mode === "create" ? rules.apiKeyCreate : []}
					extra={apiKeyOptional ? t("llm.configForm.apiKeyOptionalHint") : undefined}
				>
					<Input.Password
						placeholder={
							mode === "create"
								? apiKeyOptional
									? t("llm.configForm.apiKeyOptionalPlaceholder")
									: t("llm.configForm.apiKeyPlaceholder")
								: t("llm.configForm.apiKeyEditPlaceholder")
						}
						autoComplete="off"
						data-testid="llm-config-api-key"
					/>
				</Form.Item>

				<Form.Item<FormValues> name="model" label={t("llm.configForm.model")} rules={rules.model}>
					<Input
						placeholder={t("llm.configForm.modelPlaceholder")}
						data-testid="llm-config-model"
					/>
				</Form.Item>

				<Form.Item<FormValues>
					name="fallbackToSystemDefaultOnFailure"
					label={t("llm.configForm.fallback")}
					valuePropName="checked"
				>
					<Space direction="vertical" size={4} style={{ width: "100%" }}>
						<Switch data-testid="llm-config-fallback-switch" />
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("llm.configForm.fallbackHint")}
						</Text>
					</Space>
				</Form.Item>

				<Form.Item label={null}>
					<Button type="link" onClick={onClose} style={{ padding: 0 }}>
						{t("llm.configForm.backLink")}
					</Button>
				</Form.Item>
			</Form>
		</Modal>
	);
}
