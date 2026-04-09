package model

import (
	"fmt"
	"time"
)

// LLMProviderType enumerates the provider shapes the Resolver knows how to
// speak. Values MUST stay in sync with the llm_configs.provider_type CHECK
// constraint defined in migration 011_llm_configs.up.sql.
type LLMProviderType string

// Provider type string constants. These are the only values the CHECK
// constraint accepts; adding a new value requires a migration update.
const (
	ProviderClaude           LLMProviderType = "claude"
	ProviderOpenAI           LLMProviderType = "openai"
	ProviderOpenAICompatible LLMProviderType = "openai_compatible"
)

// LLMHealthStatus tracks the result of the most recent connectivity probe
// against a user's configured provider. Values MUST stay in sync with the
// llm_configs.health_status CHECK constraint in migration 011.
type LLMHealthStatus string

// Health status string constants. "unknown" is the default before any probe
// has run; "healthy" / "failing" come from ProbeConnectivity results.
const (
	HealthHealthy LLMHealthStatus = "healthy"
	HealthFailing LLMHealthStatus = "failing"
	HealthUnknown LLMHealthStatus = "unknown"
)

// LLMConfig is the persistence model for a user's LLM provider configuration.
//
// Safety contract: the APIKeyCipher and APIKeyNonce byte slices are the
// encrypted form of the user's plaintext API key. They MUST NEVER be written
// to a log, returned over the API, embedded in an error message, or printed
// via %v / %+v. Any callsite that needs to produce a string representation
// of an LLMConfig for logging or metrics MUST use the Masked method below.
//
// LLMConfig deliberately does NOT implement Stringer so that a stray
// fmt.Printf("%v", cfg) cannot accidentally dump the ciphertext bytes.
type LLMConfig struct {
	ConfigID                         int64           `json:"configId" db:"config_id"`
	UserID                           int64           `json:"userId" db:"user_id"`
	ProviderType                     LLMProviderType `json:"providerType" db:"provider_type"`
	BaseURL                          *string         `json:"baseUrl,omitempty" db:"base_url"`
	APIKeyCipher                     []byte          `json:"-" db:"api_key_cipher"`
	APIKeyNonce                      []byte          `json:"-" db:"api_key_nonce"`
	APIKeyHint                       string          `json:"apiKeyHint" db:"api_key_hint"`
	Model                            string          `json:"model" db:"model"`
	UseSystemDefaultWhenUnconfigured bool            `json:"useSystemDefaultWhenUnconfigured" db:"use_system_default_when_unconfigured"`  //nolint:revive // long tag name mirrors column
	FallbackToSystemDefaultOnFailure bool            `json:"fallbackToSystemDefaultOnFailure" db:"fallback_to_system_default_on_failure"` //nolint:revive // long tag name mirrors column
	HealthStatus                     LLMHealthStatus `json:"healthStatus" db:"health_status"`
	LastProbeAt                      *time.Time      `json:"lastProbeAt,omitempty" db:"last_probe_at"`
	LastProbeError                   *string         `json:"lastProbeError,omitempty" db:"last_probe_error"`
	CreatedAt                        time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt                        time.Time       `json:"updatedAt" db:"updated_at"`
	Creator                          string          `json:"-" db:"creator"`
	Modifier                         string          `json:"-" db:"modifier"`
	IsDeleted                        int16           `json:"-" db:"is_deleted"`
}

// Masked returns a log-safe string representation that NEVER includes the
// plaintext api key or the ciphertext bytes. This is the ONLY method callers
// may use to serialize LLMConfig for structured logs, errors, and metrics.
// The returned string is stable enough to be grep-friendly and short enough
// to embed in a zap log field.
func (c *LLMConfig) Masked() string {
	if c == nil {
		return "LLMConfig{nil}"
	}
	baseURL := ""
	if c.BaseURL != nil {
		baseURL = *c.BaseURL
	}
	return fmt.Sprintf(
		"LLMConfig{user=%d type=%s model=%s base_url=%s key_hint=%s health=%s}",
		c.UserID, c.ProviderType, c.Model, baseURL, c.APIKeyHint, c.HealthStatus,
	)
}
