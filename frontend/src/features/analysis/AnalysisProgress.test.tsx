import { AnalysisProgress } from "./AnalysisProgress";
import { renderWithProviders } from "@/test/utils";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server } from "@/test/server";

const API_BASE = "http://localhost:8080/api/v1";

describe("AnalysisProgress", () => {
	it("starts analysis and shows running state", async () => {
		renderWithProviders(<AnalysisProgress />);
		const user = userEvent.setup();
		const button = screen.getByRole("button", { name: /start analysis/i });
		await user.click(button);
		expect(await screen.findByText(/analysis running/i)).toBeInTheDocument();
	});

	it("shows completion message when task completes", async () => {
		server.use(
			http.get(`${API_BASE}/analysis/tasks/:taskId`, async () =>
				HttpResponse.json({
					data: {
						taskId: "task-complete",
						status: "completed",
						progress: 1,
						error: "",
						startedAt: new Date().toISOString(),
						doneAt: new Date().toISOString(),
					},
				}),
			),
		);

		renderWithProviders(<AnalysisProgress />);
		const user = userEvent.setup();
		await user.click(screen.getByRole("button", { name: /start analysis/i }));
	await screen.findByText(/decision cards have been updated/i);
	});
});
