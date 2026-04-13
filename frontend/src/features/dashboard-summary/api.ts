import { requestV1 as request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { DashboardSummaryDTO } from "./types";

// getDashboardSummary loads the aggregated dashboard payload. The endpoint
// is lightweight compared to /decision-cards (single SELECT per user) so
// the UI can poll it on every dashboard mount without coalescing.
export function getDashboardSummary() {
	return request<ApiResponse<DashboardSummaryDTO>>("/dashboard/summary");
}
