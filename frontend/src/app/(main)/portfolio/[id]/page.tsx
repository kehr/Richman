"use client";

import { HoldingForm, TradeRecordList, useHoldings } from "@/features/portfolio";
import { Card, PageContainer, Skeleton } from "@/ui-kit/eat";
import { useRouter } from "next/navigation";
import { use } from "react";

interface HoldingDetailPageProps {
	params: Promise<{ id: string }>;
}

export default function HoldingDetailPage({ params }: HoldingDetailPageProps) {
	const { id } = use(params);
	const router = useRouter();
	const holdingId = Number(id);
	const { data: holdings, isLoading } = useHoldings();

	const holding = holdings?.find((h) => h.holdingId === holdingId);

	if (isLoading) {
		return (
			<PageContainer title="Edit Holding">
				<Skeleton active />
			</PageContainer>
		);
	}

	if (!holding) {
		return (
			<PageContainer title="Edit Holding">
				<Card>Holding not found</Card>
			</PageContainer>
		);
	}

	return (
		<PageContainer title={`Edit: ${holding.assetName}`}>
			<Card title="Holding Details" style={{ marginBottom: 16 }}>
				<HoldingForm initialValues={holding} onSuccess={() => router.push("/portfolio")} />
			</Card>
			<TradeRecordList holdingId={holdingId} />
		</PageContainer>
	);
}
