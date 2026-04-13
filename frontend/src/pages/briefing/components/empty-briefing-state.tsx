import { Button, Empty, Flex, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

interface EmptyBriefingStateProps {
	onAddHolding: () => void;
}

// EmptyBriefingState is shown when the user has no holdings yet (TRD SS6.1).
// It prompts the user to add their first holding to unlock the briefing.
export function EmptyBriefingState({ onAddHolding }: EmptyBriefingStateProps) {
	const { t } = useTranslation("app");

	return (
		<Flex vertical align="center" justify="center" style={{ minHeight: 300, padding: "40px 0" }}>
			<Empty
				description={
					<Flex vertical align="center" gap={8}>
						<Typography.Text strong style={{ fontSize: 16 }}>
							{t("briefing.empty.title")}
						</Typography.Text>
						<Typography.Text type="secondary" style={{ textAlign: "center", maxWidth: 360 }}>
							{t("briefing.empty.description")}
						</Typography.Text>
					</Flex>
				}
			>
				<Button type="primary" onClick={onAddHolding}>
					{t("briefing.empty.addButton")}
				</Button>
			</Empty>
		</Flex>
	);
}
