import type { RecognizeResponse } from "@/features/portfolio";
import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ScreenshotImportModal } from "./ScreenshotImportModal";

const importMutate = vi.fn<(file: File) => Promise<RecognizeResponse>>();
const createMutate = vi.fn(async () => undefined);

vi.mock("@/features/portfolio", async () => {
	const actual =
		await vi.importActual<typeof import("@/features/portfolio")>("@/features/portfolio");
	return {
		...actual,
		useScreenshotImport: () => ({
			mutateAsync: (file: File) => importMutate(file),
			isPending: false,
		}),
		useCreateHolding: () => ({ mutateAsync: createMutate, isPending: false }),
	};
});

// jsdom does not implement createObjectURL; the modal calls it whenever the
// dragger accepts a file so we install a stub before the suite runs.
beforeEach(() => {
	importMutate.mockReset();
	createMutate.mockClear();
	if (!URL.createObjectURL) {
		URL.createObjectURL = vi.fn(() => "blob:mock");
	} else {
		(URL.createObjectURL as unknown as ReturnType<typeof vi.fn>) = vi.fn(() => "blob:mock");
	}
	if (!URL.revokeObjectURL) {
		URL.revokeObjectURL = vi.fn();
	}
});

function makeResponse(): RecognizeResponse {
	return {
		overallStatus: "ok",
		holdings: [
			{
				assetName: { value: "贵州茅台", confidence: 0.95 },
				assetCode: { value: "600519", confidence: 0.9 },
				costPrice: { value: "1700", confidence: 0.88 },
				positionPct: { value: "30", confidence: 0.92 },
				assetTypeGuess: "a_share_individual",
			},
			{
				assetName: { value: "宁德时代", confidence: 0.7 },
				assetCode: { value: "300750", confidence: 0.7 },
				costPrice: { value: "180", confidence: 0.5 },
				positionPct: { value: "15", confidence: 0.5 },
				assetTypeGuess: "a_share_individual",
			},
		],
	};
}

function uploadFile() {
	// antd Upload renders the file input outside the dragger node we tag
	// with data-testid, so query the entire portaled modal body for the
	// underlying input element instead.
	const input = document.querySelector(
		'.ant-upload input[type="file"]',
	) as HTMLInputElement | null;
	if (!input) throw new Error("upload input not found");
	const file = new File(["fake"], "screen.png", { type: "image/png" });
	fireEvent.change(input, { target: { files: [file] } });
}

describe("ScreenshotImportModal", () => {
	it("renders the upload dragger in the initial state", () => {
		renderWithProviders(
			<ScreenshotImportModal open onClose={() => {}} currentHoldingCount={0} holdingLimit={5} />,
		);
		expect(screen.getByTestId("screenshot-upload-dragger")).toBeInTheDocument();
	});

	it("shows the recognized table after a successful upload", async () => {
		importMutate.mockResolvedValue(makeResponse());
		renderWithProviders(
			<ScreenshotImportModal open onClose={() => {}} currentHoldingCount={0} holdingLimit={5} />,
		);
		uploadFile();
		await waitFor(() => {
			expect(screen.getByTestId("recognized-holding-table")).toBeInTheDocument();
		});
		expect(screen.getByTestId("screenshot-confirm-button")).toBeInTheDocument();
		// Both rows pre-checked since the user has zero holdings.
		const summary = screen.getByTestId("recognized-summary");
		expect(summary).toHaveTextContent("将新增 2 个持仓");
	});

	it("caps the number of selectable rows by the remaining slots", async () => {
		importMutate.mockResolvedValue(makeResponse());
		renderWithProviders(
			<ScreenshotImportModal open onClose={() => {}} currentHoldingCount={4} holdingLimit={5} />,
		);
		uploadFile();
		await waitFor(() => {
			expect(screen.getByTestId("recognized-holding-table")).toBeInTheDocument();
		});
		// Only one slot remains, so the second row should not be pre-checked.
		expect(screen.getByTestId("recognized-summary")).toHaveTextContent("将新增 1 个持仓");
		expect(screen.getByTestId("recognized-cap-warning")).toBeInTheDocument();
	});

	it("calls createHolding sequentially on confirm", async () => {
		importMutate.mockResolvedValue(makeResponse());
		const onClose = vi.fn();
		renderWithProviders(
			<ScreenshotImportModal open onClose={onClose} currentHoldingCount={0} holdingLimit={5} />,
		);
		uploadFile();
		await waitFor(() => {
			expect(screen.getByTestId("screenshot-confirm-button")).toBeInTheDocument();
		});
		fireEvent.click(screen.getByTestId("screenshot-confirm-button"));
		await waitFor(() => {
			expect(createMutate).toHaveBeenCalledTimes(2);
		});
		expect(onClose).toHaveBeenCalled();
	});
});
