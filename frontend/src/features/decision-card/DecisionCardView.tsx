"use client";

import { formatCurrency, formatDate, formatPercent } from "@/domain/ui/format";
import { Alert, Card, Col, Divider, Row, Space, Tag, Typography } from "@/ui-kit/eat";
import { ConfidenceBadge } from "./ConfidenceBadge";
import type { DecisionCardDto } from "./api";

const { Text, Title, Paragraph } = Typography;

function directionColor(direction: string): string {
	switch (direction) {
		case "bullish":
			return "green";
		case "bearish":
			return "red";
		default:
			return "default";
	}
}

function recommendationColor(rec: string): string {
	switch (rec) {
		case "add":
			return "green";
		case "hold":
			return "gold";
		case "reduce":
			return "red";
		default:
			return "default";
	}
}

interface DecisionCardViewProps {
	card: DecisionCardDto;
	compact?: boolean;
}

export function DecisionCardView({ card, compact = false }: DecisionCardViewProps) {
	return (
		<Card
			title={
				<Space>
					<Text strong>
						{card.assetName} ({card.assetCode})
					</Text>
					<Tag>{card.assetType}</Tag>
				</Space>
			}
			extra={
				<Space>
					<ConfidenceBadge value={card.confidence} />
					<Tag color={recommendationColor(card.recommendation)}>
						{card.recommendation.toUpperCase()}
					</Tag>
				</Space>
			}
		>
			<Space direction="vertical" style={{ width: "100%" }} size="middle">
				{/* Asset info */}
				<Row gutter={16}>
					<Col span={12}>
						<Text type="secondary">Cost: </Text>
						<Text>{formatCurrency(card.costPrice)}</Text>
					</Col>
					<Col span={12}>
						<Text type="secondary">Position: </Text>
						<Text>{formatPercent(card.positionRatio)}</Text>
					</Col>
				</Row>

				<Divider style={{ margin: "8px 0" }} />

				{/* 3D Summary */}
				<div>
					<Title level={5} style={{ marginBottom: 8 }}>
						3D Analysis
					</Title>
					<Space direction="vertical" style={{ width: "100%" }} size="small">
						<div>
							<Tag color={directionColor(card.trendDirection)}>Trend: {card.trendDirection}</Tag>
							<Text>{card.trendSummary}</Text>
						</div>
						<div>
							<Tag color={directionColor(card.positionDirection)}>
								Position: {card.positionDirection}
							</Tag>
							<Text>{card.positionSummary}</Text>
						</div>
						<div>
							<Tag color={directionColor(card.catalystDirection)}>
								Catalyst: {card.catalystDirection}
							</Tag>
							<Text>{card.catalystSummary}</Text>
						</div>
					</Space>
				</div>

				{/* Weights */}
				<Row gutter={8}>
					<Col span={8}>
						<Text type="secondary">Trend Weight: {formatPercent(card.weightTrend)}</Text>
					</Col>
					<Col span={8}>
						<Text type="secondary">Position Weight: {formatPercent(card.weightPosition)}</Text>
					</Col>
					<Col span={8}>
						<Text type="secondary">Catalyst Weight: {formatPercent(card.weightCatalyst)}</Text>
					</Col>
				</Row>

				<Divider style={{ margin: "8px 0" }} />

				{/* Action advice */}
				<div>
					<Title level={5} style={{ marginBottom: 8 }}>
						Action Advice
					</Title>
					<Paragraph>{card.actionAdvice}</Paragraph>
					{!compact && card.detailedAdvice && (
						<details>
							<summary style={{ cursor: "pointer", color: "#1677ff" }}>Detailed Advice</summary>
							<Paragraph style={{ marginTop: 8 }}>{card.detailedAdvice}</Paragraph>
						</details>
					)}
				</div>

				{/* Risk warnings */}
				{card.riskWarnings.length > 0 && (
					<Alert
						type="warning"
						message="Risk Warnings"
						description={
							<ul style={{ margin: 0, paddingLeft: 20 }}>
								{card.riskWarnings.map((w) => (
									<li key={w}>{w}</li>
								))}
							</ul>
						}
					/>
				)}

				{/* Today's highlights */}
				{card.todayHighlights && (
					<div>
						<Title level={5} style={{ marginBottom: 8 }}>
							Today&apos;s Highlights
						</Title>
						<Paragraph>{card.todayHighlights}</Paragraph>
					</div>
				)}

				{/* Analyzed time */}
				<Text type="secondary" style={{ fontSize: 12 }}>
					Analyzed: {formatDate(card.analyzedAt, "datetime")}
				</Text>
			</Space>
		</Card>
	);
}
