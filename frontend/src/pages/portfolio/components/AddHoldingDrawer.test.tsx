import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AddHoldingDrawer } from "./AddHoldingDrawer";

const createMutate = vi.fn(async () => undefined);

vi.mock("@/features/portfolio", async () => {
	const actual =
		await vi.importActual<typeof import("@/features/portfolio")>("@/features/portfolio");
	return {
		...actual,
		useCreateHolding: () => ({ mutateAsync: createMutate, isPending: false }),
	};
});

vi.mock("@/features/asset-catalog", async () => {
	const actual = await vi.importActual<typeof import("@/features/asset-catalog")>(
		"@/features/asset-catalog",
	);
	return {
		...actual,
		useAssets: () => ({
			data: [
				{
					code: "510300",
					name: "沪深 300",
					nameEn: "CSI 300",
					assetType: "a_share_broad",
					exchange: "SSE",
				},
			],
			isLoading: false,
		}),
	};
});

describe("AddHoldingDrawer", () => {
	beforeEach(() => {
		createMutate.mockClear();
	});

	it("renders step 1 with asset type tabs and the asset select", () => {
		renderWithProviders(<AddHoldingDrawer open={true} onClose={() => {}} />);
		expect(screen.getByTestId("add-holding-drawer")).toBeInTheDocument();
		expect(screen.getByTestId("asset-type-tabs")).toBeInTheDocument();
		expect(screen.getByTestId("asset-select")).toBeInTheDocument();
		// Save button is disabled until an asset is selected.
		expect(screen.getByTestId("add-holding-save")).toBeDisabled();
	});

	it("disables the detail and screenshot tabs in step 2", async () => {
		renderWithProviders(<AddHoldingDrawer open={true} onClose={() => {}} />);
		// Skip step 1 by selecting an asset directly via the (mocked) select.
		// Use the fact that the inner ant select renders a search input we can
		// drive — but easier: trigger the underlying onChange via the rendered
		// Select option after focusing it.
		const select = screen
			.getByTestId("asset-select")
			.querySelector(".ant-select-selector") as HTMLElement;
		fireEvent.mouseDown(select);
		await waitFor(() => {
			expect(screen.getByText("510300 沪深 300")).toBeInTheDocument();
		});
		fireEvent.click(screen.getByText("510300 沪深 300"));

		await waitFor(() => {
			expect(screen.getByTestId("selected-asset-chip")).toBeInTheDocument();
		});
		expect(screen.getByTestId("tab-detail-disabled")).toBeInTheDocument();
		expect(screen.getByTestId("tab-screenshot-disabled")).toBeInTheDocument();
		expect(screen.getByTestId("quick-holding-form")).toBeInTheDocument();
	});
});
