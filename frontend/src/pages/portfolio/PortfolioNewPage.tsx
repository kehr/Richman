import { AssetPicker } from "@/features/asset-catalog";
import { HoldingForm } from "@/features/portfolio";
import { Card, PageContainer } from "@/ui-kit/eat";
import { useNavigate } from "react-router";

export default function PortfolioNewPage() {
	const navigate = useNavigate();

	return (
		<PageContainer title="New Holding">
			<Card>
				<HoldingForm
					onSuccess={() => navigate("/portfolio")}
					renderAssetPicker={(props) => (
						<AssetPicker open={props.open} onClose={props.onClose} onSelect={props.onSelect} />
					)}
				/>
			</Card>
		</PageContainer>
	);
}
