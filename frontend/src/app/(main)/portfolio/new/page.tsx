"use client";

import { HoldingForm } from "@/features/portfolio";
import { Card, PageContainer } from "@/ui-kit/eat";
import { useRouter } from "next/navigation";

export default function NewHoldingPage() {
	const router = useRouter();

	return (
		<PageContainer title="New Holding">
			<Card>
				<HoldingForm onSuccess={() => router.push("/portfolio")} />
			</Card>
		</PageContainer>
	);
}
