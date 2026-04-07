import type { Trade } from "@/features/portfolio";
import {
	Button,
	DeleteOutlined,
	EditOutlined,
	Space,
	Table,
	Tag,
	Tooltip,
	Typography,
} from "@/ui-kit/eat";

// TransactionTable renders the per-holding trade history for the
// transactions sub-page (PRD §4.4). The edit and delete columns are present
// to keep the row layout stable, but both actions are disabled today: the
// backend exposes GET + POST /holdings/:id/trades but no PATCH or DELETE,
// and we deliberately avoid a frontend-only soft delete here.

interface TransactionTableProps {
	trades: Trade[];
	loading?: boolean;
}

function formatDateTime(iso: string): string {
	try {
		const d = new Date(iso);
		const pad = (n: number) => n.toString().padStart(2, "0");
		return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(
			d.getHours(),
		)}:${pad(d.getMinutes())}`;
	} catch {
		return iso;
	}
}

export function TransactionTable({ trades, loading }: TransactionTableProps) {
	const columns = [
		{
			title: "时间",
			dataIndex: "tradedAt",
			key: "tradedAt",
			width: 180,
			render: (value: string) => <Typography.Text>{formatDateTime(value)}</Typography.Text>,
		},
		{
			title: "价格",
			dataIndex: "price",
			key: "price",
			width: 140,
			render: (value: number) => `¥${value.toFixed(2)}`,
		},
		{
			title: "数量",
			dataIndex: "quantity",
			key: "quantity",
			width: 140,
			render: (value: number) => value.toString(),
		},
		{
			title: "方向",
			dataIndex: "direction",
			key: "direction",
			width: 100,
			render: (value: Trade["direction"]) =>
				value === "buy" ? <Tag color="red">买入</Tag> : <Tag color="green">卖出</Tag>,
		},
		{
			title: "操作",
			key: "actions",
			width: 200,
			render: (_: unknown, record: Trade) => (
				<Space size="small" data-testid={`trade-actions-${record.tradeId}`}>
					<Tooltip title="编辑接口待后端补齐">
						<Button type="link" size="small" icon={<EditOutlined />} disabled>
							编辑
						</Button>
					</Tooltip>
					<Tooltip title="删除接口待后端补齐">
						<Button
							type="link"
							size="small"
							danger
							icon={<DeleteOutlined />}
							disabled
							data-testid={`trade-delete-${record.tradeId}`}
						>
							删除
						</Button>
					</Tooltip>
				</Space>
			),
		},
	];

	return (
		<Table<Trade>
			columns={columns}
			dataSource={trades}
			rowKey="tradeId"
			loading={loading}
			pagination={false}
			data-testid="transaction-table"
		/>
	);
}
