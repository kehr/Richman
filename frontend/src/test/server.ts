import type { AnalysisTaskStatusDto } from "@/features/analysis/api";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

const API_BASE = "http://localhost:8080/api/v1";

const defaultTask: AnalysisTaskStatusDto = {
	taskId: "task-mock",
	status: "running",
	progress: 0.35,
	startedAt: new Date().toISOString(),
};

export const handlers = [
	http.post(`${API_BASE}/analysis/trigger`, async () =>
		HttpResponse.json({
			data: { taskId: "task-mock", message: "analysis started" },
		}),
	),
	http.get(`${API_BASE}/analysis/tasks/:taskId`, async () =>
		HttpResponse.json({ data: defaultTask }),
	),
];

export const server = setupServer(...handlers);
