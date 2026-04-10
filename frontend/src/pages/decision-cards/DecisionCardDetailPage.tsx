import { useDecisionCardDetail, useHoldingHistory } from "@/features/decision-card";
import { Alert, Col, PageContainer, Row, Skeleton, Space } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { Link, useNavigate, useParams } from "react-router";
import { CardHero } from "./components/CardHero";
import { ConclusionBanner } from "./components/ConclusionBanner";
import { DimensionReasoning } from "./components/DimensionReasoning";
import { ExecutionPlanFull } from "./components/ExecutionPlanFull";
import { MainRisks } from "./components/MainRisks";
import { MetaSidebar } from "./components/MetaSidebar";

// formatAnalysisTime renders the analysis timestamp shown in the page subtitle.
// Returns a dash placeholder when the input is invalid.
function formatAnalysisTime(iso: string | undefined): string {
	if (!iso) return "--";
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return "--";
	const fmt = new Intl.DateTimeFormat("en-GB", {
		timeZone: "Asia/Shanghai",
		year: "numeric",
		month: "2-digit",
		day: "2-digit",
		hour: "2-digit",
		minute: "2-digit",
		hour12: false,
	});
	const parts = fmt.formatToParts(d);
	const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "";
	return `${get("year")}-${get("month")}-${get("day")} ${get("hour")}:${get("minute")}`;
}

// DecisionCardDetailPage renders the full 5-block reasoning view for a
// single decision card per PRD section 5. Layout uses a two-column grid
// (main content + 260px right sidebar) that collapses to a single column
// stack on viewports narrower than 1024px via the antd Row/Col responsive
// breakpoints (xs:24 / lg:18+6).
//
// Data flow:
//   - useDecisionCardDetail(id)            : current card
//   - useDecisionCardDetail(prevCardId)    : optional previous card, lazily
//                                            fetched only when prev id known
//   - useDecisionCards()                   : latest list, used to derive a
//                                            short history strip on the
//                                            sidebar by filtering same holding
export default function DecisionCardDetailPage() {
	const { t } = useTranslation("app");
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	// Parse the route param defensively: a non-numeric or zero/negative id
	// (bookmark typo, stale link) collapses to 0 which the hook's enabled
	// guard treats as "do not fetch", so we render the not-found branch
	// without a wasted network round-trip.
	const parsed = Number(id);
	const cardId = Number.isFinite(parsed) && parsed > 0 ? parsed : 0;

	const detailQuery = useDecisionCardDetail(cardId);
	const card = detailQuery.data;

	// prevCard fetch is lazy: useDecisionCardDetail's enabled guard skips the
	// request when prevCardId is null/undefined, so we can pass 0 as a safe
	// sentinel without triggering a network call.
	const prevCardId = card?.prevCardId ?? 0;
	const prevQuery = useDecisionCardDetail(prevCardId);
	const prevCard = prevQuery.data;

	// History strip: load recent cards for this holding specifically.
	// The hook is disabled until card is loaded (holdingId 0 is sentinel).
	const historyQuery = useHoldingHistory(card?.holdingId ?? 0);
	const historicalCards = historyQuery.data ?? [];

	// Invalid id parsed to 0 — short-circuit to the not-found branch before
	// the hooks resolve. The detailQuery is disabled in this case so it
	// will never produce data.
	if (cardId === 0) {
		return (
			<PageContainer title={t("decisionCard.detailTitle")}>
				<Alert
					type="warning"
					showIcon
					message={t("decisionCard.notFound.title")}
					description={t("decisionCard.notFound.description")}
					data-testid="detail-not-found"
				/>
			</PageContainer>
		);
	}

	if (detailQuery.isLoading) {
		return (
			<PageContainer title={t("decisionCard.detailTitle")}>
				<Skeleton active paragraph={{ rows: 12 }} />
			</PageContainer>
		);
	}

	if (detailQuery.error) {
		return (
			<PageContainer title={t("decisionCard.detailTitle")}>
				<Alert
					type="error"
					showIcon
					message={t("decisionCard.loadError.title")}
					description={t("decisionCard.loadError.description")}
					data-testid="detail-error"
				/>
			</PageContainer>
		);
	}

	if (!card) {
		return (
			<PageContainer title={t("decisionCard.detailTitle")}>
				<Alert
					type="warning"
					showIcon
					message={t("decisionCard.notFound.title")}
					description={t("decisionCard.notFound.deleted")}
					data-testid="detail-not-found"
				/>
			</PageContainer>
		);
	}

	return (
		<PageContainer
			title={card.assetName}
			subTitle={formatAnalysisTime(card.analyzedAt)}
			breadcrumb={{
				items: [
					{ title: <Link to="/briefing">{t("nav.briefing", { ns: "common" })}</Link> },
					{ title: card.assetName },
				],
			}}
			data-testid="decision-card-detail"
		>
			<Row gutter={[16, 16]}>
				<Col xs={24} lg={18}>
					<Space direction="vertical" size={16} style={{ width: "100%" }}>
						<CardHero card={card} />
						<ConclusionBanner card={card} prevCard={prevCard} />
						<ExecutionPlanFull
							execution={card.recommendation.execution}
							positionAmountCny={card.positionAmount}
							positionRatioPct={card.positionRatio}
						/>
						<DimensionReasoning card={card} prevCard={prevCard} />
						<MainRisks riskWarnings={card.riskWarnings} />
					</Space>
				</Col>
				<Col xs={24} lg={6}>
					<MetaSidebar
						card={card}
						historicalCards={historicalCards}
						onSelectHistory={(historyCardId) => navigate(`/decision-cards/${historyCardId}`)}
					/>
				</Col>
			</Row>
		</PageContainer>
	);
}
