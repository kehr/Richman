import { type DecisionCardDTO, DecisionCardSummary } from "@/features/decision-card";
import type { HoldingProgress } from "@/features/decision-card";
import { Alert, Button, Empty, Skeleton } from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";

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
	holdingsProgress?: HoldingProgress[];
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
	holdingsProgress,
}: DecisionCardWallProps) {
	const { t } = useTranslation("app");

	const gridStyle: CSSProperties = {
		display: "grid",
		gridTemplateColumns: "repeat(auto-fill, minmax(380px, 1fr))",
		gap: 16,
	};

	if (isLoading) {
		return (
			<div style={gridStyle} data-testid="decision-card-wall-loading">
				{[0, 1, 2].map((i) => (
					<Skeleton key={i} active paragraph={{ rows: 6 }} />
				))}
			</div>
		);
	}

	if (error) {
		return (
			<Alert
				type="error"
				showIcon
				message={t("dashboard.cardWall.loadError")}
				description={t("dashboard.cardWall.loadErrorDesc")}
				action={
					<Button size="small" onClick={onRetry}>
						{t("dashboard.cardWall.retry")}
					</Button>
				}
				data-testid="decision-card-wall-error"
			/>
		);
	}

	if (cards.length === 0) {
		return (
			<Empty description={t("dashboard.cardWall.empty")} data-testid="decision-card-wall-empty" />
		);
	}

	return (
		// CSS grid auto-fill: columns are at least 380px wide and stretch to
		// fill available space. The browser determines column count from the
		// container width, so the layout adapts automatically as more cards
		// are added without any breakpoint configuration.
		<div style={gridStyle} data-testid="decision-card-wall">
			{cards.map((card) => {
				const holding = holdingsProgress?.find((h) => h.symbol === card.assetCode);
				return (
					<div
						key={card.cardId}
						ref={(node) => {
							if (!cardRefs) return;
							if (node) {
								cardRefs.set(card.cardId, node);
							} else {
								cardRefs.delete(card.cardId);
							}
						}}
						data-card-anchor={card.cardId}
						style={{ height: "100%" }}
					>
						<DecisionCardSummary
							card={card}
							onClick={onCardClick}
							analysisStatus={holding?.status}
							analysisProgress={holding?.progress}
						/>
					</div>
				);
			})}
		</div>
	);
}
