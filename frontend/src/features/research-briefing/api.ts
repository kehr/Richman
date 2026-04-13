import { requestV2 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { BriefingDto } from "./types";

// fetchBriefing loads the aggregated briefing payload for the authenticated user.
// Calls the v2 endpoint which is backed by richson analysis data.
export function fetchBriefing() {
	return requestV2<ApiResponse<BriefingDto>>("/briefing");
}
