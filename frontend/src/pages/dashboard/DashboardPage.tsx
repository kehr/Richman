import { StatsOverview } from "@/features/dashboard";
import { PageContainer } from "@/ui-kit/eat";

export default function DashboardPage() {
	return (
		<PageContainer title="Dashboard">
			<StatsOverview />
		</PageContainer>
	);
}
