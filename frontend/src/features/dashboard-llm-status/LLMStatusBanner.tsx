import { Alert, App, Button, Space, Typography } from "@/ui-kit/eat";
import { useLLMStatusBanner } from "./useLLMStatusBanner";

const { Text } = Typography;

interface LLMStatusBannerProps {
	// needsReanalysis is sourced from dashboard-summary by the composing
	// page. The feature itself does not call useDashboardSummary to
	// preserve feature isolation (dependency-cruiser forbids cross-feature
	// imports) and to keep this component independent of the data
	// fetching strategy.
	needsReanalysis: boolean;
	// staleCardCount is used to format the banner copy. Undefined is
	// treated as "部分" so the copy still reads naturally when the caller
	// does not have an exact count.
	staleCardCount?: number;
	// onReanalyze is the reanalyze-all mutation trigger. The feature does
	// not reach across to decision-card to grab the hook itself — the
	// composing page is expected to pass the mutation surface in because
	// the dependency-cruiser forbids cross-feature imports.
	onReanalyze: () => Promise<void>;
	isReanalyzing: boolean;
}

// LLMStatusBanner renders the dashboard-top degraded-contract banner.
// It reads the session-scoped dismiss flag from useLLMStatusBanner and
// returns null when needsReanalysis is false OR the user has already
// dismissed the banner in the current session. Clicking the reanalyze
// CTA triggers the mutation passed by the composing page which
// invalidates the dashboard-summary cache, so the banner will disappear
// automatically once the backend flips needsReanalysis back to false.
export function LLMStatusBanner({
	needsReanalysis,
	staleCardCount,
	onReanalyze,
	isReanalyzing,
}: LLMStatusBannerProps) {
	const { message } = App.useApp();
	const { dismissed, dismiss } = useLLMStatusBanner();

	if (!needsReanalysis) return null;
	if (dismissed) return null;

	const countCopy = typeof staleCardCount === "number" ? `${staleCardCount} 个` : "部分";

	const handleReanalyze = async () => {
		try {
			await onReanalyze();
			message.success("已触发重新分析，请稍后刷新查看。");
		} catch (err) {
			const msg = err instanceof Error ? err.message : "请稍后再试";
			message.error(`重新分析请求失败：${msg}`);
		}
	};

	return (
		<Alert
			type="info"
			showIcon
			closable
			onClose={dismiss}
			data-testid="llm-status-banner"
			message={
				<Space direction="vertical" size={2}>
					<Text strong>AI 解读已配置</Text>
					<Text>
						你有 {countCopy} 持仓的历史卡片仍基于规则引擎生成，
						点击右侧按钮可触发一次重新分析以升级到 AI 解读。
					</Text>
				</Space>
			}
			action={
				<Button
					type="primary"
					size="small"
					loading={isReanalyzing}
					onClick={handleReanalyze}
					data-testid="llm-status-banner-reanalyze"
				>
					重新分析所有持仓
				</Button>
			}
		/>
	);
}
