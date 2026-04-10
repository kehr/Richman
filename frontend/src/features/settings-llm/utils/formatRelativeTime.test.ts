import { describe, expect, it } from "vitest";
import { formatRelativeTime } from "./formatRelativeTime";

const now = Date.now();

function dateSecondsAgo(seconds: number): Date {
	return new Date(now - seconds * 1000);
}

describe("formatRelativeTime", () => {
	it("returns em dash for null", () => {
		expect(formatRelativeTime(null, "zh")).toBe("\u2014");
	});

	it("returns em dash for undefined", () => {
		expect(formatRelativeTime(undefined, "en")).toBe("\u2014");
	});

	it("uses second unit for a date 30 seconds ago", () => {
		const result = formatRelativeTime(dateSecondsAgo(30), "en");
		expect(result).toMatch(/second/);
	});

	it("uses minute unit for a date 2 minutes ago (zh)", () => {
		const result = formatRelativeTime(dateSecondsAgo(120), "zh");
		expect(result).toMatch(/分钟/);
	});

	it("uses minute unit for a date 2 minutes ago (en)", () => {
		const result = formatRelativeTime(dateSecondsAgo(120), "en");
		expect(result).toMatch(/minute/);
	});

	it("uses hour unit for a date 3 hours ago", () => {
		const result = formatRelativeTime(dateSecondsAgo(3 * 3600), "en");
		expect(result).toMatch(/hour/);
	});

	it("uses day unit for a date 5 days ago", () => {
		const result = formatRelativeTime(dateSecondsAgo(5 * 86400), "en");
		expect(result).toMatch(/day/);
	});

	it("accepts an ISO string date input", () => {
		const iso = new Date(now - 120 * 1000).toISOString();
		const result = formatRelativeTime(iso, "en");
		expect(result).toMatch(/minute/);
	});
});
