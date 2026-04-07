import {
	ASSET_CATEGORIES,
	ASSET_CATEGORY_META,
	type AssetCategory,
} from "@/features/asset-catalog";
import { usePatchUserSettings } from "@/features/user-settings";
import { Button, Card, Col, Row, Typography, message } from "@/ui-kit/eat";
import { useState } from "react";
import { useNavigate } from "react-router";
import { OnboardingLayout } from "./components/OnboardingLayout";

const { Text, Title } = Typography;

export default function CategoriesPage() {
	const navigate = useNavigate();
	const patch = usePatchUserSettings();
	const [selected, setSelected] = useState<AssetCategory[]>([]);

	const toggle = (key: AssetCategory) => {
		setSelected((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
	};

	const canContinue = selected.length > 0;

	const handleNext = async () => {
		if (!canContinue) return;
		try {
			await patch.mutateAsync({ categories: selected });
			navigate("/onboarding/first-holding");
		} catch {
			message.error("保存失败，请重试");
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
					disabled={!canContinue}
					loading={patch.isPending}
					onClick={handleNext}
				>
					下一步 →
				</Button>
			}
		>
			<Row gutter={[16, 16]}>
				{ASSET_CATEGORIES.map((key) => {
					const meta = ASSET_CATEGORY_META[key];
					const isSelected = selected.includes(key);
					return (
						<Col xs={24} sm={12} key={key}>
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
								onClick={() => toggle(key)}
								style={{
									width: "100%",
									padding: 0,
									background: "none",
									border: "none",
									textAlign: "left",
									cursor: "pointer",
								}}
							>
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
							</button>
						</Col>
					);
				})}
			</Row>
		</OnboardingLayout>
	);
}
