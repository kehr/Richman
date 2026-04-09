import { Form, type FormInstance, InputNumber, Typography } from "@/ui-kit/eat";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

// QuickHoldingForm is the "quick" tab body of the AddHoldingDrawer (PRD §4.2).
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
	const { t } = useTranslation("app");

	// Memoize rules so they stay reactive when the locale changes.
	const rules = useMemo(
		() => ({
			costPrice: [
				{ required: true, message: t("portfolio.quickHoldingForm.validation.costPriceRequired") },
			],
			positionRatio: [
				{
					required: true,
					message: t("portfolio.quickHoldingForm.validation.positionRatioRequired"),
				},
				{
					type: "number" as const,
					min: 0,
					max: 100,
					message: t("portfolio.quickHoldingForm.validation.positionRatioRange"),
				},
			],
		}),
		[t],
	);

	return (
		<Form form={form} layout="vertical" onFinish={onFinish} data-testid="quick-holding-form">
			<Typography.Paragraph type="secondary">
				{t("portfolio.quickHoldingForm.hint")}
			</Typography.Paragraph>

			<Form.Item
				label={t("portfolio.quickHoldingForm.costPrice")}
				name="costPrice"
				rules={rules.costPrice}
			>
				<InputNumber
					min={0}
					step={0.01}
					style={{ width: "100%" }}
					placeholder={t("portfolio.quickHoldingForm.costPricePlaceholder")}
					addonBefore="¥"
				/>
			</Form.Item>

			<Form.Item
				label={t("portfolio.quickHoldingForm.positionRatio")}
				name="positionRatio"
				rules={rules.positionRatio}
			>
				<InputNumber
					min={0}
					max={100}
					step={1}
					style={{ width: "100%" }}
					placeholder={t("portfolio.quickHoldingForm.positionRatioPlaceholder")}
					addonAfter="%"
				/>
			</Form.Item>
		</Form>
	);
}
