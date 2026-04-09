import { useMoney } from "@/domain/money/useMoney";
import { computeNextAnalysisTime, formatHm } from "@/features/decision-card";
import { Button, Card, Col, ReloadOutlined, Row, Space, Tooltip, Typography } from "@/ui-kit/eat";
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
	const { t } = useTranslation("app");
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
							{t("dashboard.todayDecision")}
						</Title>
						<Text type="secondary" data-testid="dashboard-top-strip-times">
							{t("dashboard.lastAnalyzed")} {formatHm(lastAnalyzedAt)} · {t("dashboard.nextAuto")}{" "}
							{formatHm(nextAnalysisAt)}
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
						{t("dashboard.rerunButton")}
					</Button>
				</Col>
			</Row>

			<Row gutter={[16, 16]} style={{ marginTop: 20 }}>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Text type="secondary">{t("dashboard.stat.holdingCount")}</Text>
						<Text strong style={{ fontSize: 22 }} data-testid="stat-holding-count">
							{holdingCount}
						</Text>
					</Space>
				</Col>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Text type="secondary">{t("dashboard.stat.totalCapital")}</Text>
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
								{t("dashboard.stat.totalCapitalCta")}
							</Button>
						)}
					</Space>
				</Col>
				<Col xs={12} md={6}>
					<Space direction="vertical" size={2}>
						<Text type="secondary">{t("dashboard.stat.suggestedRebalance")}</Text>
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
						<Text type="secondary">{t("dashboard.stat.allocatedPosition")}</Text>
						<Text strong style={{ fontSize: 22 }} data-testid="stat-allocated-position">
							{money.format(totalPositionRatio)}
						</Text>
					</Space>
				</Col>
			</Row>
		</Card>
	);
}
