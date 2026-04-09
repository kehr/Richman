import { getSectionIds } from "@/i18n/help";
import { renderWithProviders, testI18n } from "@/test/utils";
import { act, fireEvent, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";
import HelpPage from "./HelpPage";

// jsdom does not implement IntersectionObserver — stub it so the hook runs
// without throwing. The stub records the callback so tests can trigger
// synthetic visibility events if they need to.
class IntersectionObserverMock {
	static lastInstance: IntersectionObserverMock | null = null;
	callback: IntersectionObserverCallback;
	constructor(cb: IntersectionObserverCallback) {
		this.callback = cb;
		IntersectionObserverMock.lastInstance = this;
	}
	observe() {}
	unobserve() {}
	disconnect() {}
	takeRecords() {
		return [];
	}
}

// biome-ignore lint/suspicious/noExplicitAny: test stub
(globalThis as any).IntersectionObserver = IntersectionObserverMock;

// jsdom does not implement Element.scrollIntoView — the HelpPage calls it
// from both the hash effect and the sidebar click handler. Stub it as a
// no-op so the test run does not throw.
if (!Element.prototype.scrollIntoView) {
	Element.prototype.scrollIntoView = () => {};
}

function renderHelp(initialEntry = "/help") {
	return renderWithProviders(
		<MemoryRouter initialEntries={[initialEntry]}>
			<HelpPage />
		</MemoryRouter>,
	);
}

describe("HelpPage", () => {
	beforeEach(async () => {
		await testI18n.changeLanguage("en");
	});

	it("renders all 9 section ids from PRD §7.2", () => {
		renderHelp();
		const expectedIds = [
			"badge",
			"dimensions",
			"actions",
			"plan",
			"confidence",
			"data",
			"degradation",
			"push",
			"risk",
		];
		expect(getSectionIds("en")).toEqual(expectedIds);
		for (const id of expectedIds) {
			expect(screen.getByTestId(`help-section-${id}`)).toBeInTheDocument();
		}
	});

	it("highlights the sidebar entry after clicking it", () => {
		renderHelp();
		const link = screen.getByTestId("help-sidebar-link-actions");
		fireEvent.click(link);
		expect(link).toHaveAttribute("aria-current", "location");
	});

	it("highlights the sidebar entry when IntersectionObserver reports it visible", () => {
		renderHelp();
		const target = document.getElementById("dimensions");
		expect(target).not.toBeNull();
		const observer = IntersectionObserverMock.lastInstance;
		expect(observer).not.toBeNull();
		act(() => {
			observer?.callback(
				[
					{
						isIntersecting: true,
						target: target as Element,
						boundingClientRect: { top: 100 } as DOMRectReadOnly,
						intersectionRatio: 1,
						intersectionRect: {} as DOMRectReadOnly,
						rootBounds: null,
						time: 0,
					},
				],
				observer as unknown as IntersectionObserver,
			);
		});
		expect(screen.getByTestId("help-sidebar-link-dimensions")).toHaveAttribute(
			"aria-current",
			"location",
		);
	});

	it("surfaces English content when the locale is en", () => {
		// testI18n is initialized to "en" and reset to "en" in beforeEach, so
		// HelpPage will pick up English content via i18n.language.
		renderHelp();
		// Section titles show up in both the sidebar and the main heading, so
		// use getAllByText and assert at least one match rather than a unique
		// render. This keeps the test robust to future layout tweaks.
		expect(screen.getAllByText("Confidence").length).toBeGreaterThan(0);
		expect(screen.getAllByText("Change Badges").length).toBeGreaterThan(0);
		// Structure invariant: all 9 ids still render regardless of locale.
		for (const id of getSectionIds("en")) {
			expect(screen.getByTestId(`help-section-${id}`)).toBeInTheDocument();
		}
	});
});
