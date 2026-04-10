import {
	type Action,
	type DecisionCardDTO,
	computeNextAnalysisTime,
} from "@/features/decision-card";
import { Card, Divider, Space, Typography } from "@/ui-kit/eat";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { type MouseEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { HoldingScheduleSection, useHoldingNextAnalysisAt } from "./HoldingScheduleSection";

const PAGE_SIZE = 5;

// ACTION_STYLE maps each recommendation action to a dot color and label color
// used in the history strip. Colors follow the same semantic palette as the
// conclusion banner: green = add, blue = hold, orange/red = reduce.
const ACTION_STYLE: Record<Action, { dot: string; label: string }> = {
	aggressive_add: { dot: "#389e0d", label: "#389e0d" },
	small_add: { dot: "#52c41a", label: "#52c41a" },
	hold: { dot: "#1677ff", label: "#1677ff" },
	gradual_reduce: { dot: "#fa8c16", label: "#fa8c16" },
	control_position: { dot: "#f5222d", label: "#f5222d" },
};

const { Text } = Typography;

interface MetaSidebarProps {
	card: DecisionCardDTO;
	// historicalCards is the full ordered history list for the same holding,
	// including the current card. Pagination is handled internally.
	historicalCards?: DecisionCardDTO[];
	onSelectHistory?: (cardId: number) => void;
}

const SHANGHAI_TZ = "Asia/Shanghai";

function formatShanghaiDateTime(date: Date | null, locale: string): string {
	if (!date) return "--";
	const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
	const fmt = new Intl.DateTimeFormat(intlLocale, {
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

// formatHistoryDate renders a compact "MM-DD HH:mm" date for the history strip.
// formatHistoryDate renders "YYYY-MM-DD HH:mm" in Shanghai time.
function formatHistoryDate(isoDate: string, locale: string): string {
	const d = new Date(isoDate);
	if (Number.isNaN(d.getTime())) return "--";
	const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
	const fmt = new Intl.DateTimeFormat(intlLocale, {
		timeZone: SHANGHAI_TZ,
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
		hour12: false,
	});
	const parts = fmt.formatToParts(d);
	const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "";
	return `${get("year")}-${get("month")}-${get("day")} ${get("hour")}:${get("minute")}:${get("second")}`;
}

export function MetaSidebar({ card, historicalCards = [], onSelectHistory }: MetaSidebarProps) {
	const { t, i18n } = useTranslation("app");
	const [page, setPage] = useState(0);
	const analyzedAt = new Date(card.analyzedAt);

	const backendNextAt = useHoldingNextAnalysisAt(card.holdingId);
	const nextAnalysisAt = backendNextAt
		? new Date(backendNextAt)
		: computeNextAnalysisTime(new Date());

	const totalPages = Math.ceil(historicalCards.length / PAGE_SIZE);
	const pageItems = historicalCards.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

	const handleSelect = (cardId: number) => (event: MouseEvent<HTMLElement>) => {
		event.preventDefault();
		onSelectHistory?.(cardId);
	};

	return (
		<Card data-testid="meta-sidebar" size="small" title={t("decisionCard.metaSidebar.title")}>
			<Space direction="vertical" size={12} style={{ width: "100%" }}>
				<div>
					<Text type="secondary">{t("decisionCard.metaSidebar.analyzedAt")}</Text>
					<div data-testid="meta-analyzed-at">
						{formatShanghaiDateTime(analyzedAt, i18n.language)} (Asia/Shanghai)
					</div>
				</div>

				<div data-testid="meta-data-source">
					<Text type="secondary">{t("decisionCard.metaSidebar.dataSource")}</Text>
					<Typography.Paragraph type="secondary" style={{ margin: 0 }}>
						{t("decisionCard.metaSidebar.dataSourcePending")}
					</Typography.Paragraph>
				</div>

				<div>
					<Text type="secondary">{t("decisionCard.metaSidebar.nextAnalysis")}</Text>
					<div data-testid="meta-next-analysis">
						{formatShanghaiDateTime(nextAnalysisAt, i18n.language)}
					</div>
				</div>

				<HoldingScheduleSection holdingId={card.holdingId} />

				<Divider style={{ margin: "4px 0" }} />

				<div>
					<Text type="secondary">{t("decisionCard.metaSidebar.history")}</Text>

					{historicalCards.length === 0 ? (
						<Typography.Paragraph type="secondary" style={{ margin: "4px 0 0" }}>
							{t("decisionCard.metaSidebar.noHistory")}
						</Typography.Paragraph>
					) : (
						<>
							<div style={{ marginTop: 8 }} data-testid="meta-history-list">
								{pageItems.map((h) => {
									const isCurrent = h.cardId === card.cardId;
									const style = ACTION_STYLE[h.recommendation.action] ?? {
										dot: "#8c8c8c",
										label: "#8c8c8c",
									};
									return (
										<button
											key={h.cardId}
											type="button"
											onClick={isCurrent ? undefined : handleSelect(h.cardId)}
											disabled={isCurrent}
											data-testid={`meta-history-${h.cardId}`}
											style={{
												display: "flex",
												alignItems: "flex-start",
												gap: 8,
												width: "100%",
												background: isCurrent ? "#fafafa" : "transparent",
												border: "none",
												borderRadius: 6,
												padding: "6px 8px",
												marginBottom: 2,
												cursor: isCurrent ? "default" : "pointer",
												textAlign: "left",
												transition: "background 0.15s",
											}}
											onMouseEnter={(e) => {
												if (!isCurrent) {
													(e.currentTarget as HTMLButtonElement).style.background = "#f5f5f5";
												}
											}}
											onMouseLeave={(e) => {
												if (!isCurrent) {
													(e.currentTarget as HTMLButtonElement).style.background = "transparent";
												}
											}}
										>
											{/* colored indicator dot, vertically centered to first line */}
											<span
												style={{
													flexShrink: 0,
													marginTop: 4,
													width: 6,
													height: 6,
													borderRadius: "50%",
													background: style.dot,
												}}
											/>

											<div style={{ flex: 1, minWidth: 0 }}>
												{/* date row */}
												<div
													style={{
														display: "flex",
														alignItems: "center",
														justifyContent: "space-between",
														gap: 4,
													}}
												>
													<span
														style={{
															fontSize: 12,
															color: "#8c8c8c",
															fontVariantNumeric: "tabular-nums",
															flexShrink: 0,
														}}
													>
														{formatHistoryDate(h.analyzedAt, i18n.language)}
													</span>
													{isCurrent && (
														<span
															style={{
																fontSize: 11,
																color: "#8c8c8c",
																border: "1px solid #d9d9d9",
																borderRadius: 3,
																padding: "0 4px",
																lineHeight: "16px",
																flexShrink: 0,
															}}
														>
															{t("decisionCard.metaSidebar.historyCurrent")}
														</span>
													)}
												</div>

												{/* action label row */}
												<div
													style={{
														marginTop: 2,
														fontSize: 12,
														fontWeight: isCurrent ? 600 : 500,
														color: style.label,
														whiteSpace: "nowrap",
														overflow: "hidden",
														textOverflow: "ellipsis",
													}}
												>
													{t(`decisionCard.recommendation.${h.recommendation.action}`)}
												</div>
											</div>
										</button>
									);
								})}
							</div>

							{/* pagination — only rendered when there are multiple pages */}
							{totalPages > 1 && (
								<div
									style={{
										display: "flex",
										alignItems: "center",
										justifyContent: "center",
										gap: 8,
										marginTop: 6,
									}}
									data-testid="meta-history-pagination"
								>
									<button
										type="button"
										onClick={() => setPage((p) => Math.max(0, p - 1))}
										disabled={page === 0}
										style={{
											display: "inline-flex",
											alignItems: "center",
											background: "transparent",
											border: "none",
											cursor: page === 0 ? "default" : "pointer",
											color: page === 0 ? "#d9d9d9" : "#595959",
											padding: "2px 4px",
											borderRadius: 4,
										}}
										aria-label="previous page"
									>
										<ChevronLeft size={14} />
									</button>

									<span
										style={{ fontSize: 12, color: "#8c8c8c", minWidth: 32, textAlign: "center" }}
									>
										{t("decisionCard.metaSidebar.historyPageOf", {
											current: page + 1,
											total: totalPages,
										})}
									</span>

									<button
										type="button"
										onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
										disabled={page === totalPages - 1}
										style={{
											display: "inline-flex",
											alignItems: "center",
											background: "transparent",
											border: "none",
											cursor: page === totalPages - 1 ? "default" : "pointer",
											color: page === totalPages - 1 ? "#d9d9d9" : "#595959",
											padding: "2px 4px",
											borderRadius: 4,
										}}
										aria-label="next page"
									>
										<ChevronRight size={14} />
									</button>
								</div>
							)}
						</>
					)}
				</div>

				<Divider style={{ margin: "4px 0" }} />

				<Text type="secondary" data-testid="meta-disclaimer">
					{t("decisionCard.metaSidebar.disclaimer")}
				</Text>
			</Space>
		</Card>
	);
}
