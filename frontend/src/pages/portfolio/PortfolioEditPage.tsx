import { HoldingForm, TradeRecordList, useHoldings } from "@/features/portfolio";
import { Card, PageContainer, Skeleton } from "@/ui-kit/eat";
import { useNavigate, useParams } from "react-router";

export default function PortfolioEditPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
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
				<HoldingForm initialValues={holding} onSuccess={() => navigate("/portfolio")} />
			</Card>
			<TradeRecordList holdingId={holdingId} />
		</PageContainer>
	);
}
