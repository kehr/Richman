import { type DecisionCardDTO, computeNextAnalysisTime } from "@/features/decision-card";
import { Card, Divider, Space, Typography } from "@/ui-kit/eat";
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
// sidebar renders times in this timezone regardless of the viewer's locale
// so "下一次分析" matches the schedule the backend actually uses.
const SHANGHAI_TZ = "Asia/Shanghai";

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

				<div data-testid="meta-data-source">
					<Text type="secondary">数据源状态</Text>
					<Paragraph type="secondary" style={{ margin: 0 }}>
						数据源健康检查将在后续版本开放
					</Paragraph>
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
