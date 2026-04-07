import type { DecisionCardDTO } from "@/features/decision-card";
import { Card, Divider, Space, Tag, Typography } from "@/ui-kit/eat";
import type { MouseEvent } from "react";

const { Text, Paragraph } = Typography;

interface MetaSidebarProps {
	card: DecisionCardDTO;
	// historicalCards is the recent list of cards for the same holding. The
	// current card may appear in the list and is filtered out automatically
	// so consumers do not need to pre-slice it.
	historicalCards?: DecisionCardDTO[];
	onSelectHistory?: (cardId: number) => void;
}

// SHANGHAI_TZ mirrors backend/internal/service/analysis/scheduler.go. The
// sidebar must render times in this timezone regardless of the viewer's
// locale so "下一次分析" matches the schedule the backend actually uses.
const SHANGHAI_TZ = "Asia/Shanghai";

interface ScheduleSlot {
	hour: number;
	minute: number;
	weekdays: number[];
}

const ANALYSIS_SCHEDULE: ScheduleSlot[] = [
	{ hour: 6, minute: 0, weekdays: [2, 3, 4, 5, 6] },
	{ hour: 8, minute: 30, weekdays: [1, 2, 3, 4, 5] },
	{ hour: 15, minute: 30, weekdays: [1, 2, 3, 4, 5] },
];

// shanghaiPartsFromInstant projects a UTC instant into Shanghai wall-clock
// parts so we can index into the schedule correctly.
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
		hour: Number.parseInt(get("hour"), 10) % 24,
		minute: Number.parseInt(get("minute"), 10),
		weekday: weekdayMap[get("weekday")] ?? 0,
	};
}

// computeNextAnalysisTime returns the next scheduled analysis slot strictly
// after `now`. Duplicated locally (not imported from dashboard) so the
// decision card detail page does not gain a page-to-page dependency.
export function computeNextAnalysisTime(now: Date): Date | null {
	const start = shanghaiPartsFromInstant(now);
	const startMinutes = start.hour * 60 + start.minute;
	for (let offset = 0; offset < 7; offset += 1) {
		const weekday = (start.weekday + offset) % 7;
		for (const slot of ANALYSIS_SCHEDULE) {
			if (!slot.weekdays.includes(weekday)) continue;
			const slotMinutes = slot.hour * 60 + slot.minute;
			if (offset === 0 && slotMinutes <= startMinutes) continue;
			const baseUtcMs = Date.UTC(start.year, start.month - 1, start.day) + offset * 86400000;
			const candidateUtcMs = baseUtcMs + (slot.hour - 8) * 3600000 + slot.minute * 60000;
			return new Date(candidateUtcMs);
		}
	}
	return null;
}

// formatShanghaiDateTime renders a Date as "YYYY-MM-DD HH:mm" in Shanghai
// wall-clock. Returns a dash placeholder when input is null so the sidebar
// always has a stable layout.
function formatShanghaiDateTime(date: Date | null): string {
	if (!date) return "--";
	const fmt = new Intl.DateTimeFormat("en-GB", {
		timeZone: SHANGHAI_TZ,
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
		hour12: false,
	});
	const parts = fmt.formatToParts(date);
	const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "";
	return `${get("year")}-${get("month")}-${get("day")} ${get("hour")}:${get("minute")}`;
}

// MetaSidebar renders the right-hand meta column for the decision card
// detail page per PRD section 5: analysis time + timezone, data source
// health, next scheduled analysis, short history list, and risk disclaimer.
//
// The data source status block is currently a static mock because the
// backend DTO does not yet expose per-source freshness; Step 17 (trade
// ledger + screenshot intake) is expected to add the real feed.
export function MetaSidebar({ card, historicalCards = [], onSelectHistory }: MetaSidebarProps) {
	const analyzedAt = new Date(card.analyzedAt);
	const nextAnalysisAt = computeNextAnalysisTime(new Date());
	const historyItems = historicalCards.filter((c) => c.cardId !== card.cardId).slice(0, 5);

	const handleSelect = (cardId: number) => (event: MouseEvent<HTMLElement>) => {
		event.preventDefault();
		onSelectHistory?.(cardId);
	};

	return (
		<Card data-testid="meta-sidebar" size="small" title="分析元信息">
			<Space direction="vertical" size={12} style={{ width: "100%" }}>
				<div>
					<Text type="secondary">分析时间</Text>
					<div data-testid="meta-analyzed-at">
						{formatShanghaiDateTime(analyzedAt)} (Asia/Shanghai)
					</div>
				</div>

				<div>
					<Text type="secondary">数据源状态</Text>
					<Space direction="vertical" size={2} style={{ width: "100%" }}>
						<Space>
							<Tag color="green">AKShare</Tag>
							<Text type="secondary">正常</Text>
						</Space>
						<Space>
							<Tag color="green">Yahoo Finance</Tag>
							<Text type="secondary">正常</Text>
						</Space>
						<Space>
							<Tag color="green">Polymarket</Tag>
							<Text type="secondary">正常</Text>
						</Space>
					</Space>
				</div>

				<div>
					<Text type="secondary">下一次自动分析</Text>
					<div data-testid="meta-next-analysis">{formatShanghaiDateTime(nextAnalysisAt)}</div>
				</div>

				<Divider style={{ margin: "4px 0" }} />

				<div>
					<Text type="secondary">历史分析</Text>
					{historyItems.length === 0 ? (
						<Paragraph type="secondary" style={{ margin: 0 }}>
							暂无更多历史
						</Paragraph>
					) : (
						<Space direction="vertical" size={4} style={{ width: "100%" }}>
							{historyItems.map((h) => (
								<button
									type="button"
									key={h.cardId}
									onClick={handleSelect(h.cardId)}
									data-testid={`meta-history-${h.cardId}`}
									style={{
										textAlign: "left",
										background: "transparent",
										border: "none",
										padding: 0,
										cursor: onSelectHistory ? "pointer" : "default",
										color: "#1677ff",
									}}
								>
									{formatShanghaiDateTime(new Date(h.analyzedAt))} · {h.recommendation.label}
								</button>
							))}
						</Space>
					)}
				</div>

				<Divider style={{ margin: "4px 0" }} />

				<Text type="secondary" data-testid="meta-disclaimer">
					本内容仅供参考，不构成投资建议。投资有风险，决策需谨慎。
				</Text>
			</Space>
		</Card>
	);
}
