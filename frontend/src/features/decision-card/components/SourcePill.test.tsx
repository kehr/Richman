import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SourcePill } from "./SourcePill";

describe("SourcePill", () => {
	it("renders an AI pill when source is llm", () => {
		render(<SourcePill source="llm" provider="user" />);
		expect(screen.getByTestId("source-pill-llm")).toHaveTextContent("AI");
	});

	it("renders a Mixed pill when source is mixed", () => {
		render(<SourcePill source="mixed" provider="system_default" />);
		expect(screen.getByTestId("source-pill-mixed")).toHaveTextContent("Mixed");
	});

	it("renders a Rules pill when source is template", () => {
		render(<SourcePill source="template" provider="none" />);
		expect(screen.getByTestId("source-pill-template")).toHaveTextContent("Rules");
	});

	it("renders nothing when source is unknown", () => {
		const { container } = render(<SourcePill source="unknown" provider="unknown" />);
		expect(container).toBeEmptyDOMElement();
	});
});
