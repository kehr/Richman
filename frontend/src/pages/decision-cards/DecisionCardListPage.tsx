import { RiskDisclaimer } from "@/components/RiskDisclaimer";
import { DecisionCardView, useLatestCards } from "@/features/decision-card";
import { Col, Empty, PageContainer, Row, Skeleton } from "@/ui-kit/eat";

export default function DecisionCardListPage() {
	const { data: cards, isLoading } = useLatestCards();

	return (
		<PageContainer title="Decision Cards" footer={[<RiskDisclaimer key="disclaimer" />]}>
			{isLoading ? (
				<Skeleton active paragraph={{ rows: 6 }} />
			) : !cards || cards.length === 0 ? (
				<Empty description="No decision cards yet. Run analysis first." />
			) : (
				<Row gutter={[16, 16]}>
					{cards.map((card) => (
						<Col key={card.cardId} xs={24} lg={12}>
							<DecisionCardView card={card} compact />
						</Col>
					))}
				</Row>
			)}
		</PageContainer>
	);
}
