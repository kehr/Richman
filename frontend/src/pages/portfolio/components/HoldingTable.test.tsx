import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { HoldingTable } from "./HoldingTable";

vi.mock("@/features/user-settings", () => ({
	useUserSettings: () => ({ data: { totalCapitalCny: 100_000 }, isLoading: false }),
}));

vi.mock("@/domain/money/useMoney", () => ({
	useMoney: () => ({
		hasCapital: true,
		format: (pct: number, amount?: number | null) =>
			amount != null ? `${pct}% · ¥${amount}` : `${pct}%`,
		formatAmountOnly: (amount?: number | null) => (amount != null ? `¥${amount}` : null),
	}),
}));

const sample = [
	{
		holdingId: 1,
		assetCode: "510300",
		assetName: "沪深 300",
		assetType: "a_share_broad",
		costPrice: 4.12,
		positionRatio: 20,
		quantity: 0,
	},
	{
		holdingId: 2,
		assetCode: "518880",
		assetName: "华安黄金",
		assetType: "gold_etf",
		costPrice: 5.55,
		positionRatio: 10,
		quantity: 0,
	},
];

describe("HoldingTable", () => {
	it("renders all rows with the seven required columns", () => {
		renderWithProviders(<HoldingTable holdings={sample} />);
		expect(screen.getByTestId("holding-row-1")).toBeInTheDocument();
		expect(screen.getByTestId("holding-row-2")).toBeInTheDocument();
		// Headers from PRD §4.1.
		expect(screen.getByRole("columnheader", { name: "标的" })).toBeInTheDocument();
		expect(screen.getByRole("columnheader", { name: "类型" })).toBeInTheDocument();
		expect(screen.getByRole("columnheader", { name: "成本" })).toBeInTheDocument();
		expect(screen.getByRole("columnheader", { name: "现价" })).toBeInTheDocument();
		expect(screen.getByRole("columnheader", { name: "仓位" })).toBeInTheDocument();
		expect(screen.getByRole("columnheader", { name: "浮盈亏" })).toBeInTheDocument();
		expect(screen.getByRole("columnheader", { name: "操作" })).toBeInTheDocument();
	});

	it("invokes onRowClick when a body row is clicked", () => {
		const onRowClick = vi.fn();
		renderWithProviders(<HoldingTable holdings={sample} onRowClick={onRowClick} />);
		fireEvent.click(screen.getByTestId("holding-row-1"));
		expect(onRowClick).toHaveBeenCalledWith(sample[0]);
	});

	it("does not bubble row click when an action button is pressed", () => {
		const onRowClick = vi.fn();
		const onEdit = vi.fn();
		renderWithProviders(<HoldingTable holdings={sample} onRowClick={onRowClick} onEdit={onEdit} />);
		// The Space wrapping action buttons stops propagation; click an Edit
		// button and assert the row click handler is not invoked.
		const actionsCell = screen.getByTestId("holding-actions-1");
		const editButton = actionsCell.querySelector("button");
		expect(editButton).not.toBeNull();
		fireEvent.click(editButton as HTMLButtonElement);
		expect(onEdit).toHaveBeenCalledWith(sample[0]);
		expect(onRowClick).not.toHaveBeenCalled();
	});
});
