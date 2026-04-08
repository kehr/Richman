// User mirrors the backend model.User struct JSON payload (see
// backend/internal/model/user.go). Field names, casing, and optionality
// must stay in lock-step with the backend marshaller — the /auth/me and
// /auth/login / /auth/register endpoints all return this shape.
export interface User {
	userId: number;
	email: string;
	role: string;
	planId?: number | null;
	riskPreference: string;
	totalCapitalCny?: number | null;
	onboardingCompletedAt?: string | null;
	categories: string[];
	createdAt: string;
	updatedAt: string;
}
