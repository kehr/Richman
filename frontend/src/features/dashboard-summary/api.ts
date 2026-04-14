import { requestV1 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { DashboardSummaryDTO } from "./types";

// getDashboardSummary loads the aggregated dashboard payload. The endpoint
// is lightweight compared to /decision-cards (single SELECT per user) so
// the UI can poll it on every dashboard mount without coalescing.
export function getDashboardSummary() {
	return requestV1<ApiResponse<DashboardSummaryDTO>>("/dashboard/summary");
}
