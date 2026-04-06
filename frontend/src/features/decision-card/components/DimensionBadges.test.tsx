import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DimensionBadges } from "./DimensionBadges";

describe("DimensionBadges", () => {
	it("renders three dimension labels", () => {
		render(
			<DimensionBadges
				trend={{ label: "Trend", current: "bullish" }}
				position={{ label: "Position", current: "neutral" }}
				catalyst={{ label: "Catalyst", current: "bearish" }}
			/>,
		);
		expect(screen.getByTestId("dim-trend-current")).toHaveTextContent("bullish");
		expect(screen.getByTestId("dim-position-current")).toHaveTextContent("neutral");
		expect(screen.getByTestId("dim-catalyst-current")).toHaveTextContent("bearish");
	});

	it("renders strikethrough previous value and arrow when flipped", () => {
		render(
			<DimensionBadges
				trend={{ label: "Trend", current: "bullish", previous: "bearish" }}
				position={{ label: "Position", current: "neutral" }}
				catalyst={{ label: "Catalyst", current: "neutral" }}
			/>,
		);
		const prev = screen.getByTestId("dim-trend-prev");
		expect(prev).toHaveTextContent("bearish");
		expect(prev).toHaveStyle({ textDecoration: "line-through" });
		expect(screen.getByTestId("dim-trend-current")).toHaveTextContent("bullish");
	});

	it("does not render a previous span when current equals previous", () => {
		render(
			<DimensionBadges
				trend={{ label: "Trend", current: "bullish", previous: "bullish" }}
				position={{ label: "Position", current: "neutral" }}
				catalyst={{ label: "Catalyst", current: "neutral" }}
			/>,
		);
		expect(screen.queryByTestId("dim-trend-prev")).toBeNull();
	});
});
