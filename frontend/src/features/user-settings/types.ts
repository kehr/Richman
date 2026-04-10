// User settings DTOs mirror backend/internal/service/user_settings.UserSettings
// and onboarding.Status, plus the sparse PATCH payload. All fields use
// camelCase to match the Go json tags returned from /api/v1/user/settings.

export type RiskPreference = "conservative" | "neutral" | "aggressive";

export type Language = "en" | "zh";

export interface UserSettings {
	userId: number;
	totalCapitalCny?: number | null;
	riskPreference: RiskPreference;
	categories: string[];
	language: Language;
	onboardingCompleted: boolean;
	onboardingCompletedAt?: string | null;
}

// PatchUserSettings is a sparse update. Undefined fields mean "leave
// unchanged". To clear the total capital back to null set
// clearTotalCapitalCny=true and omit totalCapitalCny.
export interface PatchUserSettings {
	totalCapitalCny?: number;
	clearTotalCapitalCny?: boolean;
	riskPreference?: RiskPreference;
	categories?: string[];
	language?: Language;
}

export interface OnboardingStatus {
	completed: boolean;
	completedAt?: string | null;
	skipped: boolean;
	skippedAt?: string | null;
}
