import { Form, type FormInstance, Radio, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

// PositionTier maps to a canonical positionRatio midpoint (TRD SS7.1):
//   light  (<10%)  -> midpoint 5%
//   medium (10-25%) -> midpoint 17.5%
//   heavy  (>25%)  -> midpoint 30%
export type PositionTier = "light" | "medium" | "heavy";

export const POSITION_TIER_RATIOS: Record<PositionTier, number> = {
	light: 5,
	medium: 17.5,
	heavy: 30,
};

export interface TagModeFormValues {
	positionTier: PositionTier;
}

interface TagModeFormProps {
	form: FormInstance<TagModeFormValues>;
	onFinish: (values: TagModeFormValues) => void;
}

// TagModeForm is the lightest holding entry form (TRD SS7.1).
// The user only picks a position tier; price is auto-filled from current market data.
export function TagModeForm({ form, onFinish }: TagModeFormProps) {
	const { t } = useTranslation("app");

	return (
		<Form form={form} layout="vertical" onFinish={onFinish} data-testid="tag-mode-form">
			<Typography.Paragraph type="secondary">
				{t("portfolio.tagModeForm.hint")}
			</Typography.Paragraph>

			<Form.Item
				label={t("portfolio.tagModeForm.positionTier")}
				name="positionTier"
				initialValue="medium"
				rules={[{ required: true }]}
			>
				<Radio.Group>
					<Radio.Button value="light">{t("portfolio.tagModeForm.tier.light")}</Radio.Button>
					<Radio.Button value="medium">{t("portfolio.tagModeForm.tier.medium")}</Radio.Button>
					<Radio.Button value="heavy">{t("portfolio.tagModeForm.tier.heavy")}</Radio.Button>
				</Radio.Group>
			</Form.Item>
		</Form>
	);
}
