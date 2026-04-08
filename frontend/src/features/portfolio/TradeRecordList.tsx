import type { DayjsLike } from "@/domain/datetime/dayjs-like";
import { formatCurrency, formatDate } from "@/domain/ui/format";
import {
	Button,
	DatePicker,
	Form,
	InputNumber,
	Modal,
	ProTable,
	Select,
	Space,
	Tag,
	message,
} from "@/ui-kit/eat";
import { PlusOutlined } from "@/ui-kit/eat";
import { useState } from "react";
import type { Trade, TradeDirection } from "./trade-types";
import { useCreateTrade, useTrades } from "./usePortfolio";

interface TradeRecordListProps {
	holdingId: number;
}

export function TradeRecordList({ holdingId }: TradeRecordListProps) {
	const { data: trades, isLoading } = useTrades(holdingId);
	const createTrade = useCreateTrade(holdingId);
	const [modalOpen, setModalOpen] = useState(false);
	const [form] = Form.useForm();

	const handleCreateTrade = async (values: Record<string, unknown>) => {
		try {
			// tradedAt is enforced as required by the antd Form rule below, so
			// we rely on validation to reject empty submissions and convert the
			// Dayjs value straight to an ISO string.
			const tradedAt = (values.tradedAt as DayjsLike).toDate().toISOString();
			await createTrade.mutateAsync({
				direction: values.direction as TradeDirection,
				price: values.price as number,
				quantity: values.quantity as number,
				tradedAt,
			});
			message.success("Trade recorded");
			setModalOpen(false);
			form.resetFields();
		} catch {
			message.error("Failed to record trade");
		}
	};

	const columns = [
		{
			title: "Direction",
			dataIndex: "direction",
			key: "direction",
			render: (_: unknown, record: Trade) => (
				<Tag color={record.direction === "buy" ? "green" : "red"}>
					{record.direction.toUpperCase()}
				</Tag>
			),
		},
		{
			title: "Price",
			dataIndex: "price",
			key: "price",
			render: (_: unknown, record: Trade) => formatCurrency(record.price),
		},
		{
			title: "Quantity",
			dataIndex: "quantity",
			key: "quantity",
		},
		{
			title: "Traded At",
			dataIndex: "tradedAt",
			key: "tradedAt",
			render: (_: unknown, record: Trade) => formatDate(record.tradedAt, "datetime"),
		},
	];

	return (
		<>
			<ProTable<Trade>
				headerTitle="Trade Records"
				columns={columns}
				dataSource={trades}
				rowKey="tradeId"
				loading={isLoading}
				search={false}
				toolBarRender={() => [
					<Button
						key="add"
						type="primary"
						icon={<PlusOutlined />}
						onClick={() => setModalOpen(true)}
					>
						Add Trade
					</Button>,
				]}
				pagination={{ pageSize: 10 }}
			/>

			<Modal
				title="Record Trade"
				open={modalOpen}
				onCancel={() => setModalOpen(false)}
				footer={null}
			>
				<Form form={form} layout="vertical" onFinish={handleCreateTrade}>
					<Form.Item
						label="Direction"
						name="direction"
						rules={[{ required: true, message: "Please select direction" }]}
					>
						<Select
							options={[
								{ value: "buy", label: "Buy" },
								{ value: "sell", label: "Sell" },
							]}
						/>
					</Form.Item>
					<Form.Item
						label="Price"
						name="price"
						rules={[{ required: true, message: "Please enter price" }]}
					>
						<InputNumber min={0} step={0.01} style={{ width: "100%" }} />
					</Form.Item>
					<Form.Item
						label="Quantity"
						name="quantity"
						rules={[{ required: true, message: "Please enter quantity" }]}
					>
						<InputNumber min={1} step={1} style={{ width: "100%" }} />
					</Form.Item>
					<Form.Item
						label="Traded At"
						name="tradedAt"
						rules={[{ required: true, message: "Please select trade time" }]}
					>
						<DatePicker showTime style={{ width: "100%" }} />
					</Form.Item>
					<Space>
						<Button type="primary" htmlType="submit" loading={createTrade.isPending}>
							Submit
						</Button>
						<Button onClick={() => setModalOpen(false)}>Cancel</Button>
					</Space>
				</Form>
			</Modal>
		</>
	);
}
