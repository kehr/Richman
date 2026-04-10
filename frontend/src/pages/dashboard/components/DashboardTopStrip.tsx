import { DeltaDisplay } from "@/domain/money/DeltaDisplay";
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
	// taskProgress is a 0-1 fraction shown in the "View Progress" button while running.
	taskProgress: number;
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
	taskProgress,
	hasRecentTask,
	onShowHistory,
	onConfigureCapital,
}: DashboardTopStripProps) {
	const { t } = useTranslation("app");
	const money = useMoney();

	const hasCapital = totalCapitalCny != null;
	const capitalDisplay = money.formatAmountOnly(totalCapitalCny);

	const amountStr =
		money.hasCapital && aggregatePnlAmount != null
			? money.formatAmountOnly(aggregatePnlAmount)
			: null;

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
						{/* When running: progress button with percentage opens analysis modal */}
						{isRunning ? (
							<Button type="primary" onClick={onShowHistory} data-testid="dashboard-rerun-button">
								{t("analysisProgress.buttonRunning", { pct: Math.round(taskProgress * 100) })}
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
			<Flex vertical gap={6} style={{ margin: "20px 0 16px" }}>
				<Flex align="center" gap={5}>
					<Text type="secondary" style={{ fontSize: 12 }}>
						{t("dashboard.stat.suggestedRebalance")}
					</Text>
					<Tooltip title={t("dashboard.stat.suggestedRebalanceHint")}>
						<QuestionCircleOutlined style={{ fontSize: 11, color: "#8C8C8C", cursor: "default" }} />
					</Tooltip>
				</Flex>
				<DeltaDisplay
					pct={aggregatePnlPct}
					amount={amountStr}
					convention="green-up"
					showSign={false}
					primarySize={36}
					secondarySize={18}
					layout="horizontal"
					align="left"
					data-testid="stat-aggregate-pnl"
				/>
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
						<DeltaDisplay
							pct={holdingPnlPct}
							amount={holdingPnlAmountStr}
							convention="green-up"
							showSign={false}
							primarySize={20}
							secondarySize={12}
							align="left"
							data-testid="stat-holding-pnl"
						/>
					</Flex>
				</Col>
			</Row>
		</Card>
	);
}
