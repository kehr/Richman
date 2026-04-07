import { Alert, Space, Typography } from "@/ui-kit/eat";

const { Text } = Typography;

interface MainRisksProps {
	riskWarnings: string[];
}

// MainRisks renders the yellow "main risks" block per PRD section 5. When
// the backend supplies an empty list we still render a placeholder so the
// page does not collapse — the user wants to see risk reasoning even when
// the model produced none ("no significant risks identified").
export function MainRisks({ riskWarnings }: MainRisksProps) {
	const hasRisks = riskWarnings.length > 0;
	return (
		<Alert
			type="warning"
			showIcon
			data-testid="main-risks"
			message={<Text strong>主要风险</Text>}
			description={
				<Space direction="vertical" size={4} style={{ width: "100%" }}>
					{hasRisks ? (
						<ul style={{ margin: 0, paddingLeft: 20 }}>
							{riskWarnings.map((risk) => (
								<li key={risk}>{risk}</li>
							))}
						</ul>
					) : (
						<Text type="secondary">本次分析未识别到显著风险。</Text>
					)}
					<Text type="secondary">终止本计划的条件: 触发上述任一风险，或执行计划止损线。</Text>
				</Space>
			}
		/>
	);
}
