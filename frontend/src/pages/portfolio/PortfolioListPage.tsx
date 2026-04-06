import { formatCurrency, formatPercent } from "@/domain/ui/format";
import { useDeleteHolding, useHoldings } from "@/features/portfolio";
import type { HoldingDto } from "@/features/portfolio";
import {
	Button,
	DeleteOutlined,
	EditOutlined,
	PageContainer,
	PlusOutlined,
	Popconfirm,
	ProTable,
	Space,
	Tag,
	message,
} from "@/ui-kit/eat";
import { useNavigate } from "react-router";

export default function PortfolioListPage() {
	const navigate = useNavigate();
	const { data: holdings, isLoading } = useHoldings();
	const deleteMutation = useDeleteHolding();

	const handleDelete = async (id: number) => {
		try {
			await deleteMutation.mutateAsync(id);
			message.success("Holding deleted");
		} catch {
			message.error("Failed to delete holding");
		}
	};

	const columns = [
		{
			title: "Asset Code",
			dataIndex: "assetCode",
			key: "assetCode",
			width: 120,
		},
		{
			title: "Asset Name",
			dataIndex: "assetName",
			key: "assetName",
		},
		{
			title: "Type",
			dataIndex: "assetType",
			key: "assetType",
			width: 100,
			render: (_: unknown, record: HoldingDto) => <Tag>{record.assetType}</Tag>,
		},
		{
			title: "Cost Price",
			dataIndex: "costPrice",
			key: "costPrice",
			width: 120,
			render: (_: unknown, record: HoldingDto) => formatCurrency(record.costPrice),
		},
		{
			title: "Position Ratio",
			dataIndex: "positionRatio",
			key: "positionRatio",
			width: 120,
			render: (_: unknown, record: HoldingDto) => formatPercent(record.positionRatio),
		},
		{
			title: "Quantity",
			dataIndex: "quantity",
			key: "quantity",
			width: 100,
		},
		{
			title: "Actions",
			key: "actions",
			width: 160,
			render: (_: unknown, record: HoldingDto) => (
				<Space>
					<Button
						type="link"
						size="small"
						icon={<EditOutlined />}
						onClick={() => navigate(`/portfolio/${record.holdingId}`)}
					>
						Edit
					</Button>
					<Popconfirm title="Delete this holding?" onConfirm={() => handleDelete(record.holdingId)}>
						<Button type="link" size="small" danger icon={<DeleteOutlined />}>
							Delete
						</Button>
					</Popconfirm>
				</Space>
			),
		},
	];

	return (
		<PageContainer title="Portfolio">
			<ProTable<HoldingDto>
				columns={columns}
				dataSource={holdings}
				rowKey="holdingId"
				loading={isLoading}
				search={false}
				toolBarRender={() => [
					<Button
						key="add"
						type="primary"
						icon={<PlusOutlined />}
						onClick={() => navigate("/portfolio/new")}
					>
						Add Holding
					</Button>,
				]}
				pagination={{ pageSize: 20 }}
			/>
		</PageContainer>
	);
}
