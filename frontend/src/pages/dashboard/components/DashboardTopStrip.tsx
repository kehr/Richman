import { useMoney } from "@/domain/money/useMoney";
import { Button, Card, Col, ReloadOutlined, Row, Space, Typography } from "@/ui-kit/eat";

const { Text, Title } = Typography;

// Analysis schedule constants mirror backend/internal/service/analysis/scheduler.go:
//   AM brief  : 08:30 Asia/Shanghai, Mon-Fri
//   PM digest : 15:30 Asia/Shanghai, Mon-Fri
//   US digest : 06:00 Asia/Shanghai, Tue-Sat
// When the backend grows an explicit "next analysis" endpoint this helper
// should switch to that response. Until then we compute the next slot on the
// client so the strip always has a value to show.
interface ScheduleSlot {
	hour: number;
	minute: number;
	weekdays: number[]; // 0 = Sunday
}

const ANALYSIS_SCHEDULE: ScheduleSlot[] = [
	{ hour: 6, minute: 0, weekdays: [2, 3, 4, 5, 6] }, // US digest
	{ hour: 8, minute: 30, weekdays: [1, 2, 3, 4, 5] }, // A-share AM
	{ hour: 15, minute: 30, weekdays: [1, 2, 3, 4, 5] }, // A-share PM
];

// computeNextAnalysisTime returns the next scheduled analysis slot strictly
// after `now`, expressed in Asia/Shanghai local wall time. We scan the next
// seven days to guarantee a hit even across weekends.
export function computeNextAnalysisTime(now: Date): Date | null {
	const candidates: Date[] = [];
	for (let offset = 0; offset < 7; offset += 1) {
		const day = new Date(now);
		day.setDate(day.getDate() + offset);
		const weekday = day.getDay();
		for (const slot of ANALYSIS_SCHEDULE) {
			if (!slot.weekdays.includes(weekday)) continue;
			const candidate = new Date(day);
			candidate.setHours(slot.hour, slot.minute, 0, 0);
			if (candidate.getTime() > now.getTime()) {
				candidates.push(candidate);
			}
		}
	}
	if (candidates.length === 0) return null;
	candidates.sort((a, b) => a.getTime() - b.getTime());
	return candidates[0];
}

// formatHm renders a Date as "HH:MM" in 24h clock using the user's local
// timezone. This is fine for the Dashboard because the scheduler also runs
// in Asia/Shanghai which matches the product's primary user base.
export function formatHm(date: Date | null | undefined): string {
	if (!date) return "--:--";
	const hh = String(date.getHours()).padStart(2, "0");
	const mm = String(date.getMinutes()).padStart(2, "0");
	return `${hh}:${mm}`;
}

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
						<Text type="secondary">综合浮盈亏</Text>
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
