import type { BriefingCardDto } from "@/features/research-briefing";
import type { FeedbackRating } from "@/features/user-feedback";
import { Col, Row, Skeleton } from "@/ui-kit/eat";
import { BriefingCard } from "./briefing-card";
import type { BriefingViewMode } from "./briefing-header";

interface BriefingCardListProps {
	cards: BriefingCardDto[];
	viewMode: BriefingViewMode;
	isLoading: boolean;
	onCardClick: (card: BriefingCardDto) => void;
	onFeedback: (card: BriefingCardDto, rating: FeedbackRating) => void;
	feedbackPendingId?: number;
}

// BriefingCardList renders the responsive grid of briefing cards (TRD SS6.1).
// Uses a 3-column layout on desktop, 2-column on tablet, 1-column on mobile.
export function BriefingCardList({
	cards,
	viewMode,
	isLoading,
	onCardClick,
	onFeedback,
	feedbackPendingId,
}: BriefingCardListProps) {
	if (isLoading) {
		return (
			<Row gutter={[16, 16]}>
				{[1, 2, 3].map((i) => (
					<Col key={i} xs={24} sm={12} lg={8}>
						<Skeleton active />
					</Col>
				))}
			</Row>
		);
	}

	return (
		<Row gutter={[16, 16]}>
			{cards.map((card) => (
				<Col key={card.holdingId} xs={24} sm={12} lg={8}>
					<BriefingCard
						card={card}
						viewMode={viewMode}
						onClick={() => onCardClick(card)}
						onFeedback={(rating) => onFeedback(card, rating)}
						feedbackPending={feedbackPendingId === card.holdingId}
					/>
				</Col>
			))}
		</Row>
	);
}
