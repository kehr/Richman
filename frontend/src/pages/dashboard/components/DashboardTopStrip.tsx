import { formatPercent } from "@/domain/money/format";
import { useMoney } from "@/domain/money/useMoney";
import { computeNextAnalysisTime, formatHm } from "@/features/decision-card";
import {
	Button,
	Card,
	Col,
	Divider,
	Flex,
	QuestionCircleOutlined,
	ReloadOutlined,
	Row,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

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
	holdingPnlAmount: number | null; // unrealized P&L absolute amount; null when no price data
	holdingPnlPct: number; // unrealized P&L as % of cost basis
	lastAnalyzedAt: Date | null;
	nextAnalysisAt: Date | null;
	onRerun: () => void;
	isRunning: boolean;
	// hasRecentTask is true when a taskId is in state (running or completed)
	// so the "view last analysis" link can be shown.
	hasRecentTask: boolean;
	onShowHistory: () => void;
	onConfigureCapital: () => void;
}

// DashboardTopStrip is the top region of the Dashboard per PRD §3.1. It is a
// presentational component: all data (counts, amounts, mutation state) is
// passed by the parent page so this file stays test-friendly and can be
// rendered without a QueryClient.
//
// Layout: header row (title + button) → hero rebalance section → supporting
// stats row. The suggested rebalance is the primary actionable signal and
// receives the largest visual weight. The three supporting stats (holding
// count, total capital, allocated position) are secondary context.
export function DashboardTopStrip({
	holdingCount,
	totalCapitalCny,
	totalPositionRatio,
	aggregatePnlAmount,
	aggregatePnlPct,
	holdingPnlAmount,
	holdingPnlPct,
	lastAnalyzedAt,
	nextAnalysisAt,
	onRerun,
	isRunning,
	hasRecentTask,
	onShowHistory,
	onConfigureCapital,
}: DashboardTopStripProps) {
	const { t } = useTranslation("app");
	const money = useMoney();

	const hasCapital = totalCapitalCny != null;
	const capitalDisplay = hasCapital
		? `¥${new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 }).format(totalCapitalCny as number)}`
		: null;

	// Directional color: use theme palette values (success/error) rather than
	// saturated Chinese-market reds to keep the signal informational, not alarming.
	const pnlColor = aggregatePnlPct > 0 ? "#10B981" : aggregatePnlPct < 0 ? "#EF4444" : undefined;

	const amountStr =
		money.hasCapital && aggregatePnlAmount != null
			? money.formatAmountOnly(aggregatePnlAmount)
			: null;

	const holdingPnlColor = holdingPnlPct > 0 ? "#10B981" : holdingPnlPct < 0 ? "#EF4444" : undefined;
	const holdingPnlAmountStr =
		holdingPnlAmount != null ? money.formatAmountOnly(holdingPnlAmount) : null;

	return (
		<Card data-testid="dashboard-top-strip" styles={{ body: { padding: 20 } }}>
			{/* Header: title + meta + rerun button */}
			<Row align="middle" justify="space-between" gutter={[16, 12]}>
				<Col flex="auto">
					<Flex vertical gap={2}>
						<Title level={3} style={{ margin: 0 }}>
							{t("dashboard.todayDecision")}
						</Title>
						<Text type="secondary" data-testid="dashboard-top-strip-times">
							{t("dashboard.lastAnalyzed")} {formatHm(lastAnalyzedAt)} · {t("dashboard.nextAuto")}{" "}
							{formatHm(nextAnalysisAt)}
						</Text>
					</Flex>
				</Col>
				<Col>
					<Flex align="center" gap={8}>
						{/* When running: progress button opens analysis modal */}
						{isRunning ? (
							<Button type="primary" onClick={onShowHistory} data-testid="dashboard-rerun-button">
								{t("analysisProgress.buttonRunning")}
							</Button>
						) : (
							<Button
								icon={<ReloadOutlined />}
								onClick={onRerun}
								data-testid="dashboard-rerun-button"
							>
								{t("dashboard.rerunButton")}
							</Button>
						)}
						{/* Link to view last analysis — only visible when a task exists and is not running */}
						{hasRecentTask && !isRunning && (
							<Button
								type="link"
								size="small"
								style={{ padding: 0 }}
								onClick={onShowHistory}
								data-testid="dashboard-view-history-button"
							>
								{t("analysisProgress.viewLast")} →
							</Button>
						)}
					</Flex>
				</Col>
			</Row>

			{/* Hero: suggested rebalance — primary actionable signal */}
			<Flex align="flex-end" justify="space-between" style={{ margin: "20px 0 16px" }}>
				<Flex vertical gap={6}>
					<Flex align="center" gap={5}>
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("dashboard.stat.suggestedRebalance")}
						</Text>
						<Tooltip title={t("dashboard.stat.suggestedRebalanceHint")}>
							<QuestionCircleOutlined
								style={{ fontSize: 11, color: "#8C8C8C", cursor: "default" }}
							/>
						</Tooltip>
					</Flex>
					<Text
						strong
						style={{ fontSize: 36, lineHeight: 1, color: pnlColor }}
						data-testid="stat-aggregate-pnl"
					>
						{formatPercent(aggregatePnlPct)}
					</Text>
				</Flex>
				{amountStr != null && (
					<Text type="secondary" style={{ fontSize: 18, paddingBottom: 4 }}>
						{amountStr}
					</Text>
				)}
			</Flex>

			<Divider style={{ margin: 0 }} />

			{/* Supporting stats: context metrics, visually subordinate (4-col) */}
			<Row gutter={[16, 12]} style={{ marginTop: 16 }}>
				<Col xs={12} md={6}>
					<Flex vertical gap={3}>
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("dashboard.stat.holdingCount")}
						</Text>
						<Text strong style={{ fontSize: 20 }} data-testid="stat-holding-count">
							{holdingCount}
						</Text>
					</Flex>
				</Col>
				<Col xs={12} md={6}>
					<Flex vertical gap={3}>
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("dashboard.stat.totalCapital")}
						</Text>
						{hasCapital ? (
							<Text strong style={{ fontSize: 20 }} data-testid="stat-total-capital">
								{capitalDisplay}
							</Text>
						) : (
							<Button
								type="link"
								size="small"
								style={{ padding: 0, height: "auto", fontSize: 13 }}
								onClick={onConfigureCapital}
								data-testid="stat-total-capital-cta"
							>
								{t("dashboard.stat.totalCapitalCta")}
							</Button>
						)}
					</Flex>
				</Col>
				<Col xs={12} md={6}>
					<Flex vertical gap={3}>
						<Text type="secondary" style={{ fontSize: 12 }}>
							{t("dashboard.stat.allocatedPosition")}
						</Text>
						<Text strong style={{ fontSize: 20 }} data-testid="stat-allocated-position">
							{money.format(totalPositionRatio)}
						</Text>
					</Flex>
				</Col>
				<Col xs={12} md={6}>
					<Flex vertical gap={3}>
						<Flex align="center" gap={5}>
							<Text type="secondary" style={{ fontSize: 12 }}>
								{t("dashboard.stat.holdingPnl")}
							</Text>
							<Tooltip title={t("dashboard.stat.holdingPnlHint")}>
								<QuestionCircleOutlined
									style={{ fontSize: 11, color: "#8C8C8C", cursor: "default" }}
								/>
							</Tooltip>
						</Flex>
						<Text
							strong
							style={{ fontSize: 20, color: holdingPnlColor }}
							data-testid="stat-holding-pnl"
						>
							{formatPercent(holdingPnlPct)}
						</Text>
						{holdingPnlAmountStr != null && (
							<Text type="secondary" style={{ fontSize: 12 }}>
								{holdingPnlAmountStr}
							</Text>
						)}
					</Flex>
				</Col>
			</Row>
		</Card>
	);
}
