import { useMoney } from "@/domain/money/useMoney";
import { type Trade, useHoldings, useTrades } from "@/features/portfolio";
import { useUserSettings } from "@/features/user-settings";
import {
	Breadcrumb,
	Button,
	Card,
	Col,
	LeftOutlined,
	PageContainer,
	PlusOutlined,
	Row,
	Skeleton,
	Space,
	Statistic,
	Typography,
} from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router";
import { AddTransactionDrawer } from "./components/AddTransactionDrawer";
import { TransactionTable } from "./components/TransactionTable";

// PortfolioTransactionsPage renders the per-holding transaction history
// (PRD §4.4). It is mounted at /portfolio/:id/transactions and replaces the
// previous PortfolioEditPage alias on this route.

interface TradeAggregates {
	totalBuyAmount: number;
	totalSellAmount: number;
	weightedCostPrice: number | null;
}

function aggregateTrades(trades: Trade[]): TradeAggregates {
	let buyAmount = 0;
	let sellAmount = 0;
	let buyQuantity = 0;
	for (const t of trades) {
		const amount = t.price * t.quantity;
		if (t.direction === "buy") {
			buyAmount += amount;
			buyQuantity += t.quantity;
		} else {
			sellAmount += amount;
		}
	}
	const weightedCostPrice = buyQuantity > 0 ? buyAmount / buyQuantity : null;
	return { totalBuyAmount: buyAmount, totalSellAmount: sellAmount, weightedCostPrice };
}

export default function PortfolioTransactionsPage() {
	const { id } = useParams<{ id: string }>();
	const navigate = useNavigate();
	const { t } = useTranslation("app");
	const holdingId = Number(id);
	const { data: holdings, isLoading: holdingsLoading } = useHoldings();
	const { data: trades, isLoading: tradesLoading } = useTrades(holdingId);
	const { data: settings } = useUserSettings();
	const money = useMoney();
	const [drawerOpen, setDrawerOpen] = useState(false);

	const holding = holdings?.find((h) => h.holdingId === holdingId);

	const aggregates = useMemo(() => aggregateTrades(trades ?? []), [trades]);

	if (holdingsLoading) {
		return (
			<PageContainer title={null} header={{ title: null, breadcrumb: {} }}>
				<Skeleton active />
			</PageContainer>
		);
	}

	if (!holding) {
		return (
			<PageContainer title={null} header={{ title: null, breadcrumb: {} }}>
				<Card>
					<Typography.Text>{t("portfolio.transactions.notFound")}</Typography.Text>
				</Card>
			</PageContainer>
		);
	}

	const totalCapital = settings?.totalCapitalCny ?? null;
	const positionAmount =
		totalCapital != null ? Math.round((totalCapital * holding.positionRatio) / 100) : null;

	return (
		<PageContainer title={null} header={{ title: null, breadcrumb: {} }}>
			<div data-testid="portfolio-transactions-page">
				<Space direction="vertical" size="middle" style={{ width: "100%" }}>
					<Space align="center">
						<Button
							type="link"
							icon={<LeftOutlined />}
							onClick={() => navigate("/portfolio")}
							data-testid="transactions-back-button"
						>
							{t("portfolio.transactions.back")}
						</Button>
						<Breadcrumb
							items={[
								{
									title: (
										// biome-ignore lint/a11y/useKeyWithClickEvents: breadcrumb crumb stays a span; the back button beside it provides the keyboard-accessible action
										<span style={{ cursor: "pointer" }} onClick={() => navigate("/portfolio")}>
											{t("portfolio.transactions.holdings")}
										</span>
									),
								},
								{ title: holding.assetName },
								{ title: t("portfolio.transactions.title") },
							]}
						/>
					</Space>

					<Space align="baseline">
						<Typography.Title level={4} style={{ margin: 0 }}>
							{holding.assetName}
						</Typography.Title>
						<Typography.Text type="secondary">{holding.assetCode}</Typography.Text>
					</Space>

					<Card
						title={t("portfolio.transactions.historyTitle")}
						extra={
							<Button
								type="primary"
								icon={<PlusOutlined />}
								onClick={() => setDrawerOpen(true)}
								data-testid="add-transaction-button"
							>
								{t("portfolio.transactions.addTransaction")}
							</Button>
						}
					>
						<TransactionTable trades={trades ?? []} loading={tradesLoading} />
					</Card>

					<Card title={t("portfolio.transactions.summaryTitle")} data-testid="transactions-summary">
						<Row gutter={16}>
							<Col span={6}>
								<Statistic
									title={t("portfolio.transactions.weightedCost")}
									value={money.formatAmountOnly(aggregates.weightedCostPrice) ?? "--"}
								/>
							</Col>
							<Col span={6}>
								<Statistic
									title={t("portfolio.transactions.compositePosition")}
									value={money.format(holding.positionRatio, positionAmount)}
								/>
							</Col>
							<Col span={6}>
								<Statistic
									title={t("portfolio.transactions.totalBuy")}
									value={money.formatAmountOnly(aggregates.totalBuyAmount) ?? "--"}
								/>
							</Col>
							<Col span={6}>
								<Statistic
									title={t("portfolio.transactions.totalSell")}
									value={money.formatAmountOnly(aggregates.totalSellAmount) ?? "--"}
								/>
							</Col>
						</Row>
					</Card>
				</Space>

				<AddTransactionDrawer
					open={drawerOpen}
					holdingId={holdingId}
					onClose={() => setDrawerOpen(false)}
				/>
			</div>
		</PageContainer>
	);
}
