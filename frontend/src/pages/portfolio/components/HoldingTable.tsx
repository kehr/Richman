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

// HoldingTable renders the seven-column holdings table from PRD §4.1.
// Columns: 标的 / 类型 / 成本 / 现价 / 仓位 (with amount) / 浮盈亏 (with amount) /
// 操作. Row click anywhere outside the action column triggers onRowClick so
// the parent page can navigate to the most recent decision card for that
// holding. Money columns delegate to useMoney so capital-aware amounts are
// rendered consistently with the rest of the app.

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
	const money = useMoney();
	const { data: settings } = useUserSettings();
	const totalCapital = settings?.totalCapitalCny ?? null;

	const computeAmount = (positionRatioPct: number): number | null => {
		if (totalCapital == null) return null;
		// positionRatio in HoldingDto is a percent value (0..100) — matches
		// the dashboard aggregate and decision card consumers. Divide by 100
		// before multiplying by total capital to get the CNY amount.
		return Math.round((totalCapital * positionRatioPct) / 100);
	};

	const columns = [
		{
			title: "标的",
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
			title: "类型",
			dataIndex: "assetType",
			key: "assetType",
			width: 120,
			render: (value: string) => <Tag>{value}</Tag>,
		},
		{
			title: "成本",
			dataIndex: "costPrice",
			key: "costPrice",
			width: 120,
			render: (value: number) => `¥${value.toFixed(2)}`,
		},
		{
			title: "现价",
			key: "currentPrice",
			width: 120,
			render: () => <Typography.Text type="secondary">--</Typography.Text>,
		},
		{
			title: "仓位",
			key: "positionRatio",
			width: 160,
			render: (_: unknown, record: HoldingDto) => {
				// positionRatio is already a percent (0..100); useMoney handles the
				// capital-aware formatting so the row degrades gracefully when total
				// capital is not configured.
				return money.format(record.positionRatio, computeAmount(record.positionRatio));
			},
		},
		{
			title: "浮盈亏",
			key: "pnl",
			width: 160,
			render: () => <Typography.Text type="secondary">--</Typography.Text>,
		},
		{
			title: "操作",
			key: "actions",
			width: 220,
			render: (_: unknown, record: HoldingDto) => (
				<Space
					size="small"
					onClick={(e) => e.stopPropagation()}
					data-testid={`holding-actions-${record.holdingId}`}
				>
					<Button type="link" size="small" icon={<EditOutlined />} onClick={() => onEdit?.(record)}>
						编辑
					</Button>
					<Button type="link" size="small" onClick={() => onTransactions?.(record)}>
						交易记录
					</Button>
					<Popconfirm
						title="删除该持仓?"
						okText="删除"
						cancelText="取消"
						okType="danger"
						onConfirm={() => onDelete?.(record)}
					>
						<Button type="link" size="small" danger icon={<DeleteOutlined />}>
							删除
						</Button>
					</Popconfirm>
				</Space>
			),
		},
	];

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
