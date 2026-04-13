import { Flex, Segmented, Typography } from "@/ui-kit/eat";
import { useTranslation } from "react-i18next";

export type BriefingViewMode = "compact" | "detailed";

interface BriefingHeaderProps {
	viewMode: BriefingViewMode;
	onViewModeChange: (mode: BriefingViewMode) => void;
	generatedAt?: string;
}

// BriefingHeader renders the page title, last-updated timestamp, and
// compact/detailed mode toggle (TRD SS6.3).
export function BriefingHeader({ viewMode, onViewModeChange, generatedAt }: BriefingHeaderProps) {
	const { t } = useTranslation("app");

	const formattedDate = generatedAt
		? new Intl.DateTimeFormat(undefined, {
				month: "short",
				day: "numeric",
				hour: "2-digit",
				minute: "2-digit",
			}).format(new Date(generatedAt))
		: null;

	return (
		<Flex align="center" justify="space-between" style={{ marginBottom: 16 }}>
			<Flex vertical gap={2}>
				<Typography.Title level={3} style={{ margin: 0 }}>
					{t("briefing.title")}
				</Typography.Title>
				{formattedDate && (
					<Typography.Text type="secondary" style={{ fontSize: 12 }}>
						{t("briefing.updatedAt", { time: formattedDate })}
					</Typography.Text>
				)}
			</Flex>
			<Segmented
				value={viewMode}
				onChange={(v) => onViewModeChange(v as BriefingViewMode)}
				options={[
					{ label: t("briefing.viewMode.compact"), value: "compact" },
					{ label: t("briefing.viewMode.detailed"), value: "detailed" },
				]}
			/>
		</Flex>
	);
}
