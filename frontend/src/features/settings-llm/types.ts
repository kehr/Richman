// Settings-LLM DTOs mirror the handlers in
// backend/internal/api/v1/settings_llm.go. The field names are kept in
// camelCase to match the Go json tags returned by the backend. Every DTO is
// strictly typed — we never expose the api_key plaintext back to the UI;
// only the `apiKeyHint` (masked last-4) field is ever available.

export type LLMProviderType = "claude" | "openai" | "openai_compatible";

export type LLMHealthStatus = "healthy" | "failing" | "unknown";

// LLMSettingsDTO is the GET /api/v1/settings/llm response. When
// `configured` is false the provider-specific fields are absent and only
// `useSystemDefaultWhenUnconfigured` is meaningful (it is always echoed
// from the users table so the Settings form can render the toggle even on
// an empty config).
export interface LLMSettingsDTO {
	configured: boolean;
	providerType?: LLMProviderType;
	baseUrl?: string | null;
	model?: string;
	apiKeyHint?: string;
	useSystemDefaultWhenUnconfigured: boolean;
	fallbackToSystemDefaultOnFailure: boolean;
	healthStatus?: LLMHealthStatus;
	lastProbeAt?: string | null;
	lastProbeError?: string | null;
}

// UpsertLLMRequest is the PUT /api/v1/settings/llm body. `apiKey` is
// required on create and optional on edit (empty means "leave unchanged").
// `baseUrl` is only meaningful when `providerType === "openai_compatible"`.
// `probe` defaults to true in the UI so the user sees immediate feedback
// after a save.
export interface UpsertLLMRequest {
	providerType: LLMProviderType;
	baseUrl?: string;
	apiKey?: string;
	model: string;
	fallbackToSystemDefaultOnFailure: boolean;
	probe?: boolean;
}

// ProbeResultDTO is the POST /api/v1/settings/llm/probe response. The
// backend runs a bounded probe against the stored config (decrypting the
// key in-memory only) and returns a boolean + optional error.
export interface ProbeResultDTO {
	healthy: boolean;
	error: string | null;
	latencyMs: number;
}

// LLMConsentRequest is the POST /api/v1/onboarding/llm-consent body. The
// backend writes users.use_system_default_llm_consent to this value.
export interface LLMConsentRequest {
	useSystemDefault: boolean;
}
