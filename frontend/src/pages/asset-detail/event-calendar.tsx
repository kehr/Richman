import { EventRadarSection, useEventRadar } from "@/features/event-radar";
import { Card } from "@/ui-kit/eat";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

// EventCalendar shows the same upcoming-events feed as the market overview
// page, scoped under the asset detail risk tab. It reuses the shared
// useEventRadar query (cached cross-page via TanStack Query) and the
// EventRadarSection feature component for visual consistency.
export function EventCalendar() {
	const { t } = useTranslation("app");
	const queryClient = useQueryClient();
	const {
		data: eventData,
		isLoading: eventLoading,
		isError: eventError,
		refetch,
	} = useEventRadar();

	const handleRetry = () => {
		queryClient.invalidateQueries({ queryKey: ["events", "radar"] });
		refetch();
	};

	return (
		<Card title={t("assetDetail.risk.events.title")} size="small" style={{ marginBottom: 16 }}>
			<EventRadarSection
				data={eventData}
				isLoading={eventLoading}
				isError={eventError}
				onRetry={handleRetry}
			/>
		</Card>
	);
}
