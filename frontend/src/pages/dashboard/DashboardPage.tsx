import { LLMStatusBanner } from "@/features/dashboard-llm-status";
import { useDashboardSummary } from "@/features/dashboard-summary";
import {
	type DecisionCardDTO,
	useDecisionCards,
	useReanalyzeAll,
	useRerunAnalysis,
} from "@/features/decision-card";
import { useHoldings } from "@/features/portfolio";
import { useUserSettings } from "@/features/user-settings";
import { App, Flex, PageContainer, Space } from "@/ui-kit/eat";
import { useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { ChangeAnchorList } from "./components/ChangeAnchorList";
import { DashboardTopStrip, computeNextAnalysisTime } from "./components/DashboardTopStrip";
import { DecisionCardWall } from "./components/DecisionCardWall";
import { EmptyHoldingsHero } from "./components/EmptyHoldingsHero";
import { OnboardingSkippedNudge } from "./components/OnboardingSkippedNudge";

// DashboardPage is the composition root for PRD §3.1 three-region dashboard.
// Business logic is delegated to existing feature hooks so this file only
// orchestrates data flow between hooks and presentational sub-components.
export default function DashboardPage() {
	const navigate = useNavigate();
	const { t } = useTranslation("app");
	const { message } = App.useApp();

	const holdingsQuery = useHoldings();
	const cardsQuery = useDecisionCards();
	const settingsQuery = useUserSettings();
	const summaryQuery = useDashboardSummary();
	const rerun = useRerunAnalysis();
	const reanalyzeAll = useReanalyzeAll();

	// cardRefs is shared between DecisionCardWall (which populates it) and
	// ChangeAnchorList (which reads from it to scroll + highlight). A ref
	// instead of state keeps this out of the render cycle.
	const cardRefs = useRef(new Map<number, HTMLDivElement>()).current;

	const holdings = holdingsQuery.data ?? [];
	const cards: DecisionCardDTO[] = cardsQuery.data ?? [];
	const settings = settingsQuery.data;

	const lastAnalyzedAt = useMemo<Date | null>(() => {
		if (cards.length === 0) return null;
		const latest = cards.reduce((acc, card) => {
			const t = new Date(card.analyzedAt).getTime();
			return t > acc ? t : acc;
		}, 0);
		return latest > 0 ? new Date(latest) : null;
	}, [cards]);

	const nextAnalysisAt = useMemo(() => computeNextAnalysisTime(new Date()), []);

	const totalPositionRatio = useMemo(
		() => holdings.reduce((sum, h) => sum + (h.positionRatio ?? 0), 0),
		[holdings],
	);

	// Aggregate P&L is summed across cards with a known positionAmount.
	// When hasCapital is false (no totalCapitalCny) the amount is null and
	// the top strip renders percent-only via useMoney semantics.
	const { aggregatePnlAmount, aggregatePnlPct } = useMemo(() => {
		if (cards.length === 0) {
			return { aggregatePnlAmount: null as number | null, aggregatePnlPct: 0 };
		}
		let amount = 0;
		let hasAny = false;
		for (const card of cards) {
			if (card.positionAmount != null) {
				hasAny = true;
				// Proxy: use targetPositionAmount delta when available, else 0.
				// Backend does not yet expose a realized P&L field per card, so
				// this aggregate is a best-effort placeholder that will be
				// replaced when Step 17 (screenshot import + trade ledger) lands.
				const delta =
					card.targetPositionAmount != null ? card.targetPositionAmount - card.positionAmount : 0;
				amount += delta;
			}
		}
		const capital = settings?.totalCapitalCny ?? 0;
		const pct = hasAny && capital > 0 ? (amount / capital) * 100 : 0;
		return {
			aggregatePnlAmount: hasAny ? amount : null,
			aggregatePnlPct: pct,
		};
	}, [cards, settings?.totalCapitalCny]);

	const handleRerun = async () => {
		try {
			await rerun.mutateAsync();
			message.success(t("dashboard.message.rerunSuccess"));
		} catch {
			message.error(t("dashboard.message.rerunError"));
		}
	};

	const handleCardClick = (card: DecisionCardDTO) => {
		navigate(`/decision-cards/${card.cardId}`);
	};

	const handleConfigureCapital = () => {
		navigate("/settings");
	};

	const handleAddHolding = () => {
		// Step 16 deleted /portfolio/new in favour of an in-page Add Holding
		// Drawer on the Portfolio list. Route the empty hero CTA to the list
		// so the user can open the drawer themselves. Future polish: pass a
		// query param so the drawer auto-opens on arrival.
		navigate("/portfolio");
	};

	// staleCardCount counts decision cards with synthesisSource in
	// (template, mixed) for the LLM degraded-contract banner.
	const staleCardCount = useMemo(
		() =>
			cards.filter((c) => c.synthesisSource === "template" || c.synthesisSource === "mixed").length,
		[cards],
	);

	const llmStatus = summaryQuery.data?.llmStatus;
	const needsReanalysis = llmStatus?.needsReanalysis ?? false;

	const handleReanalyzeAll = async () => {
		await reanalyzeAll.mutateAsync();
	};

	// Restructured outer shell: a flex column wraps the nudge plus
	// whichever body branch applies, so the nudge can render above either the
	// empty-holdings hero or the populated three-region layout.
	const holdingsReady = !holdingsQuery.isLoading;
	const showEmptyHero = holdingsReady && holdings.length === 0;

	return (
		<PageContainer title={false} data-testid="dashboard-page">
			<Flex vertical gap={16}>
				<LLMStatusBanner
					needsReanalysis={needsReanalysis}
					staleCardCount={staleCardCount}
					onReanalyze={handleReanalyzeAll}
					isReanalyzing={reanalyzeAll.isPending}
				/>
				<OnboardingSkippedNudge />
				{showEmptyHero ? (
					<EmptyHoldingsHero onAddHolding={handleAddHolding} />
				) : (
					<Space direction="vertical" size={16} style={{ width: "100%" }}>
						<DashboardTopStrip
							holdingCount={holdings.length}
							totalCapitalCny={settings?.totalCapitalCny}
							totalPositionRatio={totalPositionRatio}
							aggregatePnlAmount={aggregatePnlAmount}
							aggregatePnlPct={aggregatePnlPct}
							lastAnalyzedAt={lastAnalyzedAt}
							nextAnalysisAt={nextAnalysisAt}
							onRerun={handleRerun}
							rerunLoading={rerun.isPending}
							onConfigureCapital={handleConfigureCapital}
						/>
						<ChangeAnchorList cards={cards} cardRefs={cardRefs} />
						<DecisionCardWall
							cards={cards}
							isLoading={cardsQuery.isLoading}
							error={cardsQuery.error}
							onCardClick={handleCardClick}
							onRetry={() => cardsQuery.refetch()}
							cardRefs={cardRefs}
						/>
					</Space>
				)}
			</Flex>
		</PageContainer>
	);
}
