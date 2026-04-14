// Asset Detail Page — /market/:code
// Public page with sticky header and three lazily-loaded tabs.

import { useAssetDetail } from "@/features/asset-detail";
import { Alert, Button, ReloadOutlined, ShareAltOutlined, Skeleton, Tabs } from "@/ui-kit/eat";
import { App as AntApp } from "@/ui-kit/eat";
import { Helmet } from "@dr.pogodin/react-helmet";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useParams } from "react-router";
import { AnalysisTab } from "./analysis-tab";
import { ExecutionTab } from "./execution-tab";
import { RiskTab } from "./risk-tab";
import { StickyHeader } from "./sticky-header";

type TabKey = "analysis" | "risk" | "execution";

// visitedTabs tracks which tabs have been activated so that lazy-loaded
// tabs retain their data when the user switches back.
function useTabLazyLoading() {
	const [visited, setVisited] = useState<Set<TabKey>>(new Set(["analysis"]));
	const handleTabChange = (key: string) => {
		setVisited((prev) => new Set([...prev, key as TabKey]));
	};
	return { visited, handleTabChange };
}

export default function AssetDetailPage() {
	const { code } = useParams<{ code: string }>();
	const { t } = useTranslation("app");
	const { message } = AntApp.useApp();
	const { visited, handleTabChange } = useTabLazyLoading();

	const { data: detail, isLoading, isError, refetch } = useAssetDetail(code ?? "");

	// Share handler — copies current URL to clipboard.
	const handleShare = () => {
		navigator.clipboard.writeText(window.location.href).then(() => {
			message.success(t("assetDetail.shareCopied"));
		});
	};

	if (isLoading) {
		return (
			<div style={{ padding: 24 }}>
				<Skeleton active paragraph={{ rows: 4 }} />
			</div>
		);
	}

	if (isError || !detail) {
		return (
			<div style={{ padding: 24 }}>
				<Alert
					type="error"
					message={isError ? t("assetDetail.loadError") : t("assetDetail.notFound")}
					action={
						<Button icon={<ReloadOutlined />} onClick={() => refetch()} size="small">
							{t("assetDetail.retry")}
						</Button>
					}
				/>
			</div>
		);
	}

	const signalLabel = detail.signalLevel
		? t(`assetDetail.scoreSummary.signal.${detail.signalLevel}`, detail.signalLevel)
		: null;
	const scoreText = detail.overallScore !== undefined ? `${detail.overallScore}/100` : null;
	// Note: keep these as `undefined` rather than "" — Helmet treats empty
	// strings as invalid text descendants (<title> aside) and throws.
	// `|| undefined` also folds empty strings from the backend down to undefined.
	const interpretationDescription = detail.marketInterpretation?.slice(0, 160) || undefined;
	const interpretationOg = detail.marketInterpretation?.slice(0, 100) || undefined;
	const titleSegments = [detail.name, scoreText, signalLabel, "Richman"].filter((s): s is string =>
		Boolean(s),
	);
	const pageTitle = titleSegments.join(" | ");
	const ogTitleSegments = [detail.name, signalLabel].filter((s): s is string => Boolean(s));
	const ogTitle = ogTitleSegments.length > 0 ? ogTitleSegments.join(" ") : undefined;
	const tabItems = [
		{
			key: "analysis",
			label: t("assetDetail.tab.analysis"),
			children: visited.has("analysis") ? <AnalysisTab detail={detail} /> : null,
		},
		{
			key: "risk",
			label: t("assetDetail.tab.risk"),
			children: visited.has("risk") ? <RiskTab detail={detail} /> : null,
		},
		{
			key: "execution",
			label: t("assetDetail.tab.execution"),
			children: visited.has("execution") ? <ExecutionTab detail={detail} /> : null,
		},
	];

	return (
		<div style={{ maxWidth: 900, margin: "0 auto" }}>
			<Helmet>
				<title>{pageTitle}</title>
				{interpretationDescription && (
					<meta name="description" content={interpretationDescription} />
				)}
				{ogTitle && <meta property="og:title" content={ogTitle} />}
				{interpretationOg && <meta property="og:description" content={interpretationOg} />}
			</Helmet>

			<StickyHeader detail={detail} />

			<div style={{ padding: "0 16px" }}>
				<div style={{ display: "flex", justifyContent: "flex-end", padding: "8px 0" }}>
					<Button icon={<ShareAltOutlined />} size="small" type="text" onClick={handleShare}>
						{t("assetDetail.share")}
					</Button>
				</div>

				<Tabs
					defaultActiveKey="analysis"
					onChange={handleTabChange}
					items={tabItems}
					destroyOnHidden={false}
				/>

				<div
					style={{
						color: "#8c8c8c",
						fontSize: 11,
						borderTop: "1px solid #f0f0f0",
						padding: "12px 0",
						marginTop: 24,
					}}
				>
					{t("assetDetail.disclaimer")}
				</div>
			</div>
		</div>
	);
}
