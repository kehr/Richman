import { type DecisionCardDTO, DecisionCardSummary } from "@/features/decision-card";
import { Alert, Button, Col, Empty, Row, Skeleton } from "@/ui-kit/eat";

interface DecisionCardWallProps {
	cards: DecisionCardDTO[];
	isLoading: boolean;
	error: unknown;
	onCardClick: (card: DecisionCardDTO) => void;
	onRetry: () => void;
	// cardRefs lets the parent attach scroll anchors to each card so the
	// ChangeAnchorList rows can scroll to and temporarily highlight the
	// matching card via data-testid lookup. Passing a Map makes the wiring
	// explicit without prop drilling refs into DecisionCardSummary.
	cardRefs?: Map<number, HTMLDivElement>;
}

// DecisionCardWall renders the middle region of the Dashboard: a responsive
// grid of decision cards. Columns are driven by antd Row+Col breakpoints:
//   xs (<576)  : 1 col  (24/24)
//   md (≥900)  : 2 cols (12/24)
//   xl (≥1200) : 3 cols (8/24)
// Loading and error states render inline so the parent page can stay a
// simple composition of top strip + wall + anchor list.
export function DecisionCardWall({
	cards,
	isLoading,
	error,
	onCardClick,
	onRetry,
	cardRefs,
}: DecisionCardWallProps) {
	if (isLoading) {
		return (
			<Row gutter={[16, 16]} data-testid="decision-card-wall-loading">
				{[0, 1, 2].map((i) => (
					<Col key={i} xs={24} md={12} xl={8}>
						<Skeleton active paragraph={{ rows: 6 }} />
					</Col>
				))}
			</Row>
		);
	}

	if (error) {
		return (
			<Alert
				type="error"
				showIcon
				message="加载决策卡失败"
				description="请检查网络连接后重试。"
				action={
					<Button size="small" onClick={onRetry}>
						重试
					</Button>
				}
				data-testid="decision-card-wall-error"
			/>
		);
	}

	if (cards.length === 0) {
		return (
			<Empty
				description="暂无决策卡，点击“重新分析”生成。"
				data-testid="decision-card-wall-empty"
			/>
		);
	}

	return (
		// Responsive grid per PRD §3.1: 1 column below ~900px, 2 columns
		// 900-1199px, 3 columns ≥1200px. antd has no native 900px breakpoint,
		// so we use `lg` (≥992) as the closest approximation to the spec's
		// 2-column threshold. The 92px gap (900 vs 992) is acceptable for MVP
		// and keeps us on antd's standard breakpoint token set.
		<Row gutter={[16, 16]} data-testid="decision-card-wall">
			{cards.map((card) => (
				<Col key={card.cardId} xs={24} lg={12} xl={8}>
					<div
						ref={(node) => {
							if (!cardRefs) return;
							if (node) {
								cardRefs.set(card.cardId, node);
							} else {
								cardRefs.delete(card.cardId);
							}
						}}
						data-card-anchor={card.cardId}
					>
						<DecisionCardSummary card={card} onClick={onCardClick} />
					</div>
				</Col>
			))}
		</Row>
	);
}
