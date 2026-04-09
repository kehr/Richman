import { Button, Card, Empty, Space, Typography } from "@/ui-kit/eat";

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
	const calloutCopy = (() => {
		if (systemDefaultAvailable && useSystemDefaultConsent) {
			return "当前分析将走 Richman 的系统默认 AI Provider。你可以配置自己的 Provider 以获得更好的额度和隐私控制。";
		}
		if (systemDefaultAvailable && !useSystemDefaultConsent) {
			return "系统默认 AI Provider 可用，但你尚未同意使用。当前分析会降级到规则引擎。配置你自己的 Provider 或在 onboarding 中勾选同意可以启用 AI 解读。";
		}
		return "当前分析走规则引擎。配置你自己的 LLM Provider 可以启用 AI 解读能力。";
	})();

	return (
		<Card data-testid="llm-empty-state">
			<Space direction="vertical" size={16} style={{ width: "100%" }}>
				<Empty
					description={
						<Space direction="vertical" size={4}>
							<Title level={5} style={{ margin: 0 }}>
								尚未配置 AI Provider
							</Title>
							<Text type="secondary">{calloutCopy}</Text>
						</Space>
					}
				>
					<Button type="primary" onClick={onAddProvider} data-testid="llm-add-provider-button">
						添加 LLM Provider
					</Button>
				</Empty>
			</Space>
		</Card>
	);
}
