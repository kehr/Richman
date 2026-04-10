import { describe, expect, it } from "vitest";
import { formatAmount, formatAmountOrNull, formatPercent, formatPercentWithAmount } from "./format";

describe("formatPercent", () => {
	it("renders integer values without a decimal", () => {
		expect(formatPercent(5)).toBe("5%");
	});

	it("renders fractional values with one decimal", () => {
		expect(formatPercent(3.25)).toBe("3.3%");
	});
});

describe("formatAmount", () => {
	it("adds thousand separators and the yuan symbol", () => {
		expect(formatAmount(1234567)).toBe("¥1,234,567");
	});

	it("preserves negative amounts with a leading minus", () => {
		expect(formatAmount(-2500)).toBe("-¥2,500");
	});

	it("formats zero as ¥0", () => {
		expect(formatAmount(0)).toBe("¥0");
	});

	it("formats USD amounts with $ symbol", () => {
		expect(formatAmount(1234, "en", "USD")).toBe("$1,234");
		expect(formatAmount(1234, "zh", "USD")).toBe("$1,234");
	});

	it("formats HKD amounts with HK$ symbol", () => {
		expect(formatAmount(1234, "zh", "HKD")).toBe("HK$1,234");
		expect(formatAmount(1234, "en", "HKD")).toBe("HK$1,234");
	});

	it("formats negative USD amounts", () => {
		expect(formatAmount(-500, "en", "USD")).toBe("-$500");
	});

	it("formats zero USD as $0", () => {
		expect(formatAmount(0, "en", "USD")).toBe("$0");
	});
});

describe("formatPercentWithAmount", () => {
	it("returns only the percentage when no capital is configured", () => {
		expect(formatPercentWithAmount(5, 1000, false)).toBe("5%");
	});

	it("returns only the percentage when amount is null", () => {
		expect(formatPercentWithAmount(5, null, true)).toBe("5%");
	});

	it("returns only the percentage when amount is undefined", () => {
		expect(formatPercentWithAmount(5, undefined, true)).toBe("5%");
	});

	it("returns percentage + amount when both are available", () => {
		expect(formatPercentWithAmount(5, 5000, true)).toBe("5% · ¥5,000");
	});

	it("formats large amounts with thousand separators", () => {
		expect(formatPercentWithAmount(12, 1234567, true)).toBe("12% · ¥1,234,567");
	});

	it("renders zero amounts explicitly", () => {
		expect(formatPercentWithAmount(0, 0, true)).toBe("0% · ¥0");
	});

	it("supports negative amounts (e.g. unrealized losses)", () => {
		expect(formatPercentWithAmount(-3, -1500, true)).toBe("-3% · -¥1,500");
	});
});

describe("formatAmountOrNull", () => {
	it("returns null when capital is not configured", () => {
		expect(formatAmountOrNull(1000, false)).toBeNull();
	});

	it("returns null when amount is null", () => {
		expect(formatAmountOrNull(null, true)).toBeNull();
	});

	it("returns the formatted amount when both are available", () => {
		expect(formatAmountOrNull(1000, true)).toBe("¥1,000");
	});
});

describe("edge cases: NaN and negative zero", () => {
	it("formatPercent coerces NaN to 0%", () => {
		expect(formatPercent(Number.NaN)).toBe("0%");
	});

	it("formatAmount coerces NaN to ¥0", () => {
		expect(formatAmount(Number.NaN)).toBe("¥0");
	});

	it("formatAmount renders negative zero without a sign", () => {
		// -0 === 0 is true in JS, so the zero branch runs.
		expect(formatAmount(-0)).toBe("¥0");
	});

	it("formatPercentWithAmount handles NaN percent with an amount", () => {
		expect(formatPercentWithAmount(Number.NaN, 1000, true)).toBe("0% · ¥1,000");
	});
});
