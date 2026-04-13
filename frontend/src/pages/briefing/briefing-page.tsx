import { StorageKeys } from "@/domain/storage/local-storage";
import { useLocalStorage } from "@/domain/storage/use-local-storage";
import { useHoldings } from "@/features/portfolio";
import type { BriefingCardDto } from "@/features/research-briefing";
import { useBriefing } from "@/features/research-briefing";
import { useSubmitFeedback } from "@/features/user-feedback";
import type { FeedbackRating } from "@/features/user-feedback";
import { App, Flex, PageContainer, Typography } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { BriefingCardList } from "./components/briefing-card-list";
import { BriefingHeader } from "./components/briefing-header";
import type { BriefingViewMode } from "./components/briefing-header";
import { EmptyBriefingState } from "./components/empty-briefing-state";

// BriefingPage is the composition root for the research briefing view (TRD SS6).
// It replaces the old DashboardPage: the route /briefing still points here but
// the content is now a card-based briefing digest rather than a decision-card wall.
export default function BriefingPage() {
	const navigate = useNavigate();
	const { t } = useTranslation(["app", "common"]);
	const { message } = App.useApp();

	// Persist view mode preference across sessions (TRD SS6.3).
	const [viewMode, setViewMode] = useLocalStorage<BriefingViewMode>(
		StorageKeys.briefingViewMode,
		"compact",
	);

	const holdingsQuery = useHoldings();
	const briefingQuery = useBriefing();
	const feedbackMutation = useSubmitFeedback();
	const [feedbackPendingId, setFeedbackPendingId] = useState<number | undefined>(undefined);

	const holdings = holdingsQuery.data ?? [];
	const briefing = briefingQuery.data;
	const cards: BriefingCardDto[] = briefing?.cards ?? [];

	const holdingsReady = !holdingsQuery.isLoading;
	const showEmptyState = holdingsReady && holdings.length === 0;

	const handleAddHolding = () => {
		navigate("/portfolio");
	};

	const handleCardClick = (card: BriefingCardDto) => {
		// Navigate to the asset detail page, defaulting to the execution tab
		// per TRD SS6.2: "click navigates to /market/:code (execution tab)".
		navigate(`/market/${card.assetCode}?tab=execution`);
	};

	const handleFeedback = async (card: BriefingCardDto, rating: FeedbackRating) => {
		setFeedbackPendingId(card.holdingId);
		try {
			await feedbackMutation.mutateAsync({
				target: "briefing_card",
				targetId: card.holdingId,
				rating,
			});
		} catch {
			message.error(t("briefing.feedback.error"));
		} finally {
			setFeedbackPendingId(undefined);
		}
	};

	return (
		<PageContainer title={false} data-testid="briefing-page">
			<Flex vertical gap={16}>
				{!showEmptyState && (
					<BriefingHeader
						viewMode={viewMode}
						onViewModeChange={setViewMode}
						generatedAt={briefing?.generatedAt}
					/>
				)}

				{showEmptyState ? (
					<EmptyBriefingState onAddHolding={handleAddHolding} />
				) : (
					<BriefingCardList
						cards={cards}
						viewMode={viewMode}
						isLoading={briefingQuery.isLoading}
						onCardClick={handleCardClick}
						onFeedback={handleFeedback}
						feedbackPendingId={feedbackPendingId}
					/>
				)}

				{/* Risk disclaimer footer (TRD SS14) */}
				{!showEmptyState && (
					<Typography.Text
						type="secondary"
						style={{ fontSize: 11, textAlign: "center", paddingTop: 8 }}
					>
						{t("common:disclaimer.body")}
					</Typography.Text>
				)}
			</Flex>
		</PageContainer>
	);
}
