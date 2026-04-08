import { Button, Card, Col, Row, Typography } from "@/ui-kit/eat";
import { useNavigate } from "react-router";
import { OnboardingLayout } from "./components/OnboardingLayout";

const { Text, Title } = Typography;

// Intro card data for the three-dimension preview. Keeping the copy inline
// (rather than routing it through i18n) mirrors the other onboarding pages
// and keeps Step 13 self-contained; post-MVP i18n work (Step 19) can migrate
// these strings.
const DIMENSIONS = [
	{
		key: "trend",
		title: "趋势",
		description: "持仓所在行业与大盘的方向信号",
	},
	{
		key: "position",
		title: "位置",
		description: "当前点位相对区间的偏高或偏低",
	},
	{
		key: "catalyst",
		title: "催化剂",
		description: "即将发生的事件与新闻对预期的冲击",
	},
];

export default function WelcomePage() {
	const navigate = useNavigate();

	return (
		<OnboardingLayout
			currentStep={1}
			title="欢迎，让我们开始"
			description={
				<>
					Richman 基于你的真实持仓给出明确建议。
					<br />
					三维分析覆盖趋势、位置与催化剂，只看重要的变化。
					<br />
					每一次回来都有一张新决策卡，不是一堆你消化不完的信息流。
				</>
			}
			footer={
				<Button
					type="primary"
					size="large"
					data-testid="onboarding-welcome-next"
					onClick={() => navigate("/onboarding/categories")}
				>
					开始设置 →
				</Button>
			}
		>
			<Row gutter={[16, 16]}>
				{DIMENSIONS.map((dim) => (
					<Col xs={24} sm={8} key={dim.key}>
						<Card
							data-testid={`dimension-card-${dim.key}`}
							style={{ height: "100%", textAlign: "center" }}
						>
							<Title level={4} style={{ marginTop: 0 }}>
								{dim.title}
							</Title>
							<Text type="secondary">{dim.description}</Text>
						</Card>
					</Col>
				))}
			</Row>
		</OnboardingLayout>
	);
}
