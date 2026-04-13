import { useEventRadar } from "@/features/event-radar";
import { useMarketOverview, useMarketRegime } from "@/features/market-overview";
import { getToken } from "@/domain/auth/storage";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { AssetCardWall } from "./components/asset-card-wall";
import { EventRadarSection } from "./components/event-radar-section";
import { MarketRegimeBar } from "./components/market-regime-bar";
import { RegisterCTA } from "./components/register-cta";

// MarketOverviewPage is the public landing page (/market).
// It is accessible without a JWT token (PublicShell route).
// Components handle their own loading/error states to allow partial rendering.
export default function MarketOverviewPage() {
	const { t } = useTranslation("market");
	const queryClient = useQueryClient();

	const isAuthenticated = !!getToken();

	const {
		data: regimeData,
		isLoading: regimeLoading,
		isError: regimeError,
	} = useMarketRegime();

	const { data: overviewData, isLoading: overviewLoading } = useMarketOverview();

	const {
		data: eventData,
		isLoading: eventLoading,
		isError: eventError,
		refetch: refetchEvents,
	} = useEventRadar();

	// Set document title for SEO.
	useEffect(() => {
		const prev = document.title;
		document.title = `${t("overview.title")} — Richman`;
		return () => {
			document.title = prev;
		};
	}, [t]);

	const handleEventRetry = () => {
		queryClient.invalidateQueries({ queryKey: ["events", "radar"] });
		refetchEvents();
	};

	return (
		<div
			style={{
				maxWidth: 960,
				margin: "0 auto",
				padding: isAuthenticated ? "0 0 16px" : "0 0 72px",
			}}
		>
			{/* Market regime bar — hidden on richson error (G3.9) */}
			<MarketRegimeBar
				data={regimeData}
				isLoading={regimeLoading}
				isError={regimeError}
			/>

			{/* Grouped asset card wall */}
			<AssetCardWall data={overviewData} isLoading={overviewLoading} />

			{/* Macro event radar — shows retry on error (G3.9) */}
			<EventRadarSection
				data={eventData}
				isLoading={eventLoading}
				isError={eventError}
				onRetry={handleEventRetry}
			/>

			{/* Registration prompt for unauthenticated visitors */}
			{!isAuthenticated && <RegisterCTA />}
		</div>
	);
}
