// InviteCode represents a single personal invite code entry from the backend.
export interface InviteCode {
	code: string;
	isUsed: boolean;
	usedAt: string | null;
}

// MyCodesResponse is the payload returned by GET /api/v2/invite/my-codes.
export interface MyCodesResponse {
	codes: InviteCode[];
	totalCodes: number;
	usedCount: number;
	// nextUnlockIn: days remaining before login streak unlocks a new code.
	nextUnlockIn: number;
}

// InvitedUser represents a single invited user entry (name is masked server-side).
export interface InvitedUser {
	invitedUserId: number;
	invitedUserName: string;
	invitedAt: string;
}

// MyInvitesResponse is the payload returned by GET /api/v2/invite/my-invites.
export interface MyInvitesResponse {
	invites: InvitedUser[];
	totalInvited: number;
}
