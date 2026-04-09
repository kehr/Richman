import { Alert } from "@/ui-kit/eat";

export function RiskDisclaimer() {
	return (
		<Alert
			type="warning"
			banner
			message="For reference only, not investment advice. Investment carries risks, please make decisions carefully."
			style={{ marginTop: 16 }}
		/>
	);
}
