package llm

import "errors"

// Resolver layer sentinel errors. These are the stable contract the caller
// layers (Synthesizer, Analysis Service) inspect to decide between live LLM
// output, template fallback, and hard-fail reporting. Every value MUST be
// matched with errors.Is so future wrapping does not break callers.
//
// Grouped here instead of next to Resolver because ErrConfigNotFound is also
// needed by the repo layer, which must not import the resolver file to avoid
// an import cycle. The repo package imports llm for these sentinels only.
var (
	// ErrConfigNotFound signals that the user has no active llm_configs row.
	// Repo layer normalizes pgx.ErrNoRows into this error so the Resolver
	// never sees pgx-specific details. Callers use it to distinguish
	// "user never configured" from a real DB failure.
	ErrConfigNotFound = errors.New("llm: config not found")

	// ErrConsentDenied signals that the user has no personal provider AND
	// has not granted use_system_default_llm_consent. The Resolver refuses
	// to fall through to the system default, and the caller must interpret
	// this as "render a template card" without treating it as an outage.
	ErrConsentDenied = errors.New("llm: user has not consented to system default")

	// ErrAllLayersFailed is returned when every candidate layer in the
	// fallback chain is unusable (user failed or absent-without-consent,
	// system default absent or failing). The caller should render a
	// template card and stamp synthesis_source = "template".
	ErrAllLayersFailed = errors.New("llm: all provider layers failed")

	// ErrConfigDamaged covers malformed persisted configs that survived
	// validation but cannot build a Provider — e.g. openai_compatible with
	// a nil base_url or an unknown provider_type. Kept distinct from
	// ErrDecryptFailed so health probes can classify the root cause.
	ErrConfigDamaged = errors.New("llm: config is damaged or unsupported")
)
