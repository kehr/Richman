import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OnboardingLayout } from "./OnboardingLayout";

// Mock the nav hook so the layout tests do not need to stand up a real
// OnboardingStateProvider + react-query status query + router navigation.
// Every test mutates the returned object via mockNav so we can assert
// specific call targets (prev, skip, jumpTo) without coupling to the hook's
// internal state machine.
const mockNav = {
	currentStep: 2 as 1 | 2 | 3 | 4,
	reachedStep: 2 as 1 | 2 | 3 | 4,
	canGoNext: true,
	prev: vi.fn(),
	next: vi.fn(async () => undefined),
	skip: vi.fn(async () => undefined),
	jumpTo: vi.fn(),
	registerCanGoNext: vi.fn(() => () => undefined),
};

vi.mock("../use-onboarding-nav", async () => {
	const actual =
		await vi.importActual<typeof import("../use-onboarding-nav")>("../use-onboarding-nav");
	return {
		...actual,
		useOnboardingNav: () => mockNav,
	};
});

function renderLayout(
	ui: Parameters<typeof renderWithProviders>[0] = (
		<OnboardingLayout currentStep={2} title="标题">
			<div data-testid="body">内容</div>
		</OnboardingLayout>
	),
) {
	return renderWithProviders(
		<MemoryRouter initialEntries={["/onboarding/categories"]}>{ui}</MemoryRouter>,
	);
}

describe("OnboardingLayout", () => {
	beforeEach(() => {
		mockNav.currentStep = 2;
		mockNav.reachedStep = 2;
		mockNav.canGoNext = true;
		mockNav.prev.mockReset();
		mockNav.next.mockReset();
		mockNav.skip.mockReset();
		mockNav.jumpTo.mockReset();
	});

	afterEach(() => {
		// Any Modal.confirm opened by a test is a body-level portal; remove the
		// mask manually so stray DOM from a previous test does not trip the
		// global keyboard guard in the next test. Guard with isConnected so we
		// skip nodes React has already unmounted during test cleanup.
		for (const node of document.querySelectorAll(".ant-modal-root")) {
			if (node.isConnected) {
				node.remove();
			}
		}
	});

	it("renders the title and the children content", () => {
		renderLayout();
		expect(screen.getByText("标题")).toBeInTheDocument();
		expect(screen.getByTestId("body")).toBeInTheDocument();
	});

	it("renders the description when provided", () => {
		renderLayout(
			<OnboardingLayout currentStep={2} title="标题" description={<span>副标题</span>}>
				<div>body</div>
			</OnboardingLayout>,
		);
		expect(screen.getByText("副标题")).toBeInTheDocument();
	});

	it("renders the footer slot when provided", () => {
		renderLayout(
			<OnboardingLayout currentStep={2} title="标题" footer={<button type="button">下一步</button>}>
				<div>body</div>
			</OnboardingLayout>,
		);
		expect(screen.getByRole("button", { name: "下一步" })).toBeInTheDocument();
	});

	it("hides the back button on step 1", () => {
		mockNav.currentStep = 1;
		mockNav.reachedStep = 1;
		renderLayout(
			<OnboardingLayout currentStep={1} title="欢迎">
				<div>body</div>
			</OnboardingLayout>,
		);
		expect(screen.queryByTestId("onboarding-back-button")).not.toBeInTheDocument();
	});

	it("shows the back button on step 2 and calls nav.prev on click", async () => {
		const user = userEvent.setup();
		renderLayout();
		const backButton = screen.getByTestId("onboarding-back-button");
		expect(backButton).toBeInTheDocument();
		await user.click(backButton);
		expect(mockNav.prev).toHaveBeenCalledTimes(1);
	});

	it("opens the skip confirm Modal when the skip link is clicked", async () => {
		const user = userEvent.setup();
		renderLayout();
		await user.click(screen.getByTestId("onboarding-skip-button"));
		// handleSkip defers the Modal open by one macrotask (setTimeout 0) so
		// waitFor polls until the Modal title appears in the document. The
		// confirm Modal renders the title in both an ant-modal-title container
		// and an ant-modal-confirm-title span, so we use findAllByText and
		// assert at least one match exists.
		const titles = await screen.findAllByText("Skip onboarding?", {}, { timeout: 2000 });
		expect(titles.length).toBeGreaterThan(0);
		// Assert the body copy renders through the confirm content slot.
		expect(
			screen.getByText(
				'You can restart the onboarding from Settings later, or click "Start onboarding" in the Dashboard banner to return here.',
			),
		).toBeInTheDocument();
		// Dismiss the confirm Modal by clicking the cancel button so antd's
		// portal cleanup runs in an orderly fashion before the test unmounts.
		// Leaving it open causes NotFoundError during React unmount when the
		// portal node is detached twice.
		await user.click(screen.getByRole("button", { name: "Continue onboarding" }));
		await waitFor(() => {
			expect(screen.queryByText("Skip onboarding?")).not.toBeInTheDocument();
		});
		expect(mockNav.skip).not.toHaveBeenCalled();
	});

	it("renders the step indicator with the current step label", () => {
		renderLayout();
		expect(screen.getByTestId("onboarding-step-indicator")).toBeInTheDocument();
		expect(screen.getByText("Step 2 / 5")).toBeInTheDocument();
	});

	it("invokes nav.prev when ArrowLeft is dispatched on window outside form fields", () => {
		renderLayout();
		// Dispatch a keydown with document.body as the target (the default for
		// events dispatched on window). The layout's guard only short-circuits
		// when the target is an INPUT / TEXTAREA / SELECT element, so a body
		// target must fall through to nav.prev.
		fireEvent.keyDown(window, { key: "ArrowLeft" });
		expect(mockNav.prev).toHaveBeenCalledTimes(1);
	});

	it("does NOT trap ArrowLeft when the event target is an input element", () => {
		renderLayout(
			<OnboardingLayout currentStep={2} title="标题">
				<input data-testid="text-input" type="text" />
			</OnboardingLayout>,
		);
		const input = screen.getByTestId("text-input");
		fireEvent.keyDown(input, { key: "ArrowLeft" });
		expect(mockNav.prev).not.toHaveBeenCalled();
	});
});
