import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { BadgeState } from "../types";
import { BADGE_TEXT, ChangeBadge } from "./ChangeBadge";

describe("ChangeBadge", () => {
	it("renders null for badge_state=none", () => {
		const { container } = render(<ChangeBadge badgeState="none" />);
		expect(container.firstChild).toBeNull();
	});

	const states: Exclude<BadgeState, "none">[] = [
		"data_degraded",
		"first_analysis",
		"action_upgrade",
		"action_downgrade",
		"signal_flip",
		"plan_adjust",
		"confidence_shift",
	];

	for (const state of states) {
		it(`renders the ${state} state with its canonical text`, () => {
			render(<ChangeBadge badgeState={state} />);
			expect(screen.getByTestId(`change-badge-${state}`)).toHaveTextContent(BADGE_TEXT[state]);
		});
	}
});
