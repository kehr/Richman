import { useMoney } from "@/domain/money/useMoney";
import { Button, Card, Col, ReloadOutlined, Row, Space, Tooltip, Typography } from "@/ui-kit/eat";

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

// SHANGHAI_TZ is the IANA timezone matching the backend scheduler. All
// schedule comparisons must happen in this zone so users in other locales
// (CI, server-rendered, traveling staff) still see the correct slot.
const SHANGHAI_TZ = "Asia/Shanghai";

// shanghaiPartsFromInstant projects a UTC instant into Shanghai wall-clock
// parts (year/month/day/hour/minute/weekday). This avoids the trap where
// Date#getDay/getHours read the host machine's local clock.
function shanghaiPartsFromInstant(instant: Date): {
	year: number;
	month: number;
	day: number;
	hour: number;
	minute: number;
	weekday: number;
} {
	const fmt = new Intl.DateTimeFormat("en-US", {
		timeZone: SHANGHAI_TZ,
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
		weekday: "short",
		hour12: false,
	});
	const parts = fmt.formatToParts(instant);
	const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "0";
	const weekdayMap: Record<string, number> = {
		Sun: 0,
		Mon: 1,
		Tue: 2,
		Wed: 3,
		Thu: 4,
		Fri: 5,
		Sat: 6,
	};
	return {
		year: Number.parseInt(get("year"), 10),
		month: Number.parseInt(get("month"), 10),
		day: Number.parseInt(get("day"), 10),
		// Intl can return "24" for midnight in the en-US 24h locale; normalize to 0.
		hour: Number.parseInt(get("hour"), 10) % 24,
		minute: Number.parseInt(get("minute"), 10),
		weekday: weekdayMap[get("weekday")] ?? 0,
	};
}

// computeNextAnalysisTime returns the next scheduled analysis slot strictly
// after `now`, computed against Shanghai wall-clock time so the result is
// correct regardless of where the user's browser is running. The return
// value is a UTC Date instance representing that exact instant.
export function computeNextAnalysisTime(now: Date): Date | null {
	const start = shanghaiPartsFromInstant(now);
	const startMinutes = start.hour * 60 + start.minute;
	for (let offset = 0; offset < 7; offset += 1) {
		const weekday = (start.weekday + offset) % 7;
		for (const slot of ANALYSIS_SCHEDULE) {
			if (!slot.weekdays.includes(weekday)) continue;
			const slotMinutes = slot.hour * 60 + slot.minute;
			if (offset === 0 && slotMinutes <= startMinutes) continue;
			// Build the candidate by adding `offset` days to today's Shanghai
			// date and then setting the slot's hour/minute. We construct the
			// instant via Date.UTC + the Shanghai offset of 8 hours so the
			// returned Date is the correct UTC moment.
			const baseUtcMs = Date.UTC(start.year, start.month - 1, start.day) + offset * 86400000;
			const candidateUtcMs = baseUtcMs + (slot.hour - 8) * 3600000 + slot.minute * 60000;
			return new Date(candidateUtcMs);
		}
	}
	return null;
}

// formatHm renders a Date as "HH:MM" in Shanghai wall-clock 24h. We use Intl
// rather than getHours/getMinutes so users outside Asia/Shanghai still see
// the schedule's local time, matching the rest of the product.
export function formatHm(date: Date | null | undefined): string {
	if (!date) return "--:--";
	const fmt = new Intl.DateTimeFormat("en-GB", {
		timeZone: SHANGHAI_TZ,
		hour: "2-digit",
		minute: "2-digit",
		hour12: false,
	});
	return fmt.format(date);
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
