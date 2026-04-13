import {
	type RiskPreference,
	usePatchRiskPreference,
	useUserSettings,
} from "@/features/user-settings";
import { App, Card, Flex, PageContainer, Skeleton, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

interface RiskCard {
	value: RiskPreference;
	emoji: string;
	colorBorder: string;
	colorBg: string;
	colorCheck: string;
}

const RISK_CARDS: RiskCard[] = [
	{
		value: "conservative",
		emoji: "S",
		colorBorder: "#52c41a",
		colorBg: "#f6ffed",
		colorCheck: "#52c41a",
	},
	{
		value: "neutral",
		emoji: "M",
		colorBorder: "#1677ff",
		colorBg: "#e6f4ff",
		colorCheck: "#1677ff",
	},
	{
		value: "aggressive",
		emoji: "H",
		colorBorder: "#fa8c16",
		colorBg: "#fff7e6",
		colorCheck: "#fa8c16",
	},
];

// RiskPreferenceSubPage presents three visual cards for the user to choose
// their risk preference (conservative / neutral / aggressive). On selection
// it calls PATCH /api/v1/user/settings and navigates back.
export default function RiskPreferenceSubPage() {
	const { t } = useTranslation("settings");
	const { message } = App.useApp();
	const navigate = useNavigate();
	const settingsQuery = useUserSettings();
	const patchMutation = usePatchRiskPreference();

	const current = settingsQuery.data?.riskPreference;

	const handleSelect = async (value: RiskPreference) => {
		if (value === current) return;
		try {
			await patchMutation.mutateAsync(value);
			message.success(t("riskPreference.updateSuccess"));
			navigate("/settings?tab=account");
		} catch {
			message.error(t("riskPreference.updateError"));
		}
	};

	return (
		<PageContainer
			title={t("riskPreference.pageTitle")}
			onBack={() => navigate("/settings?tab=account")}
			data-testid="risk-preference-sub-page"
		>
			<Typography.Paragraph type="secondary" style={{ maxWidth: 560 }}>
				{t("riskPreference.description")}
			</Typography.Paragraph>

			{settingsQuery.isLoading ? (
				<Skeleton active paragraph={{ rows: 4 }} />
			) : (
				<Flex gap={16} wrap="wrap" data-testid="risk-cards">
					{RISK_CARDS.map((card) => {
						const isSelected = current === card.value;
						return (
							<Card
								key={card.value}
								hoverable={!patchMutation.isPending}
								onClick={() => handleSelect(card.value)}
								style={{
									width: 200,
									cursor: patchMutation.isPending ? "not-allowed" : "pointer",
									borderColor: isSelected ? card.colorBorder : "#f0f0f0",
									borderWidth: isSelected ? 2 : 1,
									background: isSelected ? card.colorBg : "#fff",
									transition: "all 0.2s",
								}}
								data-testid={`risk-card-${card.value}`}
							>
								<Flex vertical align="center" gap={12}>
									<div
										style={{
											width: 56,
											height: 56,
											borderRadius: "50%",
											background: card.colorBg,
											border: `2px solid ${card.colorBorder}`,
											display: "flex",
											alignItems: "center",
											justifyContent: "center",
											fontSize: 22,
											fontWeight: 700,
											color: card.colorBorder,
										}}
									>
										{card.emoji}
									</div>
									<Typography.Text
										strong
										style={{ color: isSelected ? card.colorBorder : undefined }}
									>
										{t(`riskPreference.cards.${card.value}.title`)}
									</Typography.Text>
									<Typography.Text type="secondary" style={{ fontSize: 12, textAlign: "center" }}>
										{t(`riskPreference.cards.${card.value}.desc`)}
									</Typography.Text>
									{isSelected && (
										<Typography.Text
											style={{ fontSize: 12, color: card.colorCheck, fontWeight: 500 }}
										>
											{t("riskPreference.selected")}
										</Typography.Text>
									)}
								</Flex>
							</Card>
						);
					})}
				</Flex>
			)}
		</PageContainer>
	);
}
