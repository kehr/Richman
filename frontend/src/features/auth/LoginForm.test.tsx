import { renderWithProviders } from "@/test/utils";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { LoginForm } from "./LoginForm";

// Capture the options bag that LoginForm passes into useLogin so we can
// verify the full Page -> Form -> hook wire end to end. The stub also
// exposes `lastMutateArgs` so the test can simulate a successful submit
// without invoking the real network stack.
const useLoginSpy = vi.fn<(options?: { redirectTo?: string }) => unknown>();
const mutateSpy = vi.fn();

vi.mock("./useAuth", () => ({
	useLogin: (options?: { redirectTo?: string }) => {
		useLoginSpy(options);
		return { mutate: mutateSpy, isPending: false, error: null };
	},
}));

function renderForm(redirectTo?: string) {
	useLoginSpy.mockClear();
	mutateSpy.mockClear();
	return renderWithProviders(
		<MemoryRouter>
			<LoginForm redirectTo={redirectTo} />
		</MemoryRouter>,
	);
}

describe("LoginForm", () => {
	it("forwards the redirectTo prop to useLogin", () => {
		renderForm("/decision-cards/42");
		expect(useLoginSpy).toHaveBeenCalledWith({ redirectTo: "/decision-cards/42" });
	});

	it("passes undefined redirectTo when no prop is provided", () => {
		renderForm();
		expect(useLoginSpy).toHaveBeenCalledWith({ redirectTo: undefined });
	});

	it("calls the login mutation with the entered credentials on submit", async () => {
		renderForm("/portfolio");
		fireEvent.change(screen.getByPlaceholderText("Email"), {
			target: { value: "tester@example.com" },
		});
		fireEvent.change(screen.getByPlaceholderText("Password"), { target: { value: "secret1" } });
		fireEvent.click(screen.getByRole("button", { name: "Sign In" }));
		await waitFor(() => {
			expect(mutateSpy).toHaveBeenCalledWith({
				email: "tester@example.com",
				password: "secret1",
			});
		});
		// The most recent useLogin call saw the caller-provided redirectTo, so
		// the hook's onSuccess will route to /portfolio when the mutation
		// resolves. We don't invoke onSuccess here because the hook module is
		// stubbed; verifying the options-bag receipt is sufficient to prove
		// the wire.
		expect(useLoginSpy).toHaveBeenLastCalledWith({ redirectTo: "/portfolio" });
	});
});
