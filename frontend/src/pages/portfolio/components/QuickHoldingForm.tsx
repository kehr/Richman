import { Form, type FormInstance, InputNumber, Typography } from "@/ui-kit/eat";

// QuickHoldingForm is the "快速" tab body of the AddHoldingDrawer (PRD §4.2).
// It collects the average cost price and current position ratio for a
// previously-picked asset. The form instance is owned by the parent drawer
// so it can trigger validation + submission from the drawer footer.

export interface QuickHoldingFormValues {
	costPrice: number;
	positionRatio: number;
}

interface QuickHoldingFormProps {
	form: FormInstance<QuickHoldingFormValues>;
	onFinish: (values: QuickHoldingFormValues) => void;
}

export function QuickHoldingForm({ form, onFinish }: QuickHoldingFormProps) {
	return (
		<Form form={form} layout="vertical" onFinish={onFinish} data-testid="quick-holding-form">
			<Typography.Paragraph type="secondary">
				适合已经买好一段时间，直接填综合成本。
			</Typography.Paragraph>

			<Form.Item
				label="均价成本"
				name="costPrice"
				rules={[{ required: true, message: "请输入均价成本" }]}
			>
				<InputNumber
					min={0}
					step={0.01}
					style={{ width: "100%" }}
					placeholder="如 1.234"
					addonBefore="¥"
				/>
			</Form.Item>

			<Form.Item
				label="当前仓位比例"
				name="positionRatio"
				rules={[
					{ required: true, message: "请输入仓位比例" },
					{
						type: "number",
						min: 0,
						max: 100,
						message: "仓位比例应在 0-100 之间",
					},
				]}
			>
				<InputNumber
					min={0}
					max={100}
					step={1}
					style={{ width: "100%" }}
					placeholder="如 20"
					addonAfter="%"
				/>
			</Form.Item>
		</Form>
	);
}
