import { Tag } from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import type { BadgeState } from "../types";

// BADGE_TEXT is for non-React contexts only: push notifications, email
// templates, server-side logs. DO NOT use in JSX — always call
// t(`decisionCard.badge.${state}`) inside components so the locale is respected.
export const BADGE_TEXT: Record<Exclude<BadgeState, "none">, string> = {
	data_degraded: "Data Degraded",
	first_analysis: "First Analysis",
	action_upgrade: "Recommendation Upgrade",
	action_downgrade: "Recommendation Downgrade",
	signal_flip: "Signal Flip",
	plan_adjust: "Plan Adjusted",
	confidence_shift: "Confidence Shift",
};

// BADGE_COLORS maps each badge state to an antd Tag color token. Inline
// color choices keep the mapping co-located with the badge text so future
// copy changes only touch one file. Colors follow PRD §3.4:
//   data_degraded    gray
//   first_analysis   black
//   action_upgrade   green
//   action_downgrade red
//   signal_flip      blue
//   plan_adjust      amber/gold
//   confidence_shift purple
const BADGE_COLORS: Record<Exclude<BadgeState, "none">, string> = {
	data_degraded: "default",
	first_analysis: "#000000",
	action_upgrade: "green",
	action_downgrade: "red",
	signal_flip: "blue",
	plan_adjust: "gold",
	confidence_shift: "purple",
};

interface ChangeBadgeProps {
	badgeState: BadgeState;
	style?: CSSProperties;
}

// ChangeBadge renders the "what changed since the previous card" pill used
// in the top-right corner of DecisionCardSummary. Returns null for the
// `none` state so callers can unconditionally render it without a wrapper.
export function ChangeBadge({ badgeState, style }: ChangeBadgeProps) {
	const { t } = useTranslation("app");

	if (badgeState === "none") {
		return null;
	}
	const color = BADGE_COLORS[badgeState];
	const text = t(`decisionCard.badge.${badgeState}`);
	return (
		<Tag color={color} style={style} data-testid={`change-badge-${badgeState}`}>
			{text}
		</Tag>
	);
}
