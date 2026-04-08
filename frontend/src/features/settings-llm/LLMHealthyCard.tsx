import { App, Button, Card, Popconfirm, Space, Switch, Tag, Typography } from "@/ui-kit/eat";
import { LLMProbeButton } from "./LLMProbeButton";
import { useDeleteLLMSettings, useUpsertLLMSettings } from "./hooks";
import type { LLMSettingsDTO } from "./types";

const { Text, Title } = Typography;

interface LLMHealthyCardProps {
	config: LLMSettingsDTO;
	onEdit: () => void;
}

// providerLabel maps the provider enum to a human-readable brand label.
function providerLabel(providerType: LLMSettingsDTO["providerType"]): string {
	switch (providerType) {
		case "claude":
			return "Claude (Anthropic)";
		case "openai":
			return "OpenAI";
		case "openai_compatible":
			return "OpenAI 兼容";
		default:
			return "Unknown";
	}
}

// LLMHealthyCard is the "configured + healthy" variant. It shows the
// provider brand, model, masked key hint, health tag, fallback toggle and
// the three standard action buttons.
export function LLMHealthyCard({ config, onEdit }: LLMHealthyCardProps) {
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
			message.success("已更新降级偏好");
		} catch (err) {
			const msg = err instanceof Error ? err.message : "更新失败";
			message.error(msg);
		}
	};

	const handleDelete = async () => {
		try {
			await deleteMutation.mutateAsync();
			message.success("已删除 Provider 配置");
		} catch (err) {
			const msg = err instanceof Error ? err.message : "删除失败";
			message.error(msg);
		}
	};

	const probeTime = config.lastProbeAt ? new Date(config.lastProbeAt).toLocaleString() : "尚未测试";

	return (
		<Card data-testid="llm-healthy-card">
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Space align="center" wrap>
					<Title level={5} style={{ margin: 0 }}>
						{providerLabel(config.providerType)}
					</Title>
					<Tag color="success" data-testid="llm-health-tag">
						健康
					</Tag>
				</Space>

				<Space direction="vertical" size={4}>
					<Text>
						模型: <Text code>{config.model ?? "-"}</Text>
					</Text>
					<Text>
						API Key: <Text code>{config.apiKeyHint ?? "..****"}</Text>
					</Text>
					{config.baseUrl && (
						<Text>
							Base URL: <Text code>{config.baseUrl}</Text>
						</Text>
					)}
					<Text type="secondary" style={{ fontSize: 12 }}>
						最后测试: {probeTime}
					</Text>
				</Space>

				<Space align="center">
					<Switch
						checked={config.fallbackToSystemDefaultOnFailure}
						loading={upsertMutation.isPending}
						onChange={handleToggleFallback}
						data-testid="llm-fallback-switch"
					/>
					<Text>调用失败时自动降级到系统默认</Text>
				</Space>
				<Text type="secondary" style={{ fontSize: 12 }}>
					开启后，当你的 Provider 失败时，持仓数据将以加密传输方式发给 Richman 的系统默认 AI
					Provider 做分析。关闭则直接降级到规则引擎。
				</Text>

				<Space wrap>
					<LLMProbeButton />
					<Button onClick={onEdit} data-testid="llm-edit-button">
						编辑
					</Button>
					<Popconfirm
						title="确认删除 LLM Provider 配置？"
						description="删除后分析将降级到系统默认或规则引擎。"
						okText="删除"
						cancelText="取消"
						okButtonProps={{ danger: true }}
						onConfirm={handleDelete}
					>
						<Button danger loading={deleteMutation.isPending} data-testid="llm-delete-button">
							删除
						</Button>
					</Popconfirm>
				</Space>
			</Space>
		</Card>
	);
}
