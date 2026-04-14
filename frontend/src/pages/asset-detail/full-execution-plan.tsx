// full-execution-plan.tsx — complete execution plan for auth users with a holding.
// Shows current holding summary + execution plan content.
// Detects staleness (validDays expired or priceDrift > 5%) and shows refresh option.

import type { AssetDetailDto } from "@/features/asset-detail";
import { isJobTerminal, useAnalysisJob, useTriggerHoldingAnalysis } from "@/features/asset-detail";
import type { HoldingDto } from "@/features/portfolio";
import { Alert, Button, Card, Descriptions, Space } from "@/ui-kit/eat";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ExecutionPlanContent } from "./execution-plan-content";
import { computePriceDriftPercent, formatPrice } from "./utils";

interface Props {
	detail: AssetDetailDto;
	holding: HoldingDto;
}

const MAX_JOB_POLL_ATTEMPTS = 60;

export function FullExecutionPlan({ detail, holding }: Props) {
	const { t } = useTranslation("app");
	const [jobId, setJobId] = useState<string | null>(null);
	const [pollCount, setPollCount] = useState(0);
	const triggerAnalysis = useTriggerHoldingAnalysis();
	const jobQuery = useAnalysisJob(jobId);

	// Increment poll count each time job data arrives (track via data reference).
	useEffect(() => {
		if (jobQuery.data) {
			setPollCount((c) => c + 1);
		}
		// jobQuery.data reference changes on each successful refetch.
	}, [jobQuery.data]);

	// Stop polling if max attempts exceeded.
	const isTimedOut = pollCount >= MAX_JOB_POLL_ATTEMPTS;

	const drift = computePriceDriftPercent(detail.currentPrice, detail.priceAtAnalysis);
	const isStale = drift > 5;

	const handleRefresh = useCallback(() => {
		triggerAnalysis.mutate(holding.holdingId, {
			onSuccess: (data) => {
				setJobId(data.jobId);
				setPollCount(0);
			},
		});
	}, [triggerAnalysis, holding.holdingId]);

	const isRefreshing =
		triggerAnalysis.isPending || (!!jobId && jobQuery.data && !isJobTerminal(jobQuery.data.status));

	const jobFailed = jobId && jobQuery.data?.status === "failed";

	const pnl =
		holding.costPrice > 0 && detail.currentPrice !== undefined
			? ((detail.currentPrice - holding.costPrice) / holding.costPrice) * 100
			: null;

	return (
		<div style={{ padding: "16px 0" }}>
			{/* Holding summary */}
			<Card
				title={t("assetDetail.execution.fullPlan.holdingSummary")}
				size="small"
				style={{ marginBottom: 16 }}
			>
				<Descriptions size="small" column={3}>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.cost")}>
						{formatPrice(holding.costPrice, detail.currency)}
					</Descriptions.Item>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.position")}>
						{holding.positionRatio.toFixed(1)}%
					</Descriptions.Item>
					<Descriptions.Item label={t("assetDetail.execution.fullPlan.pnl")}>
						{pnl !== null ? (
							<span style={{ color: pnl >= 0 ? "#52c41a" : "#f5222d" }}>
								{pnl >= 0 ? "+" : ""}
								{pnl.toFixed(2)}%
							</span>
						) : (
							"—"
						)}
					</Descriptions.Item>
				</Descriptions>
			</Card>

			{/* Staleness warning + refresh */}
			{isStale && (
				<Alert
					type="warning"
					showIcon
					message={t("assetDetail.execution.fullPlan.stale")}
					style={{ marginBottom: 12 }}
					action={
						<Space>
							{isTimedOut ? (
								<span style={{ color: "#f5222d", fontSize: 12 }}>
									{t("assetDetail.execution.fullPlan.jobTimeout")}
								</span>
							) : jobFailed ? (
								<span style={{ color: "#f5222d", fontSize: 12 }}>
									{t("assetDetail.execution.fullPlan.jobError")}
								</span>
							) : (
								<Button
									size="small"
									onClick={handleRefresh}
									loading={isRefreshing}
									disabled={isRefreshing}
								>
									{isRefreshing
										? t("assetDetail.execution.fullPlan.refreshing")
										: t("assetDetail.execution.fullPlan.refresh")}
								</Button>
							)}
						</Space>
					}
				/>
			)}

			<ExecutionPlanContent plan={detail.executionPlan ?? null} />

			{/* Bottom disclaimer */}
			<div style={{ marginTop: 16, color: "#8c8c8c", fontSize: 11 }}>
				{t("assetDetail.disclaimer")}
			</div>
		</div>
	);
}
