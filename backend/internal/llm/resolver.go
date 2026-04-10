package llm

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/richman/backend/internal/model"
)

// ProviderLayer identifies which layer of the fallback chain actually served
// a request. The string form is written to decision_cards.provider_used so
// the dashboard banner can explain degraded analyses. Values are part of the
// API contract and MUST stay stable.
type ProviderLayer string

// Layer constants. The zero value is LayerNone so a default-constructed
// ResolvedResponse does not accidentally claim a real provider served it.
const (
	LayerUser          ProviderLayer = "user"
	LayerSystemDefault ProviderLayer = "system_default"
	LayerNone          ProviderLayer = "none"
)

// ResolvedResponse wraps a ChatResponse with the layer that produced it.
// The caller uses Layer to stamp decision_cards.provider_used and to decide
// whether to set synthesis_source to "llm" (any non-None layer) or
// "template" (returned via error, no ResolvedResponse at all).
type ResolvedResponse struct {
	Response *ChatResponse
	Layer    ProviderLayer
}

// Resolver encapsulates the three-level fallback chain:
//
//	user -> system_default -> error (caller falls back to template)
//
// Implementations MUST be safe for concurrent use across goroutines; the
// production resolverImpl achieves this because its dependencies (pgxpool,
// Crypto, stateless Provider closures) are all concurrent-safe.
type Resolver interface {
	// ResolvedChatCompletion runs the fallback chain for a specific user.
	// Returns (*ResolvedResponse, nil) when any layer serves the request.
	// Returns (nil, error) when every candidate layer is unusable — the
	// caller then renders a template card.
	ResolvedChatCompletion(
		ctx context.Context, userID int64, req ChatRequest,
	) (*ResolvedResponse, error)
}

// LLMConfigRepo is the narrow interface Resolver depends on. Declared here
// (not in repo package) so Resolver tests can substitute an in-memory fake
// without pulling in pgxpool or the full repo surface. The production
// *repo.LLMConfigRepo value satisfies this interface structurally.
type LLMConfigRepo interface {
	GetActiveByUserID(ctx context.Context, userID int64) (*model.LLMConfig, error)
	UpdateHealth(
		ctx context.Context,
		configID int64,
		status model.LLMHealthStatus,
		lastError *string,
	) error
}

// UserConsentRepo is the narrow interface Resolver needs from the user repo:
// a single read of use_system_default_llm_consent for the current user. Kept
// separate from the full UserRepo so the Resolver tests do not have to mock
// every unrelated user method.
type UserConsentRepo interface {
	GetUseSystemDefaultConsent(ctx context.Context, userID int64) (bool, error)
}

// resolverImpl is the production Resolver. Its fields are immutable after
// construction; the only mutation is the UpdateHealth side effect on the
// config repo, which is itself concurrent-safe.
type resolverImpl struct {
	configRepo    LLMConfigRepo
	consentRepo   UserConsentRepo
	crypto        *Crypto
	systemDefault Provider // may be nil when no sys default is configured

	// claudeBuilder and openaiBuilder are injected so resolver.go does not
	// need to import llm/claude or llm/openai (which would create a cycle
	// because those packages import llm for the Provider interface).
	claudeBuilder func(apiKey, model string) Provider
	openaiBuilder func(baseURL, apiKey, model string) Provider

	probeTimeout time.Duration //nolint:unused // reserved for bounded probe retries in later phases
	logger       *zap.Logger
}

// NewResolver wires a production Resolver with the given collaborators.
// systemDefault may be nil — the Resolver will skip the second layer and
// return ErrAllLayersFailed instead of dereferencing a nil Provider.
func NewResolver(
	configRepo LLMConfigRepo,
	consentRepo UserConsentRepo,
	crypto *Crypto,
	systemDefault Provider,
	claudeBuilder func(apiKey, model string) Provider,
	openaiBuilder func(baseURL, apiKey, model string) Provider,
	probeTimeout time.Duration,
	logger *zap.Logger,
) Resolver {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &resolverImpl{
		configRepo:    configRepo,
		consentRepo:   consentRepo,
		crypto:        crypto,
		systemDefault: systemDefault,
		claudeBuilder: claudeBuilder,
		openaiBuilder: openaiBuilder,
		probeTimeout:  probeTimeout,
		logger:        logger,
	}
}

