import { Button, Card, Space, Typography } from "@/ui-kit/eat";

const { Paragraph, Title } = Typography;

interface EmptyHoldingsHeroProps {
	onAddHolding: () => void;
}

// EmptyHoldingsHero is the dashboard state shown when the authenticated user
// has finished onboarding but deleted all their holdings. It is intentionally
// minimal: a big centered card with a single primary CTA that routes back to
// the portfolio add flow. PRD §3.1 specifies this as a hero — large type,
// generous padding, no secondary actions.
export function EmptyHoldingsHero({ onAddHolding }: EmptyHoldingsHeroProps) {
	return (
		<Card
			data-testid="empty-holdings-hero"
			styles={{
				body: {
					display: "flex",
					justifyContent: "center",
					alignItems: "center",
					minHeight: 360,
					padding: 48,
				},
			}}
		>
			<Space direction="vertical" align="center" size={16}>
				<Title level={2} style={{ margin: 0, textAlign: "center" }}>
					先添加一个持仓
				</Title>
				<Paragraph type="secondary" style={{ marginBottom: 0, textAlign: "center" }}>
					Richman 基于你的真实持仓生成每日决策卡，添加第一笔持仓后分析会在几秒内返回。
				</Paragraph>
				<Button
					type="primary"
					size="large"
					onClick={onAddHolding}
					data-testid="empty-holdings-hero-cta"
				>
					添加持仓 →
				</Button>
			</Space>
		</Card>
	);
}
