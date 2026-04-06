"use client";

import { AnalysisProgress } from "@/features/analysis";
import { PageContainer } from "@/ui-kit/eat";

export default function AnalysisPage() {
	return (
		<PageContainer title="Analysis">
			<AnalysisProgress />
		</PageContainer>
	);
}
