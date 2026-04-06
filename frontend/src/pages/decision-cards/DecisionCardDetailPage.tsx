import { RiskDisclaimer } from "@/components/RiskDisclaimer";
import { DecisionCardSummary, useDecisionCardDetail } from "@/features/decision-card";
import { PageContainer, Skeleton } from "@/ui-kit/eat";
import { useParams } from "react-router";

// Placeholder detail page wired to the new decision-card feature. Step 15
// replaces this with the full reasoning view (three-dimension breakdown,
// weights, risk warnings). Until then the summary card keeps the route
// functional and gives the dashboard a valid navigation target.
export default function DecisionCardDetailPage() {
	const { id } = useParams<{ id: string }>();
	const cardId = Number(id);
	const { data: card, isLoading } = useDecisionCardDetail(cardId);

	return (
		<PageContainer title="Decision Card Detail" footer={[<RiskDisclaimer key="disclaimer" />]}>
			{isLoading ? (
				<Skeleton active paragraph={{ rows: 8 }} />
			) : card ? (
				<DecisionCardSummary card={card} />
			) : (
				<div>Card not found</div>
			)}
		</PageContainer>
	);
}
