"use client";

import { RiskDisclaimer } from "@/components/RiskDisclaimer";
import { DecisionCardView, useCardById } from "@/features/decision-card";
import { PageContainer, Skeleton } from "@/ui-kit/eat";
import { use } from "react";

interface DecisionCardDetailPageProps {
	params: Promise<{ id: string }>;
}

export default function DecisionCardDetailPage({ params }: DecisionCardDetailPageProps) {
	const { id } = use(params);
	const cardId = Number(id);
	const { data: card, isLoading } = useCardById(cardId);

	return (
		<PageContainer title="Decision Card Detail" footer={[<RiskDisclaimer key="disclaimer" />]}>
			{isLoading ? (
				<Skeleton active paragraph={{ rows: 8 }} />
			) : card ? (
				<DecisionCardView card={card} />
			) : (
				<div>Card not found</div>
			)}
		</PageContainer>
	);
}
