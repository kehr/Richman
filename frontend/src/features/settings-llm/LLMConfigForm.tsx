import { App, Button, Form, Input, Modal, Select, Space, Switch, Typography } from "@/ui-kit/eat";
import { useEffect } from "react";
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

// LLMConfigForm is the modal dialog behind the "添加 LLM Provider" CTA
// (create) and the "编辑" CTA on the Healthy / Failing cards (edit). It
// owns form validation, client-side SSRF rejection (http baseUrl), and
// calls useUpsertLLMSettings on submit. On success the modal closes and
// the caller's onSaved is invoked so the parent can surface a toast from
// the latest probe result.
export function LLMConfigForm({ open, mode, initialValue, onClose, onSaved }: LLMConfigFormProps) {
	const [form] = Form.useForm<FormValues>();
	const { message } = App.useApp();
	const upsertMutation = useUpsertLLMSettings();

	const providerType = Form.useWatch("providerType", form);
	const requiresBaseUrl = providerType === "openai_compatible";

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
				message.success("已保存，连通性测试通过");
			} else if (result.healthStatus === "failing") {
				message.warning(`已保存，但连通性测试失败：${result.lastProbeError ?? "未知错误"}`);
			} else {
				message.success("已保存");
			}
			onSaved?.();
			onClose();
		} catch (err) {
			const msg = err instanceof Error ? err.message : "保存失败";
			message.error(msg);
		}
	};

	return (
		<Modal
			open={open}
			title={mode === "create" ? "添加 LLM Provider" : "编辑 LLM Provider"}
			onCancel={onClose}
			onOk={handleOk}
			confirmLoading={upsertMutation.isPending}
			okText="保存并测试"
			cancelText="取消"
			destroyOnClose
			data-testid="llm-config-form-modal"
		>
			<Form<FormValues> form={form} layout="vertical" requiredMark data-testid="llm-config-form">
				<Form.Item<FormValues>
					name="providerType"
					label="Provider 类型"
					rules={[{ required: true, message: "请选择 Provider 类型" }]}
				>
					<Select<LLMProviderType>
						options={[
							{ label: "Claude (Anthropic)", value: "claude" },
							{ label: "OpenAI", value: "openai" },
							{ label: "OpenAI 兼容 (Ollama / 自建)", value: "openai_compatible" },
						]}
						data-testid="llm-config-provider-type"
					/>
				</Form.Item>

				{requiresBaseUrl && (
					<Form.Item<FormValues>
						name="baseUrl"
						label="Base URL"
						rules={[
							{ required: true, message: "请填写 Base URL" },
							{
								validator: (_rule, value) => {
									if (typeof value !== "string" || value.length === 0) return Promise.resolve();
									if (value.startsWith("https://")) return Promise.resolve();
									return Promise.reject(new Error("Base URL 必须以 https:// 开头"));
								},
							},
						]}
					>
						<Input placeholder="https://example.com/v1" data-testid="llm-config-base-url" />
					</Form.Item>
				)}

				<Form.Item<FormValues>
					name="apiKey"
					label={mode === "create" ? "API Key" : "API Key (留空表示不修改)"}
					rules={mode === "create" ? [{ required: true, message: "请填写 API Key" }] : []}
				>
					<Input.Password
						placeholder={mode === "create" ? "sk-..." : "留空表示不修改"}
						autoComplete="off"
						data-testid="llm-config-api-key"
					/>
				</Form.Item>

				<Form.Item<FormValues>
					name="model"
					label="模型"
					rules={[{ required: true, message: "请填写模型名" }]}
				>
					<Input
						placeholder="例如 claude-sonnet-4-6 / gpt-4o-mini"
						data-testid="llm-config-model"
					/>
				</Form.Item>

				<Form.Item<FormValues>
					name="fallbackToSystemDefaultOnFailure"
					label="调用失败时自动降级到系统默认"
					valuePropName="checked"
				>
					<Space direction="vertical" size={4} style={{ width: "100%" }}>
						<Switch data-testid="llm-config-fallback-switch" />
						<Text type="secondary" style={{ fontSize: 12 }}>
							开启后，当你的 Provider
							失败（密钥过期、配额超限、网络错误）时，你的持仓数据将以加密传输方式发给 Richman
							的系统默认 AI Provider 做分析。关闭则直接降级为规则引擎。
						</Text>
					</Space>
				</Form.Item>

				<Form.Item label={null}>
					<Button type="link" onClick={onClose} style={{ padding: 0 }}>
						返回
					</Button>
				</Form.Item>
			</Form>
		</Modal>
	);
}
