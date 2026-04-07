import { type CreateTradeInput, useCreateHoldingTrade } from "@/features/portfolio";
import { Button, DatePicker, Drawer, Flex, Form, InputNumber, Radio, message } from "@/ui-kit/eat";

// AddTransactionDrawer is the right-side drawer used to record a new trade
// for a single holding (PRD §4.4). The form mirrors the backend
// CreateTradeInput shape directly so the component does not have to keep an
// intermediate model.

interface AddTransactionDrawerProps {
	open: boolean;
	holdingId: number;
	onClose: () => void;
}

interface FormValues {
	tradedAt?: { toISOString: () => string } | null;
	direction: "buy" | "sell";
	price: number;
	quantity: number;
}

export function AddTransactionDrawer({ open, holdingId, onClose }: AddTransactionDrawerProps) {
	const [form] = Form.useForm<FormValues>();
	const createTrade = useCreateHoldingTrade(holdingId);

	const handleClose = () => {
		form.resetFields();
		onClose();
	};

	const handleFinish = async (values: FormValues) => {
		const tradedAt =
			values.tradedAt && typeof values.tradedAt.toISOString === "function"
				? values.tradedAt.toISOString()
				: new Date().toISOString();
		const payload: CreateTradeInput = {
			direction: values.direction,
			price: values.price,
			quantity: values.quantity,
			tradedAt,
		};
		try {
			await createTrade.mutateAsync(payload);
			message.success("交易已记录");
			handleClose();
		} catch {
			message.error("记录交易失败");
		}
	};

	return (
		<Drawer
			title="记录交易"
			open={open}
			onClose={handleClose}
			placement="right"
			width={480}
			data-testid="add-transaction-drawer"
			footer={
				<Flex justify="flex-end" gap={8}>
					<Button onClick={handleClose}>取消</Button>
					<Button
						type="primary"
						loading={createTrade.isPending}
						onClick={() => form.submit()}
						data-testid="add-transaction-save"
					>
						保存
					</Button>
				</Flex>
			}
		>
			<Form<FormValues>
				form={form}
				layout="vertical"
				initialValues={{ direction: "buy" }}
				onFinish={handleFinish}
				data-testid="add-transaction-form"
			>
				<Form.Item
					label="方向"
					name="direction"
					rules={[{ required: true, message: "请选择交易方向" }]}
				>
					<Radio.Group>
						<Radio.Button value="buy">买入</Radio.Button>
						<Radio.Button value="sell">卖出</Radio.Button>
					</Radio.Group>
				</Form.Item>
				<Form.Item
					label="时间"
					name="tradedAt"
					rules={[{ required: true, message: "请选择交易时间" }]}
				>
					<DatePicker showTime style={{ width: "100%" }} />
				</Form.Item>
				<Form.Item label="价格" name="price" rules={[{ required: true, message: "请输入价格" }]}>
					<InputNumber min={0} step={0.01} style={{ width: "100%" }} addonBefore="¥" />
				</Form.Item>
				<Form.Item label="数量" name="quantity" rules={[{ required: true, message: "请输入数量" }]}>
					<InputNumber min={0} step={1} style={{ width: "100%" }} />
				</Form.Item>
			</Form>
		</Drawer>
	);
}
