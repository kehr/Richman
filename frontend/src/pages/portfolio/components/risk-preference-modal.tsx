import { Button, Card, Col, Flex, Modal, Row, Typography } from "@/ui-kit/eat";
import { useState } from "react";
import { useTranslation } from "react-i18next";

// RiskPreference mirrors the backend enum for user risk tolerance.
export type RiskPreference = "conservative" | "moderate" | "aggressive";

interface RiskPreferenceModalProps {
	open: boolean;
	onConfirm: (preference: RiskPreference) => void;
	onCancel: () => void;
	loading?: boolean;
}

// RiskPreferenceModal prompts the user to choose a risk profile on first holding
// entry (TRD SS7.2). Three cards are shown: conservative, moderate (default
// highlighted), and aggressive.
export function RiskPreferenceModal({
	open,
	onConfirm,
	onCancel,
	loading,
}: RiskPreferenceModalProps) {
	const { t } = useTranslation("app");
	const { t: tCommon } = useTranslation("common");
	const [selected, setSelected] = useState<RiskPreference>("moderate");

	const options: Array<{ value: RiskPreference; emoji: string }> = [
		{ value: "conservative", emoji: "🛡" },
		{ value: "moderate", emoji: "⚖" },
		{ value: "aggressive", emoji: "🚀" },
	];

	return (
		<Modal
			open={open}
			title={t("portfolio.riskPreference.title")}
			onCancel={onCancel}
			footer={
				<Flex justify="flex-end" gap={8}>
					<Button onClick={onCancel}>{tCommon("action.cancel")}</Button>
					<Button type="primary" loading={loading} onClick={() => onConfirm(selected)}>
						{tCommon("action.confirm")}
					</Button>
				</Flex>
			}
			width={520}
			data-testid="risk-preference-modal"
		>
			<Typography.Paragraph type="secondary">
				{t("portfolio.riskPreference.description")}
			</Typography.Paragraph>
			<Row gutter={12} style={{ marginTop: 16 }}>
				{options.map(({ value, emoji }) => (
					<Col key={value} span={8}>
						<Card
							hoverable
							onClick={() => setSelected(value)}
							style={{
								cursor: "pointer",
								border: selected === value ? "2px solid var(--ant-color-primary)" : undefined,
								transition: "border 0.15s",
							}}
							styles={{ body: { padding: "16px 12px", textAlign: "center" } }}
						>
							<div style={{ fontSize: 28, marginBottom: 8 }}>{emoji}</div>
							<Typography.Text strong>
								{t(`portfolio.riskPreference.option.${value}.label`)}
							</Typography.Text>
							<br />
							<Typography.Text type="secondary" style={{ fontSize: 12 }}>
								{t(`portfolio.riskPreference.option.${value}.description`)}
							</Typography.Text>
						</Card>
					</Col>
				))}
			</Row>
		</Modal>
	);
}
