import { requestV2 } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { MyCodesResponse, MyInvitesResponse } from "./types";

// getMyCodes fetches all personal invite codes for the authenticated user,
// along with the unlock progress counter.
export function getMyCodes() {
	return requestV2<ApiResponse<MyCodesResponse>>("/invite/my-codes");
}

// getMyInvites fetches the list of users invited by the authenticated user.
export function getMyInvites() {
	return requestV2<ApiResponse<MyInvitesResponse>>("/invite/my-invites");
}
