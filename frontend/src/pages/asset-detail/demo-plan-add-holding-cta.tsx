import { useDemoPlan } from "@/features/asset-detail";
import { Alert, Button, Card, Divider, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { ExecutionPlanContent } from "./execution-plan-content";

const { Text } = Typography;

interface Props {
	code: string;
}

// DemoPlanAddHoldingCta is shown to authenticated users who have no holding for this asset.
export function DemoPlanAddHoldingCta({ code }: Props) {
	const { t } = useTranslation("app");
	const navigate = useNavigate();
	const { data, isLoading } = useDemoPlan(code, true);

	return (
		<div style={{ padding: "16px 0" }}>
			<Alert
				type="info"
				message={t("assetDetail.execution.demoPlan.disclaimer")}
				style={{ marginBottom: 16 }}
				showIcon
			/>
			{data && <ExecutionPlanContent plan={data.executionPlan} isLoading={isLoading} />}
			<Divider />
			<Card style={{ textAlign: "center", marginTop: 16 }}>
				<Text style={{ display: "block", marginBottom: 12 }}>
					{t("assetDetail.execution.demoPlan.addHoldingCTA")}
				</Text>
				<Button type="primary" size="large" onClick={() => navigate("/portfolio")}>
					{t("assetDetail.execution.noHolding.addButton")}
				</Button>
			</Card>
		</div>
	);
}
