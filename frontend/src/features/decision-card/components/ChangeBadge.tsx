import { Tag } from "@/ui-kit/eat";
import type { CSSProperties } from "react";
import type { BadgeState } from "../types";

// BADGE_TEXT is the canonical, reusable copy for each of the 8 badge states
// defined in PRD §3.4. Exported so the Help page and tests can reference the
// same source of truth instead of duplicating strings.
export const BADGE_TEXT: Record<Exclude<BadgeState, "none">, string> = {
	data_degraded: "数据降级",
	first_analysis: "首次分析",
	action_upgrade: "建议升级",
	action_downgrade: "建议降级",
	signal_flip: "信号翻转",
	plan_adjust: "计划调整",
	confidence_shift: "信心度波动",
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
	if (badgeState === "none") {
		return null;
	}
	const color = BADGE_COLORS[badgeState];
	const text = BADGE_TEXT[badgeState];
	return (
		<Tag color={color} style={style} data-testid={`change-badge-${badgeState}`}>
			{text}
		</Tag>
	);
}
