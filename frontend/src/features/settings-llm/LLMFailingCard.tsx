import { Alert, App, Button, Card, Popconfirm, Space, Tag, Typography } from "@/ui-kit/eat";
import { LLMProbeButton } from "./LLMProbeButton";
import { useDeleteLLMSettings } from "./hooks";
import type { LLMSettingsDTO } from "./types";

const { Text, Title } = Typography;

interface LLMFailingCardProps {
	config: LLMSettingsDTO;
	systemDefaultAvailable: boolean;
	onEdit: () => void;
}

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

// LLMFailingCard is the "configured + failing" variant. It surfaces the
// probe error message and tells the user which fallback layer will be used
// in the meantime, then offers the three standard action buttons.
export function LLMFailingCard({ config, systemDefaultAvailable, onEdit }: LLMFailingCardProps) {
	const { message } = App.useApp();
	const deleteMutation = useDeleteLLMSettings();

	const handleDelete = async () => {
		try {
			await deleteMutation.mutateAsync();
			message.success("已删除 Provider 配置");
		} catch (err) {
			const msg = err instanceof Error ? err.message : "删除失败";
			message.error(msg);
		}
	};

	const probeTime = config.lastProbeAt ? new Date(config.lastProbeAt).toLocaleString() : "未知";

	const fallbackCopy = (() => {
		if (config.fallbackToSystemDefaultOnFailure && systemDefaultAvailable) {
			return "你已开启失败降级：调用失败时分析会自动走 Richman 的系统默认 AI Provider。";
		}
		if (config.fallbackToSystemDefaultOnFailure && !systemDefaultAvailable) {
			return "你已开启失败降级，但系统默认 Provider 当前不可用。分析将降级到规则引擎。";
		}
		return "你未开启失败降级，分析会直接使用规则引擎。";
	})();

	return (
		<Card data-testid="llm-failing-card">
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Space align="center" wrap>
					<Title level={5} style={{ margin: 0 }}>
						{providerLabel(config.providerType)}
					</Title>
					<Tag color="error" data-testid="llm-health-tag">
						失效
					</Tag>
				</Space>

				<Alert
					type="error"
					showIcon
					message="LLM Provider 连通性测试失败"
					description={config.lastProbeError ?? "未知错误"}
					data-testid="llm-failing-alert"
				/>

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

				<Text type="secondary">{fallbackCopy}</Text>

				<Space wrap>
					<LLMProbeButton label="重新测试" />
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
