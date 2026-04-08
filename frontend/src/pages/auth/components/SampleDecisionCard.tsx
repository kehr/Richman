import { Card, Space, Tag, Typography } from "@/ui-kit/eat";

const { Title, Text, Paragraph } = Typography;

// SampleDecisionCard is a pure presentational mock of the dashboard decision
// card, used inside the login / register left pane so visitors can see the
// shape of the product before they sign in. All data is hardcoded and the
// component imports nothing from the decision-card feature to avoid creating
// a pages -> features -> domain coupling chain.
//
// Visual contract: it should "look like" a real DecisionCardSummary so the
// pitch on the auth page matches what the user sees once they're in. The
// three dimension tags here use the same color tokens as
// features/decision-card/components/DimensionBadges (bullish=green,
// neutral=default, bearish=red) so the language is consistent.
export function SampleDecisionCard() {
	return (
		<Card
			data-testid="sample-decision-card"
			style={{
				borderRadius: 12,
				boxShadow: "0 2px 16px rgba(0, 0, 0, 0.06)",
			}}
			styles={{ body: { padding: 20 } }}
		>
			<div
				style={{
					display: "flex",
					alignItems: "baseline",
					justifyContent: "space-between",
					marginBottom: 12,
				}}
			>
				<div>
					<Title level={5} style={{ margin: 0 }}>
						贵州茅台
					</Title>
					<Text type="secondary" style={{ fontSize: 12 }}>
						600519
					</Text>
				</div>
				<Tag color="#000000">首次分析</Tag>
			</div>
			<Space size="small" wrap style={{ marginBottom: 12 }}>
				<Tag color="green">趋势: bullish</Tag>
				<Tag color="default">位置: neutral</Tag>
				<Tag color="green">催化剂: bullish</Tag>
			</Space>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "space-between",
					marginBottom: 8,
				}}
			>
				<Text strong style={{ fontSize: 16 }}>
					建议：小幅加仓
				</Text>
				<Text type="secondary">信心度 82</Text>
			</div>
			<Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 13 }}>
				趋势向上、位置中性，近期催化剂偏正面，可在当前价位附近小幅加仓，控制单笔仓位风险。
			</Paragraph>
		</Card>
	);
}
