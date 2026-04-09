import { useDashboardSummary } from "@/features/dashboard-summary";
import { LLMSection } from "@/features/settings-llm";

// AITab is the settings tab dedicated to AI interpretation provider
// configuration. It is a thin composition layer that reads the dashboard
// summary (to know whether the system-default provider is available) and
// delegates all mutation handling to LLMSection. No business logic lives
// in this file.
export function AITab() {
	const dashboardQuery = useDashboardSummary();
	const systemDefaultAvailable = dashboardQuery.data?.llmStatus.systemDefaultAvailable ?? false;

	return (
		<div data-testid="ai-tab">
			<LLMSection systemDefaultAvailable={systemDefaultAvailable} />
		</div>
	);
}
