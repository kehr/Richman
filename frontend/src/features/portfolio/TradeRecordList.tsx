import type { DayjsLike } from "@/domain/datetime/dayjs-like";
import { formatCurrency, formatDate } from "@/domain/ui/format";
import {
	Button,
	DatePicker,
	Form,
	InputNumber,
	Modal,
	ProTable,
	Radio,
	Space,
	Tag,
	message,
} from "@/ui-kit/eat";
import { PlusOutlined } from "@/ui-kit/eat";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
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
	const { t, i18n } = useTranslation("app");

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
			message.success(t("portfolio.tradeRecordList.saveSuccess"));
			setModalOpen(false);
			form.resetFields();
		} catch {
			message.error(t("portfolio.tradeRecordList.saveError"));
		}
	};

	const directionOptions = useMemo(
		() => [
			{ value: "buy", label: t("portfolio.addTransactionDrawer.buy") },
			{ value: "sell", label: t("portfolio.addTransactionDrawer.sell") },
		],
		[t],
	);

	const columns = [
		{
			title: t("portfolio.transactionTable.direction"),
			dataIndex: "direction",
			key: "direction",
			render: (_: unknown, record: Trade) => (
				<Tag color={record.direction === "buy" ? "green" : "red"}>
					{record.direction === "buy"
						? t("portfolio.addTransactionDrawer.buy")
						: t("portfolio.addTransactionDrawer.sell")}
				</Tag>
			),
		},
		{
			title: t("portfolio.transactionTable.price"),
			dataIndex: "price",
			key: "price",
			render: (_: unknown, record: Trade) => formatCurrency(record.price, i18n.language),
		},
		{
			title: t("portfolio.transactionTable.quantity"),
			dataIndex: "quantity",
			key: "quantity",
		},
		{
			title: t("portfolio.transactionTable.time"),
			dataIndex: "tradedAt",
			key: "tradedAt",
			render: (_: unknown, record: Trade) => formatDate(record.tradedAt, i18n.language, "datetime"),
		},
	];

	return (
		<>
			<ProTable<Trade>
				headerTitle={t("portfolio.tradeRecordList.headerTitle")}
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
						{t("portfolio.tradeRecordList.addButton")}
					</Button>,
				]}
				pagination={{ pageSize: 10 }}
			/>

			<Modal
				title={t("portfolio.addTransactionDrawer.title")}
				open={modalOpen}
				onCancel={() => setModalOpen(false)}
				footer={null}
			>
				<Form form={form} layout="vertical" onFinish={handleCreateTrade}>
					<Form.Item
						label={t("portfolio.addTransactionDrawer.direction")}
						name="direction"
						rules={[
							{
								required: true,
								message: t("portfolio.addTransactionDrawer.validation.directionRequired"),
							},
						]}
					>
						<Radio.Group options={directionOptions} />
					</Form.Item>
					<Form.Item
						label={t("portfolio.addTransactionDrawer.price")}
						name="price"
						rules={[
							{
								required: true,
								message: t("portfolio.addTransactionDrawer.validation.priceRequired"),
							},
						]}
					>
						<InputNumber min={0} step={0.01} style={{ width: "100%" }} />
					</Form.Item>
					<Form.Item
						label={t("portfolio.addTransactionDrawer.quantity")}
						name="quantity"
						rules={[
							{
								required: true,
								message: t("portfolio.addTransactionDrawer.validation.quantityRequired"),
							},
						]}
					>
						<InputNumber min={1} step={1} style={{ width: "100%" }} />
					</Form.Item>
					<Form.Item
						label={t("portfolio.addTransactionDrawer.time")}
						name="tradedAt"
						rules={[
							{
								required: true,
								message: t("portfolio.addTransactionDrawer.validation.timeRequired"),
							},
						]}
					>
						<DatePicker showTime style={{ width: "100%" }} />
					</Form.Item>
					<Space>
						<Button type="primary" htmlType="submit" loading={createTrade.isPending}>
							{t("action.submit", { ns: "common" })}
						</Button>
						<Button onClick={() => setModalOpen(false)}>
							{t("action.cancel", { ns: "common" })}
						</Button>
					</Space>
				</Form>
			</Modal>
		</>
	);
}
