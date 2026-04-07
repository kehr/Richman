import { useMoney } from "@/domain/money/useMoney";
import { computeNextAnalysisTime, formatHm } from "@/features/decision-card";
import { Button, Card, Col, ReloadOutlined, Row, Space, Tooltip, Typography } from "@/ui-kit/eat";

// Re-export the schedule helpers for callers that already imported them
// from this module before the helpers were promoted into the feature
// barrel. New callers should import directly from "@/features/decision-card".
export { computeNextAnalysisTime, formatHm };

const { Text, Title } = Typography;

interface DashboardTopStripProps {
	holdingCount: number;
	totalCapitalCny: number | null | undefined;
	totalPositionRatio: number; // 0..100, sum of all holding position ratios
	aggregatePnlAmount: number | null | undefined;
	aggregatePnlPct: number; // 0..100 percent of capital
	lastAnalyzedAt: Date | null;
	nextAnalysisAt: Date | null;
	onRerun: () => void;
	rerunLoading: boolean;
	onConfigureCapital: () => void;
}

// DashboardTopStrip is the top region of the Dashboard per PRD §3.1. It is a
// presentational component: all data (counts, amounts, mutation state) is
// passed by the parent page so this file stays test-friendly and can be
// rendered without a QueryClient.
export function DashboardTopStrip({
	holdingCount,
	totalCapitalCny,
	totalPositionRatio,
	aggregatePnlAmount,
	aggregatePnlPct,
	lastAnalyzedAt,
	nextAnalysisAt,
	onRerun,
	rerunLoading,
	onConfigureCapital,
}: DashboardTopStripProps) {
	const money = useMoney();
	const hasCapital = totalCapitalCny != null;
	const capitalDisplay = hasCapital
		? `¥${new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 }).format(totalCapitalCny as number)}`
		: null;

	return (
		<Card data-testid="dashboard-top-strip" styles={{ body: { padding: 20 } }}>
			<Row align="middle" justify="space-between" gutter={[16, 12]}>
				<Col flex="auto">
					<Space direction="vertical" size={2}>
						<Title level={3} style={{ margin: 0 }}>
							今日决策
						</Title>
						<Text type="secondary" data-testid="dashboard-top-strip-times">
							最后分析 {formatHm(lastAnalyzedAt)} · 下次自动 {formatHm(nextAnalysisAt)}
						</Text>
					</Space>
				</Col>
				<Col>
					<Button
						type="primary"
						size="large"
						icon={<ReloadOutlined />}
						loading={rerunLoading}
						onClick={onRerun}
						data-testid="dashboard-rerun-button"
					>
						重新分析
					</Button>
				</Col>
			</Row>

			<Row gutter={[16, 16]} style={{ marginTop: 20 }}>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Text type="secondary">持仓数</Text>
						<Text strong style={{ fontSize: 22 }} data-testid="stat-holding-count">
							{holdingCount}
						</Text>
					</Space>
				</Col>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Text type="secondary">总资金</Text>
						{hasCapital ? (
							<Text strong style={{ fontSize: 22 }} data-testid="stat-total-capital">
								{capitalDisplay}
							</Text>
						) : (
							<Button
								type="link"
								size="small"
								style={{ padding: 0, height: "auto", fontSize: 14 }}
								onClick={onConfigureCapital}
								data-testid="stat-total-capital-cta"
							>
								设置以查看 →
							</Button>
						)}
					</Space>
				</Col>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Tooltip title="按当前推荐目标仓位计算的整体调仓金额。MVP 阶段使用 (target - current) 作为代理；真实持仓盈亏将在 Step 17 接入交易记录后替换。">
							<Text type="secondary">建议调仓</Text>
						</Tooltip>
						<Text
							strong
							style={{
								fontSize: 22,
								color:
									aggregatePnlPct > 0 ? "#389e0d" : aggregatePnlPct < 0 ? "#cf1322" : undefined,
							}}
							data-testid="stat-aggregate-pnl"
						>
							{money.hasCapital && aggregatePnlAmount != null
								? money.format(aggregatePnlPct, aggregatePnlAmount)
								: money.format(aggregatePnlPct)}
						</Text>
					</Space>
				</Col>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Text type="secondary">已分配仓位</Text>
						<Text strong style={{ fontSize: 22 }} data-testid="stat-allocated-position">
							{money.format(totalPositionRatio)}
						</Text>
					</Space>
				</Col>
			</Row>
		</Card>
	);
}
