import { renderWithProviders } from "@/test/utils";
import { screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import LoginPage, { resolveReturnTo } from "./LoginPage";

// Capture the redirectTo argument that the page forwards into useLogin via
// the LoginForm wrapper. The mock replaces the entire @/features/auth barrel
// so we don't run the real mutation against MSW; we just need to know which
// path the page asked the form to redirect to after success and to verify
// that "Sign In" actually triggers the success callback.
const useLoginCall = vi.fn();

vi.mock("@/features/auth", () => ({
	LoginForm: ({ redirectTo }: { redirectTo?: string }) => {
		useLoginCall(redirectTo);
		return (
			<div data-testid="login-form-stub">
				<span data-testid="login-form-redirect-to">{redirectTo ?? ""}</span>
			</div>
		);
	},
	RegisterForm: () => <div data-testid="register-form-stub" />,
	useLogin: () => ({ mutate: vi.fn(), isPending: false, error: null }),
	useRegister: () => ({ mutate: vi.fn(), isPending: false, error: null }),
	useLogout: () => vi.fn(),
}));

function renderAt(initialPath: string) {
	useLoginCall.mockClear();
	return renderWithProviders(
		<MemoryRouter initialEntries={[initialPath]}>
			<LoginPage />
		</MemoryRouter>,
	);
}

describe("resolveReturnTo", () => {
	it("returns /dashboard for null, empty string, or whitespace-only fallback cases", () => {
		expect(resolveReturnTo(null)).toBe("/dashboard");
		expect(resolveReturnTo("")).toBe("/dashboard");
	});

	it("accepts a normal relative path", () => {
		expect(resolveReturnTo("/decision-cards/5")).toBe("/decision-cards/5");
		expect(resolveReturnTo("/portfolio")).toBe("/portfolio");
	});

	it("rejects values that do not start with a single slash", () => {
		expect(resolveReturnTo("dashboard")).toBe("/dashboard");
		expect(resolveReturnTo("./local")).toBe("/dashboard");
		expect(resolveReturnTo("https://evil.com/steal")).toBe("/dashboard");
	});

	it("rejects protocol-relative and backslash-prefixed paths", () => {
		expect(resolveReturnTo("//evil.com")).toBe("/dashboard");
		expect(resolveReturnTo("/\\evil.com")).toBe("/dashboard");
	});

	it("rejects values that smuggle a scheme later in the string", () => {
		expect(resolveReturnTo("/redirect?next=https://evil.com")).toBe("/dashboard");
		expect(resolveReturnTo("/foo/https://evil")).toBe("/dashboard");
	});

	it("rejects values that contain ASCII control characters", () => {
		expect(resolveReturnTo("/foo\nbar")).toBe("/dashboard");
		expect(resolveReturnTo("/foo\u0000bar")).toBe("/dashboard");
	});
});

describe("LoginPage", () => {
	it("renders the split layout with both panes and the form on the right", () => {
		renderAt("/login");
		expect(screen.getByTestId("auth-split-layout")).toBeInTheDocument();
		expect(screen.getByTestId("auth-split-layout-left")).toHaveTextContent("Richman");
		expect(screen.getByTestId("auth-split-layout-right")).toContainElement(
			screen.getByTestId("login-form-stub"),
		);
	});

	it("renders the sample decision card inside the left pane", () => {
		renderAt("/login");
		expect(screen.getByTestId("sample-decision-card")).toBeInTheDocument();
		expect(screen.getByTestId("sample-decision-card")).toHaveTextContent("Kweichow Moutai");
	});

	it("forwards a valid relative ?returnTo= to the login form", async () => {
		renderAt("/login?returnTo=%2Fdecision-cards%2F5");
		await waitFor(() => {
			expect(screen.getByTestId("login-form-redirect-to")).toHaveTextContent("/decision-cards/5");
		});
		expect(useLoginCall).toHaveBeenLastCalledWith("/decision-cards/5");
	});

	it("ignores an absolute returnTo URL and falls back to /dashboard", async () => {
		renderAt("/login?returnTo=https%3A%2F%2Fevil.com%2Fsteal");
		await waitFor(() => {
			expect(screen.getByTestId("login-form-redirect-to")).toHaveTextContent("/dashboard");
		});
	});

	it("ignores a returnTo that does not start with a slash", async () => {
		renderAt("/login?returnTo=dashboard");
		await waitFor(() => {
			expect(screen.getByTestId("login-form-redirect-to")).toHaveTextContent("/dashboard");
		});
	});

	it("ignores a protocol-relative // returnTo", async () => {
		renderAt("/login?returnTo=%2F%2Fevil.com");
		await waitFor(() => {
			expect(screen.getByTestId("login-form-redirect-to")).toHaveTextContent("/dashboard");
		});
	});

	it("falls back to /dashboard when ?returnTo= is missing", () => {
		renderAt("/login");
		expect(screen.getByTestId("login-form-redirect-to")).toHaveTextContent("/dashboard");
		expect(useLoginCall).toHaveBeenLastCalledWith("/dashboard");
	});
});