// ResolvedChatCompletion executes the three-level fallback chain. The logic
// is a linear walk through the state space enumerated in the PRD:
//
//  1. Look up the user's active config. If present, try to build a Provider
//     and call it. On success, stamp health=healthy and return LayerUser.
//     On failure (build or call), stamp health=failing and decide whether
//     to cascade based on FallbackToSystemDefaultOnFailure.
//  2. If the user has no config, read use_system_default_llm_consent. If
//     false, return ErrConsentDenied so the caller uses template output
//     without treating it as an outage.
//  3. Attempt systemDefault. If it is nil or errors, return
//     ErrAllLayersFailed so the caller renders a template card.
//
// The error contract is intentionally narrow: callers distinguish only
// ErrConsentDenied, ErrAllLayersFailed, and "any other error" (e.g. the
// propagated user call error when fallback is off).
func (r *resolverImpl) ResolvedChatCompletion(
	ctx context.Context,
	userID int64,
	req ChatRequest,
) (*ResolvedResponse, error) {
	cfg, lookupErr := r.configRepo.GetActiveByUserID(ctx, userID)
	if lookupErr != nil && !errors.Is(lookupErr, ErrConfigNotFound) {
		// Unexpected DB error reading the config. Log with masked context
		// and fall through to the "user absent" branch so a transient
		// Postgres hiccup cannot black out every analysis indefinitely.
		r.logger.Warn("llm config lookup failed",
			zap.Int64("user_id", userID),
			zap.Error(lookupErr),
		)
		cfg = nil
	}

	// Layer 1: user provider
	if cfg != nil {
		if resp, layerErr, fellThrough := r.tryUserProvider(ctx, cfg, req); !fellThrough {
			return resp, layerErr
		}
	} else {
		// No user config: consent gate decides whether we continue.
		consent, consentErr := r.consentRepo.GetUseSystemDefaultConsent(ctx, userID)
		if consentErr != nil {
			r.logger.Warn("consent lookup failed",
				zap.Int64("user_id", userID),
				zap.Error(consentErr),
			)
			return nil, ErrConsentDenied
		}
		if !consent {
			return nil, ErrConsentDenied
		}
	}

	// Layer 2: system default
	if r.systemDefault == nil {
		return nil, ErrAllLayersFailed
	}
	resp, callErr := r.systemDefault.ChatCompletion(ctx, req)
	if callErr != nil {
		r.logger.Warn("system_default provider failed",
			zap.Int64("user_id", userID),
			zap.String("provider", r.systemDefault.Name()),
			zap.Error(callErr),
		)
		return nil, ErrAllLayersFailed
	}
	return &ResolvedResponse{Response: resp, Layer: LayerSystemDefault}, nil
}

// tryUserProvider exercises the user config layer. The third return value
// is true when the caller should continue to the system_default layer; the
// first two values are meaningful only when fellThrough is false, in which
// case they are the terminal resolver result.
//
// Breaking this out of ResolvedChatCompletion keeps the top-level walk
// linear and lets the SSRF / build / call / FallbackToSystemDefaultOnFailure
// decision tree live in one place.
func (r *resolverImpl) tryUserProvider(
	ctx context.Context,
	cfg *model.LLMConfig,
	req ChatRequest,
) (resp *ResolvedResponse, err error, fellThrough bool) {
	userProvider, buildErr := r.buildUserProvider(cfg)
	if buildErr != nil {
		r.logger.Warn("user provider build failed",
			zap.Int64("user_id", cfg.UserID),
			zap.String("config", cfg.Masked()),
			zap.Error(buildErr),
		)
		if !cfg.FallbackToSystemDefaultOnFailure {
			return nil, buildErr, false
		}
		return nil, nil, true
	}

	response, callErr := userProvider.ChatCompletion(ctx, req)
	if callErr == nil {
		if updateErr := r.configRepo.UpdateHealth(
			ctx, cfg.ConfigID, model.HealthHealthy, nil,
		); updateErr != nil {
			// Non-fatal: log and keep the successful response. A failed
			// health write cannot be allowed to mask a live LLM answer.
			r.logger.Warn("update health (healthy) failed",
				zap.Int64("config_id", cfg.ConfigID),
				zap.Error(updateErr),
			)
		}
		return &ResolvedResponse{Response: response, Layer: LayerUser}, nil, false
	}

	errStr := callErr.Error()
	if updateErr := r.configRepo.UpdateHealth(
		ctx, cfg.ConfigID, model.HealthFailing, &errStr,
	); updateErr != nil {
		r.logger.Warn("update health (failing) failed",
			zap.Int64("config_id", cfg.ConfigID),
			zap.Error(updateErr),
		)
	}
	r.logger.Warn("user provider failed",
		zap.Int64("user_id", cfg.UserID),
		zap.String("config", cfg.Masked()),
		zap.Error(callErr),
	)

	if !cfg.FallbackToSystemDefaultOnFailure {
		return nil, callErr, false
	}
	return nil, nil, true
}

// buildUserProvider decrypts the cfg's api key, runs the SSRF gate when the
// provider type is openai_compatible, and hands the plaintext to the
// injected builder closure. The plaintext slice is zeroized via defer so
// even an unexpected panic in the builder cannot leave it on the heap.
//
// Note: this is where we re-run ValidateBaseURL on every live call. The
// Phase 1 design memo explicitly calls this out as a DNS-rebinding defense,
// so DO NOT cache the validation result between calls.
func (r *resolverImpl) buildUserProvider(cfg *model.LLMConfig) (Provider, error) {
	plaintext, err := r.crypto.Decrypt(cfg.APIKeyCipher, cfg.APIKeyNonce)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(plaintext)

	switch cfg.ProviderType {
	case model.ProviderClaude:
		return r.claudeBuilder(string(plaintext), cfg.Model), nil
	case model.ProviderOpenAI:
		return r.openaiBuilder("", string(plaintext), cfg.Model), nil
	case model.ProviderOpenAICompatible:
		if cfg.BaseURL == nil {
			return nil, ErrConfigDamaged
		}
		if validateErr := ValidateSelfHostedBaseURL(*cfg.BaseURL); validateErr != nil {
			return nil, validateErr
		}
		return r.openaiBuilder(*cfg.BaseURL, string(plaintext), cfg.Model), nil
	default:
		return nil, ErrConfigDamaged
	}
}

// zeroBytes overwrites b with zeros so decrypted plaintext does not linger
// on the heap any longer than the decrypt/build window requires.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
