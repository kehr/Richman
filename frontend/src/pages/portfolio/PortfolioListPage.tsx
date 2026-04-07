import { useDecisionCards } from "@/features/decision-card";
import { useDeleteHolding, useHoldings } from "@/features/portfolio";
import type { HoldingDto } from "@/features/portfolio";
import {
	Button,
	CameraOutlined,
	Flex,
	PageContainer,
	PlusOutlined,
	Space,
	Tooltip,
	Typography,
	message,
} from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { AddHoldingDrawer } from "./components/AddHoldingDrawer";
import { HoldingTable } from "./components/HoldingTable";
import { TotalCapitalRow } from "./components/TotalCapitalRow";

// PortfolioListPage renders the new PRD §4.1 layout: title + counter +
// add/import buttons, total capital row, then the seven-column holding
// table. Row click on a holding navigates to the latest decision card for
// that holding when one exists, otherwise to the edit page.
//
// The screenshot import button is wired up but disabled in Step 16; Step 17
// will activate it and open the screenshot-import drawer.

const HOLDING_LIMIT = 5;

export default function PortfolioListPage() {
	const navigate = useNavigate();
	const { data: holdings, isLoading } = useHoldings();
	const { data: decisionCards } = useDecisionCards();
	const deleteMutation = useDeleteHolding();
	const [drawerOpen, setDrawerOpen] = useState(false);

	const holdingsList = holdings ?? [];
	const count = holdingsList.length;
	const atLimit = count >= HOLDING_LIMIT;

	// Build a holdingId -> latest decision card lookup so row click can jump
	// straight into the detail page for the most recent analysis. The server
	// already returns the latest card per holding (useDecisionCards contract)
	// so a simple Map is sufficient.
	const latestCardByHolding = useMemo(() => {
		const map = new Map<number, number>();
		for (const card of decisionCards ?? []) {
			if (!map.has(card.holdingId)) {
				map.set(card.holdingId, card.cardId);
			}
		}
		return map;
	}, [decisionCards]);

	const handleRowClick = (holding: HoldingDto) => {
		const cardId = latestCardByHolding.get(holding.holdingId);
		if (cardId != null) {
			navigate(`/decision-cards/${cardId}`);
		} else {
			navigate(`/portfolio/${holding.holdingId}`);
		}
	};

	const handleEdit = (holding: HoldingDto) => {
		navigate(`/portfolio/${holding.holdingId}`);
	};

	const handleTransactions = (holding: HoldingDto) => {
		navigate(`/portfolio/${holding.holdingId}/transactions`);
	};

	const handleDelete = async (holding: HoldingDto) => {
		try {
			await deleteMutation.mutateAsync(holding.holdingId);
			message.success("持仓已删除");
		} catch {
			message.error("删除持仓失败");
		}
	};

	const addButton = (
		<Button
			key="add"
			type="primary"
			icon={<PlusOutlined />}
			disabled={atLimit}
			onClick={() => setDrawerOpen(true)}
			data-testid="add-holding-button"
		>
			添加持仓
		</Button>
	);

	return (
		<PageContainer
			title={null}
			header={{ title: null, breadcrumb: {} }}
			data-testid="portfolio-list-page"
		>
			<Flex
				align="center"
				justify="space-between"
				style={{ marginBottom: 12 }}
				data-testid="portfolio-header"
			>
				<Space align="baseline" size="middle">
					<Typography.Title level={3} style={{ margin: 0 }}>
						我的持仓
					</Typography.Title>
					<Typography.Text type="secondary" data-testid="holding-counter">
						{count}/{HOLDING_LIMIT} 个持仓
					</Typography.Text>
				</Space>
				<Space>
					{atLimit ? <Tooltip title="MVP 最多 5 个标的">{addButton}</Tooltip> : addButton}
					<Tooltip title="即将推出（Step 17）">
						<Button icon={<CameraOutlined />} disabled data-testid="screenshot-import-button">
							截图批量导入
						</Button>
					</Tooltip>
				</Space>
			</Flex>

			<TotalCapitalRow />

			<HoldingTable
				holdings={holdingsList}
				loading={isLoading}
				onRowClick={handleRowClick}
				onEdit={handleEdit}
				onTransactions={handleTransactions}
				onDelete={handleDelete}
			/>

			<AddHoldingDrawer open={drawerOpen} onClose={() => setDrawerOpen(false)} />
		</PageContainer>
	);
}
