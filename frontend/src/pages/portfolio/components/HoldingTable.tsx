import { formatPercent } from "@/domain/money/format";
import { useMoney } from "@/domain/money/useMoney";
import type { HoldingDto } from "@/features/portfolio";
import { useUserSettings } from "@/features/user-settings";
import {
	Button,
	DeleteOutlined,
	EditOutlined,
	Popconfirm,
	Space,
	Table,
	Tag,
	Typography,
} from "@/ui-kit/eat";
import { History } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

// HoldingTable renders the seven-column holdings table from PRD §4.1.
// Columns: asset / type / cost / current price / position (with amount) /
// P&L (with amount) / actions. Row click anywhere outside the action column
// triggers onRowClick so the parent page can navigate to the most recent
// decision card for that holding.

interface HoldingTableProps {
	holdings: HoldingDto[];
	loading?: boolean;
	onRowClick?: (holding: HoldingDto) => void;
	onEdit?: (holding: HoldingDto) => void;
	onTransactions?: (holding: HoldingDto) => void;
	onDelete?: (holding: HoldingDto) => void;
}

export function HoldingTable({
	holdings,
	loading,
	onRowClick,
	onEdit,
	onTransactions,
	onDelete,
}: HoldingTableProps) {
	const { t } = useTranslation("app");
	const money = useMoney();
	const { data: settings } = useUserSettings();
	const totalCapital = settings?.totalCapitalCny ?? null;

	const columns = useMemo(() => {
		const computeAmount = (positionRatioPct: number): number | null => {
			if (totalCapital == null) return null;
			return Math.round((totalCapital * positionRatioPct) / 100);
		};
		return [
			{
				title: t("portfolio.holdingTable.asset"),
				key: "asset",
				render: (_: unknown, record: HoldingDto) => (
					<Space direction="vertical" size={0}>
						<Typography.Text strong>{record.assetName}</Typography.Text>
						<Typography.Text type="secondary" style={{ fontSize: 12 }}>
							{record.assetCode}
						</Typography.Text>
					</Space>
				),
			},
			{
				title: t("portfolio.holdingTable.type"),
				dataIndex: "assetType",
				key: "assetType",
				width: 120,
				render: (value: string) => <Tag>{value}</Tag>,
			},
			{
				title: t("portfolio.holdingTable.cost"),
				dataIndex: "costPrice",
				key: "costPrice",
				width: 120,
				render: (value: number) => `¥${value.toFixed(2)}`,
			},
			{
				title: t("portfolio.holdingTable.currentPrice"),
				key: "currentPrice",
				width: 120,
				render: () => <Typography.Text type="secondary">--</Typography.Text>,
			},
			{
				title: t("portfolio.holdingTable.position"),
				key: "positionRatio",
				width: 140,
				render: (_: unknown, record: HoldingDto) => {
					const amountStr = money.formatAmountOnly(computeAmount(record.positionRatio));
					return (
						<Space direction="vertical" size={0}>
							<Typography.Text>{formatPercent(record.positionRatio)}</Typography.Text>
							{amountStr != null && (
								<Typography.Text type="secondary" style={{ fontSize: 12 }}>
									{amountStr}
								</Typography.Text>
							)}
						</Space>
					);
				},
			},
			{
				title: t("portfolio.holdingTable.pnl"),
				key: "pnl",
				width: 160,
				render: () => <Typography.Text type="secondary">--</Typography.Text>,
			},
			{
				title: t("portfolio.holdingTable.actions"),
				key: "actions",
				width: 220,
				render: (_: unknown, record: HoldingDto) => (
					<Space
						size="small"
						onClick={(e) => e.stopPropagation()}
						data-testid={`holding-actions-${record.holdingId}`}
					>
						<Button
							type="link"
							size="small"
							icon={<EditOutlined />}
							onClick={() => onEdit?.(record)}
						>
							{t("portfolio.holdingTable.editButton")}
						</Button>
						<Button
							type="link"
							size="small"
							icon={<History size={14} />}
							onClick={() => onTransactions?.(record)}
						>
							{t("portfolio.holdingTable.transactionsButton")}
						</Button>
						<Popconfirm
							title={t("portfolio.holdingTable.deleteConfirm")}
							okText={t("portfolio.holdingTable.deleteButton")}
							cancelText={t("portfolio.holdingTable.deleteButton")}
							okType="danger"
							onConfirm={() => onDelete?.(record)}
						>
							<Button type="link" size="small" danger icon={<DeleteOutlined />}>
								{t("portfolio.holdingTable.deleteButton")}
							</Button>
						</Popconfirm>
					</Space>
				),
			},
		];
	}, [t, money, onEdit, onTransactions, onDelete, totalCapital]);

	return (
		<Table<HoldingDto>
			columns={columns}
			dataSource={holdings}
			rowKey="holdingId"
			loading={loading}
			pagination={false}
			data-testid="holding-table"
			onRow={(record) => ({
				onClick: () => onRowClick?.(record),
				style: { cursor: onRowClick ? "pointer" : undefined },
				"data-testid": `holding-row-${record.holdingId}`,
			})}
		/>
	);
}
