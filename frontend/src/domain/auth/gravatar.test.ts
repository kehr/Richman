import { describe, expect, it } from "vitest";
import { gravatarUrl } from "./gravatar";

describe("gravatarUrl", () => {
	it("returns empty string for empty email", () => {
		expect(gravatarUrl("")).toBe("");
	});

	it("trims leading and trailing spaces from email", () => {
		const emailWithSpaces = "  user@example.com  ";
		const emailWithoutSpaces = "user@example.com";
		expect(gravatarUrl(emailWithSpaces)).toBe(gravatarUrl(emailWithoutSpaces));
	});

	it("converts email to lowercase before hashing", () => {
		const uppercaseEmail = "User@Example.com";
		const lowercaseEmail = "user@example.com";
		expect(gravatarUrl(uppercaseEmail)).toBe(gravatarUrl(lowercaseEmail));
	});

	it("includes d=identicon parameter in URL", () => {
		const url = gravatarUrl("user@example.com");
		expect(url).toContain("d=identicon");
	});

	it("includes r=g parameter in URL", () => {
		const url = gravatarUrl("user@example.com");
		expect(url).toContain("r=g");
	});

	it("includes correct s parameter for given size", () => {
		const url = gravatarUrl("user@example.com", 64);
		expect(url).toContain("s=64");
	});

	it("uses default size of 32 when no size argument provided", () => {
		const url = gravatarUrl("user@example.com");
		expect(url).toContain("s=32");
	});
});
