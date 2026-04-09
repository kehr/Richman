import { Button, Card, Col, Row, Typography } from "@/ui-kit/eat";
import { motion, useReducedMotion } from "framer-motion";
import { useTranslation } from "react-i18next";
import { OnboardingLayout } from "./components/OnboardingLayout";
import { useOnboardingNav } from "./use-onboarding-nav";

const { Text, Title } = Typography;

const DIMENSION_KEYS = ["trend", "position", "catalyst"] as const;

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
	const { t } = useTranslation("auth");
	const nav = useOnboardingNav();
	const reducedMotion = useReducedMotion();
	const items = reducedMotion ? reducedItemVariants : itemVariants;

	return (
		<OnboardingLayout
			currentStep={1}
			title={t("onboarding.welcome.title")}
			description={
				<>
					{t("onboarding.welcome.description")}
					<br />
					{t("onboarding.welcome.descriptionLine2")}
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
					{t("onboarding.welcome.startButton")}
				</Button>
			}
		>
			<motion.div
				variants={containerVariants}
				initial="hidden"
				animate="visible"
				style={{ display: "grid", gap: 16, gridTemplateColumns: "repeat(3, 1fr)" }}
			>
				{DIMENSION_KEYS.map((key) => (
					<motion.div key={key} variants={items}>
						<Card
							data-testid={`dimension-card-${key}`}
							style={{ height: "100%", textAlign: "center" }}
						>
							<Title level={4} style={{ marginTop: 0 }}>
								{t(`onboarding.welcome.dimension.${key}.title`)}
							</Title>
							<Text type="secondary">{t(`onboarding.welcome.dimension.${key}.description`)}</Text>
						</Card>
					</motion.div>
				))}
			</motion.div>
		</OnboardingLayout>
	);
}
