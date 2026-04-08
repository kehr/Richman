import { Button, Card, Col, Row, Typography } from "@/ui-kit/eat";
import { motion, useReducedMotion } from "framer-motion";
import { OnboardingLayout } from "./components/OnboardingLayout";
import { useOnboardingNav } from "./use-onboarding-nav";

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

const containerVariants = {
	hidden: { opacity: 0 },
	visible: {
		opacity: 1,
		transition: { staggerChildren: 0.08 },
	},
};

const itemVariants = {
	hidden: { opacity: 0, y: 20 },
	visible: {
		opacity: 1,
		y: 0,
		transition: { duration: 0.4, ease: "easeOut" },
	},
};

const reducedItemVariants = {
	hidden: { opacity: 0 },
	visible: { opacity: 1, transition: { duration: 0.2 } },
};

export default function WelcomePage() {
	const nav = useOnboardingNav();
	const reducedMotion = useReducedMotion();
	const items = reducedMotion ? reducedItemVariants : itemVariants;

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
					onClick={() => {
						void nav.next();
					}}
				>
					开始设置 →
				</Button>
			}
		>
			<motion.div
				variants={containerVariants}
				initial="hidden"
				animate="visible"
				style={{ display: "grid", gap: 16, gridTemplateColumns: "repeat(3, 1fr)" }}
			>
				{DIMENSIONS.map((dim) => (
					<motion.div key={dim.key} variants={items}>
						<Card
							data-testid={`dimension-card-${dim.key}`}
							style={{ height: "100%", textAlign: "center" }}
						>
							<Title level={4} style={{ marginTop: 0 }}>
								{dim.title}
							</Title>
							<Text type="secondary">{dim.description}</Text>
						</Card>
					</motion.div>
				))}
			</motion.div>
		</OnboardingLayout>
	);
}
