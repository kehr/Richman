import type { DayjsLike } from "@/domain/datetime/dayjs-like";
import { type CreateTradeInput, useCreateTrade } from "@/features/portfolio";
import { Button, DatePicker, Drawer, Flex, Form, InputNumber, Radio, message } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

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
	tradedAt: DayjsLike;
	direction: "buy" | "sell";
	price: number;
	quantity: number;
}

export function AddTransactionDrawer({ open, holdingId, onClose }: AddTransactionDrawerProps) {
	const { t } = useTranslation("app");
	const [form] = Form.useForm<FormValues>();
	const createTrade = useCreateTrade(holdingId);

	const handleClose = () => {
		form.resetFields();
		onClose();
	};

	const handleFinish = async (values: FormValues) => {
		// tradedAt is enforced as required by the antd Form rule below, so we
		// rely on validation having already rejected an empty submission and
		// just convert the Dayjs value straight into an ISO string.
		const payload: CreateTradeInput = {
			direction: values.direction,
			price: values.price,
			quantity: values.quantity,
			tradedAt: values.tradedAt.toDate().toISOString(),
		};
		try {
			await createTrade.mutateAsync(payload);
			message.success(t("portfolio.addTransactionDrawer.saveSuccess"));
			handleClose();
		} catch {
			message.error(t("portfolio.addTransactionDrawer.saveError"));
		}
	};

	// Memoize rules so they stay reactive when the locale changes.
	const rules = useMemo(
		() => ({
			direction: [
				{
					required: true,
					message: t("portfolio.addTransactionDrawer.validation.directionRequired"),
				},
			],
			tradedAt: [
				{ required: true, message: t("portfolio.addTransactionDrawer.validation.timeRequired") },
			],
			price: [
				{ required: true, message: t("portfolio.addTransactionDrawer.validation.priceRequired") },
			],
			quantity: [
				{
					required: true,
					message: t("portfolio.addTransactionDrawer.validation.quantityRequired"),
				},
			],
		}),
		[t],
	);

	return (
		<Drawer
			title={t("portfolio.addTransactionDrawer.title")}
			open={open}
			onClose={handleClose}
			placement="right"
			width={480}
			data-testid="add-transaction-drawer"
			footer={
				<Flex justify="flex-end" gap={8}>
					<Button onClick={handleClose}>{t("action.cancel", { ns: "common" })}</Button>
					<Button
						type="primary"
						loading={createTrade.isPending}
						onClick={() => form.submit()}
						data-testid="add-transaction-save"
					>
						{t("action.save", { ns: "common" })}
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
					label={t("portfolio.addTransactionDrawer.direction")}
					name="direction"
					rules={rules.direction}
				>
					<Radio.Group>
						<Radio.Button value="buy">{t("portfolio.addTransactionDrawer.buy")}</Radio.Button>
						<Radio.Button value="sell">{t("portfolio.addTransactionDrawer.sell")}</Radio.Button>
					</Radio.Group>
				</Form.Item>
				<Form.Item
					label={t("portfolio.addTransactionDrawer.time")}
					name="tradedAt"
					rules={rules.tradedAt}
				>
					<DatePicker showTime style={{ width: "100%" }} />
				</Form.Item>
				<Form.Item
					label={t("portfolio.addTransactionDrawer.price")}
					name="price"
					rules={rules.price}
				>
					<InputNumber min={0} step={0.01} style={{ width: "100%" }} addonBefore="¥" />
				</Form.Item>
				<Form.Item
					label={t("portfolio.addTransactionDrawer.quantity")}
					name="quantity"
					rules={rules.quantity}
				>
					<InputNumber min={0} step={1} style={{ width: "100%" }} />
				</Form.Item>
			</Form>
		</Drawer>
	);
}
