import { BADGE_TEXT, ChangeBadge, type DecisionCardDTO } from "@/features/decision-card";
import { Card, Space, Typography } from "@/ui-kit/eat";
import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import "./ChangeAnchorList.css";

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
	const { t } = useTranslation("app");
	// Track the active highlight timer + node so a rapid second click cancels
	// the previous timer and we can clean up on unmount. Without this the
	// timer would race against re-clicks and could leave the highlight class
	// stuck on a card whose timer fired after a newer click already removed
	// and re-added it.
	const activeTimerRef = useRef<number | null>(null);
	const activeNodeRef = useRef<HTMLDivElement | null>(null);

	useEffect(() => {
		// Cleanup on unmount: cancel any pending timer and strip the class
		// from whichever card was last highlighted so a stale class never
		// persists across remounts.
		return () => {
			if (activeTimerRef.current !== null) {
				window.clearTimeout(activeTimerRef.current);
				activeTimerRef.current = null;
			}
			if (activeNodeRef.current) {
				activeNodeRef.current.classList.remove(HIGHLIGHT_CLASS);
				activeNodeRef.current = null;
			}
		};
	}, []);

	const changed = cards.filter((card) => card.badgeState !== "none");
	if (changed.length === 0) {
		return null;
	}

	const handleClick = (cardId: number) => {
		const node = cardRefs.get(cardId);
		if (!node) return;
		// Cancel any in-flight highlight before starting a new one so rapid
		// re-clicks don't strobe.
		if (activeTimerRef.current !== null) {
			window.clearTimeout(activeTimerRef.current);
			activeTimerRef.current = null;
		}
		if (activeNodeRef.current && activeNodeRef.current !== node) {
			activeNodeRef.current.classList.remove(HIGHLIGHT_CLASS);
		}
		node.scrollIntoView({ behavior: "smooth", block: "center" });
		node.classList.add(HIGHLIGHT_CLASS);
		activeNodeRef.current = node;
		activeTimerRef.current = window.setTimeout(() => {
			node.classList.remove(HIGHLIGHT_CLASS);
			if (activeNodeRef.current === node) {
				activeNodeRef.current = null;
			}
			activeTimerRef.current = null;
		}, CHANGE_HIGHLIGHT_DURATION_MS);
	};

	return (
		<Card data-testid="change-anchor-list">
			<Title level={5} style={{ marginTop: 0 }}>
				{t("dashboard.changeAnchor.title")}
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
