import { HoldingForm, TradeRecordList, useHoldings } from "@/features/portfolio";
import { Card, PageContainer, Skeleton } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router";

export default function PortfolioEditPage() {
	const { t } = useTranslation("app");
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const holdingId = Number(id);
	const { data: holdings, isLoading } = useHoldings();

	const holding = holdings?.find((h) => h.holdingId === holdingId);

	if (isLoading) {
		return (
			<PageContainer title={t("portfolio.editPage.title")}>
				<Skeleton active />
			</PageContainer>
		);
	}

	if (!holding) {
		return (
			<PageContainer title={t("portfolio.editPage.title")}>
				<Card>{t("portfolio.editPage.notFound")}</Card>
			</PageContainer>
		);
	}

	return (
		<PageContainer title={t("portfolio.editPage.editTitle", { name: holding.assetName })}>
			<Card title={t("portfolio.editPage.holdingDetails")} style={{ marginBottom: 16 }}>
				<HoldingForm initialValues={holding} onSuccess={() => navigate("/portfolio")} />
			</Card>
			<TradeRecordList holdingId={holdingId} />
		</PageContainer>
	);
}
