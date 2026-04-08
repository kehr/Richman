import {
	ASSET_CATEGORIES,
	ASSET_CATEGORY_META,
	type AssetCategory,
} from "@/features/asset-catalog";
import { usePatchUserSettings } from "@/features/user-settings";
import { Button, Card, Col, Row, Typography, message } from "@/ui-kit/eat";
import { motion, useReducedMotion } from "framer-motion";
import { useEffect } from "react";
import { OnboardingLayout } from "./components/OnboardingLayout";
import { useOnboardingState } from "./state";
import { useOnboardingNav } from "./use-onboarding-nav";

const { Text, Title } = Typography;

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

export default function CategoriesPage() {
	const nav = useOnboardingNav();
	const { state, update } = useOnboardingState();
	const patch = usePatchUserSettings();
	const reducedMotion = useReducedMotion();
	const items = reducedMotion ? reducedItemVariants : itemVariants;

	const toggleCategory = (key: AssetCategory) => {
		const next = state.categories.includes(key)
			? state.categories.filter((c) => c !== key)
			: [...state.categories, key];
		update({ categories: next });
	};

	useEffect(() => {
		return nav.registerCanGoNext(() => state.categories.length >= 1);
	}, [nav, state.categories]);

	const handleNext = async () => {
		if (!nav.canGoNext) return;
		try {
			await patch.mutateAsync({ categories: state.categories });
			await nav.next();
		} catch {
			message.error("保存失败，请稍后重试");
		}
	};

	return (
		<OnboardingLayout
			currentStep={2}
			title="你想关注哪些类型？"
			description="至少选 1 个。我们只分析你选的类型，后续随时可以在设置里调整。"
			footer={
				<Button
					type="primary"
					size="large"
					data-testid="onboarding-categories-next"
					disabled={!nav.canGoNext}
					loading={patch.isPending}
					onClick={handleNext}
				>
					下一步 →
				</Button>
			}
		>
			<motion.div
				variants={containerVariants}
				initial="hidden"
				animate="visible"
				style={{ display: "grid", gap: 16, gridTemplateColumns: "repeat(2, 1fr)" }}
			>
				{ASSET_CATEGORIES.map((key) => {
					const meta = ASSET_CATEGORY_META[key];
					const isSelected = state.categories.includes(key);
					return (
						<motion.div key={key} variants={items}>
							{/*
							  The card is wrapped in a native <button> so keyboard users
							  reach it via Tab and toggle with Enter/Space, while the
							  outer Card keeps its visual treatment. aria-pressed communicates
							  the multi-select toggle state to assistive tech.
							*/}
							<button
								type="button"
								data-testid={`category-card-${key}`}
								data-selected={isSelected ? "true" : "false"}
								aria-pressed={isSelected}
								aria-label={`${meta.label} ${meta.description}`}
								onClick={() => toggleCategory(key)}
								style={{
									width: "100%",
									padding: 0,
									background: "none",
									border: "none",
									textAlign: "left",
									cursor: "pointer",
								}}
							>
								<motion.div whileTap={{ scale: 0.98 }} transition={{ duration: 0.1 }}>
									<Card
										hoverable
										style={{
											borderColor: isSelected ? "#000" : undefined,
											borderWidth: isSelected ? 2 : 1,
											backgroundColor: isSelected ? "#f5f5f5" : undefined,
											transition: "all 0.15s",
										}}
									>
										<Title level={4} style={{ marginTop: 0, marginBottom: 4 }}>
											{meta.label}
										</Title>
										<Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
											{meta.description}
										</Text>
										<Text style={{ fontSize: 12, color: "#8c8c8c" }}>例如：{meta.examples}</Text>
									</Card>
								</motion.div>
							</button>
						</motion.div>
					);
				})}
			</motion.div>
		</OnboardingLayout>
	);
}
