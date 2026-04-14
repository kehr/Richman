import type { ExecutionPlanDto } from "@/features/asset-detail";
import {
	Alert,
	Badge,
	Card,
	Descriptions,
	Skeleton,
	Space,
	Tag,
	Timeline,
	Typography,
} from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

interface Props {
	plan: ExecutionPlanDto | null;
	isLoading?: boolean;
}

// ExecutionPlanContent renders the shared execution plan body.
// Used by both the demo plan CTAs and the full execution plan component.
export function ExecutionPlanContent({ plan, isLoading }: Props) {
	const { t } = useTranslation("app");

	if (isLoading) return <Skeleton active />;
	if (!plan) return null;

	const scenarios = plan.scenarios ?? [];

	return (
		<div>
			{plan.concentrationWarning && (
				<Alert
					type="warning"
					message={t("assetDetail.execution.fullPlan.concentrationWarning")}
					description={plan.concentrationWarning}
					showIcon
					style={{ marginBottom: 12 }}
				/>
			)}

			<Card size="small" style={{ marginBottom: 12 }}>
				<Descriptions size="small" column={2}>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.recommendation")}>
						<Tag color="blue">{plan.recommendation}</Tag>
					</Descriptions.Item>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.stopLoss")}>
						{plan.stopLoss !== null ? plan.stopLoss : t("assetDetail.execution.fullPlan.notSet")}
					</Descriptions.Item>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.takeProfit")}>
						{plan.takeProfit !== null
							? plan.takeProfit
							: t("assetDetail.execution.fullPlan.notSet")}
					</Descriptions.Item>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.validDays")}>
						{t("assetDetail.execution.fullPlan.validDays", { days: plan.validDays })}
					</Descriptions.Item>
				</Descriptions>
			</Card>

			<Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
				{plan.defaultAdvice}
			</Text>

			{scenarios.length > 0 && (
				<div style={{ marginTop: 12 }}>
					<Text strong style={{ display: "block", marginBottom: 8 }}>
						{t("assetDetail.execution.fullPlan.scenarios")}
					</Text>
					<Timeline
						items={scenarios.map((s) => ({
							color: s.priority === 1 ? "red" : "blue",
							children: (
								<div
									key={s.id}
									style={{
										border: s.priority === 1 ? "1px solid #f5222d" : "1px solid #d9d9d9",
										borderRadius: 4,
										padding: "8px 12px",
										marginBottom: 4,
									}}
								>
									<Space>
										{s.priority === 1 && (
											<Badge
												status="error"
												text={t("assetDetail.execution.fullPlan.priorityLabel")}
											/>
										)}
										<Text strong>{s.condition}</Text>
									</Space>
									<Text style={{ display: "block", marginTop: 4 }}>{s.action}</Text>
									<Text type="secondary" style={{ fontSize: 12 }}>
										{s.rationale}
									</Text>
								</div>
							),
						}))}
					/>
				</div>
			)}

			<Text type="secondary" style={{ fontSize: 11, display: "block", marginTop: 12 }}>
				{t("assetDetail.execution.fullPlan.disclaimer")}
			</Text>
		</div>
	);
}
