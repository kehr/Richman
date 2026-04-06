"use client";

import { formatDate } from "@/domain/ui/format";
import { Col, Row, Skeleton, StatisticCard } from "@/ui-kit/eat";
import { useStats } from "./useStats";

export function StatsOverview() {
	const { data, isLoading } = useStats();

	if (isLoading) {
		return <Skeleton active paragraph={{ rows: 4 }} />;
	}

	return (
		<Row gutter={[16, 16]}>
			<Col xs={24} sm={8}>
				<StatisticCard
					statistic={{
						title: "Holdings",
						value: data?.holdingCount ?? 0,
					}}
				/>
			</Col>
			<Col xs={24} sm={8}>
				<StatisticCard
					statistic={{
						title: "Total Positions",
						value: data?.totalPositions ?? 0,
					}}
				/>
			</Col>
			<Col xs={24} sm={8}>
				<StatisticCard
					statistic={{
						title: "Latest Analysis",
						value: data?.latestAnalysisTime
							? formatDate(data.latestAnalysisTime, "datetime")
							: "N/A",
					}}
				/>
			</Col>
		</Row>
	);
}
