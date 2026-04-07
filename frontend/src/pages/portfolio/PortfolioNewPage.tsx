import { HoldingForm } from "@/features/portfolio";
import { Card, PageContainer } from "@/ui-kit/eat";
import { useNavigate } from "react-router";

// PortfolioNewPage is a temporary thin wrapper around HoldingForm. Step 16
// will replace this page with a proper AddHoldingDrawer that embeds a real
// asset picker + quick / detail / screenshot tabs.
export default function PortfolioNewPage() {
	const navigate = useNavigate();

	return (
		<PageContainer title="New Holding">
			<Card>
				<HoldingForm onSuccess={() => navigate("/portfolio")} />
			</Card>
		</PageContainer>
	);
}
