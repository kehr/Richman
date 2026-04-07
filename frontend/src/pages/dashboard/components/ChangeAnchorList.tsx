import { BADGE_TEXT, ChangeBadge, type DecisionCardDTO } from "@/features/decision-card";
import { Card, Space, Typography } from "@/ui-kit/eat";

const { Text, Title } = Typography;

// CHANGE_HIGHLIGHT_DURATION_MS is the length of the temporary highlight added
// to a decision card after a user clicks on its anchor row. 1.5s is long
// enough to draw attention without being disruptive (per PRD §3.1).
const CHANGE_HIGHLIGHT_DURATION_MS = 1500;
const HIGHLIGHT_CLASS = "dashboard-change-anchor-highlight";

interface ChangeAnchorListProps {
	cards: DecisionCardDTO[];
	// cardRefs is the Map populated by DecisionCardWall so the anchor list can
	// locate the DOM node for each card without a separate ref collection.
	cardRefs: Map<number, HTMLDivElement>;
}

// buildChangeSummary renders the right-hand side of an anchor row. The copy
// mirrors the badge-state taxonomy from PRD §3.4 and reuses BADGE_TEXT as the
// single source of truth for the badge label.
function buildChangeSummary(card: DecisionCardDTO): string {
	if (card.badgeState === "none") return "";
	return BADGE_TEXT[card.badgeState];
}

// ChangeAnchorList is the bottom "变化锚点" region of the Dashboard per PRD
// §3.1. It filters the card list down to the cards whose badgeState is not
// "none" and, when a row is clicked, scrolls to and temporarily highlights
// the matching card in the wall. When there are no changed cards the whole
// block returns null so the Dashboard layout collapses cleanly.
export function ChangeAnchorList({ cards, cardRefs }: ChangeAnchorListProps) {
	const changed = cards.filter((card) => card.badgeState !== "none");
	if (changed.length === 0) {
		return null;
	}

	const handleClick = (cardId: number) => {
		const node = cardRefs.get(cardId);
		if (!node) return;
		node.scrollIntoView({ behavior: "smooth", block: "center" });
		node.classList.add(HIGHLIGHT_CLASS);
		window.setTimeout(() => {
			node.classList.remove(HIGHLIGHT_CLASS);
		}, CHANGE_HIGHLIGHT_DURATION_MS);
	};

	return (
		<Card data-testid="change-anchor-list">
			<Title level={5} style={{ marginTop: 0 }}>
				今日变化
			</Title>
			<Space direction="vertical" size={8} style={{ width: "100%" }}>
				{changed.map((card) => (
					<button
						type="button"
						key={card.cardId}
						onClick={() => handleClick(card.cardId)}
						data-testid={`change-anchor-row-${card.cardId}`}
						style={{
							display: "flex",
							alignItems: "center",
							gap: 12,
							width: "100%",
							padding: "8px 12px",
							border: "1px solid transparent",
							borderRadius: 6,
							background: "transparent",
							cursor: "pointer",
							textAlign: "left",
						}}
					>
						<ChangeBadge badgeState={card.badgeState} />
						<Text strong>{card.assetName}</Text>
						<Text type="secondary">→ {buildChangeSummary(card)}</Text>
					</button>
				))}
			</Space>
		</Card>
	);
}
