// Dashboard summary DTOs returned by GET /api/v1/dashboard/summary.
//
// The degraded-contract PRD introduced the `llmStatus` subtree which every
// dashboard-adjacent surface (banner, reanalyze CTA, decision card wall)
// reads from to render provenance affordances. Keeping the DTO local to the
// dashboard-summary feature means the decision-card feature does not need to
// reach across feature boundaries to consume it.

export type LLMProviderHealth = "healthy" | "failing" | "not_configured";

// LLMStatusDTO mirrors the `llmStatus` subtree of the backend
// dashboard-summary response. `configured` is true iff the user has an
// active row in llm_configs; `userProviderHealth` reflects the result of
// the last probe; `systemDefaultAvailable` is true when the backend's
// LLM_DEFAULT_* env is filled and the provider is reachable;
// `needsReanalysis` is true when at least one latest decision card still
// has synthesisSource ∈ (template, mixed) while an LLM layer (user or
// system-default, depending on consent) is available to upgrade it.
export interface LLMStatusDTO {
	configured: boolean;
	userProviderHealth: LLMProviderHealth;
	systemDefaultAvailable: boolean;
	needsReanalysis: boolean;
}

// DashboardSummaryDTO is the full payload surfaced by
// GET /api/v1/dashboard/summary. At the moment only `llmStatus` is wired
// through because the dashboard page assembles the rest of its numbers from
// decision cards and holdings; the DTO intentionally leaves room for future
// summary fields (e.g. aggregate PnL, top movers) without another schema
// round trip.
export interface DashboardSummaryDTO {
	llmStatus: LLMStatusDTO;
}
