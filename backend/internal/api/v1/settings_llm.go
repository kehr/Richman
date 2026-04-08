package v1

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// LLMConfigRepo is the narrow persistence interface the settings-llm
// handler depends on. Declared in the handler package (not imported from
// repo) so unit tests can substitute a fake without spinning up pgx.
// The production *repo.LLMConfigRepo satisfies it structurally.
type LLMConfigRepo interface {
	GetActiveByUserID(ctx context.Context, userID int64) (*model.LLMConfig, error)
	Upsert(ctx context.Context, cfg *model.LLMConfig) error
	SoftDelete(ctx context.Context, userID int64, modifier string) error
	UpdateHealth(
		ctx context.Context,
		configID int64,
		status model.LLMHealthStatus,
		lastError *string,
	) error
}

// LLMConsentRepo is the narrow user-consent interface. Same rationale as
// LLMConfigRepo: the handler imports only what it calls so the test fake
// stays small.
type LLMConsentRepo interface {
	GetUseSystemDefaultConsent(ctx context.Context, userID int64) (bool, error)
	SetUseSystemDefaultConsent(ctx context.Context, userID int64, consent bool) error
}

// ClaudeBuilder / OpenAIBuilder are the same closures the Resolver uses
// to construct a live Provider from decrypted plaintext. The handler reuses
// them for probe operations so tests can stub the network call.
type (
	ClaudeBuilder func(apiKey, model string) llm.Provider
	OpenAIBuilder func(baseURL, apiKey, model string) llm.Provider
)

// LLMSettingsHandler owns the user-facing LLM configuration endpoints. The
// crypto instance may be nil in dev environments where no master key is
// set; when nil, every mutating handler short-circuits with a 503 so the
// misconfiguration cannot silently store plaintext. probeTimeout bounds the
// probe round trip so a hung provider cannot stall the HTTP worker.
type LLMSettingsHandler struct {
	configRepo    LLMConfigRepo
	consentRepo   LLMConsentRepo
	crypto        *llm.Crypto
	claudeBuilder ClaudeBuilder
	openaiBuilder OpenAIBuilder
	probeTimeout  time.Duration
	logger        *zap.Logger
}

// LLMSettingsDeps bundles the collaborators the handler needs so main.go
// can construct the handler with a single struct literal.
type LLMSettingsDeps struct {
	ConfigRepo    LLMConfigRepo
	ConsentRepo   LLMConsentRepo
	Crypto        *llm.Crypto
	ClaudeBuilder ClaudeBuilder
	OpenAIBuilder OpenAIBuilder
	ProbeTimeout  time.Duration
	Logger        *zap.Logger
}

// NewLLMSettingsHandler constructs a handler from its dependencies. A nil
// Crypto / builder is tolerated at construction time so main.go can wire
// the handler even when the master key is absent; mutating calls then
// respond with 503 until the operator fixes the env.
func NewLLMSettingsHandler(deps LLMSettingsDeps) *LLMSettingsHandler {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	timeout := deps.ProbeTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &LLMSettingsHandler{
		configRepo:    deps.ConfigRepo,
		consentRepo:   deps.ConsentRepo,
		crypto:        deps.Crypto,
		claudeBuilder: deps.ClaudeBuilder,
		openaiBuilder: deps.OpenAIBuilder,
		probeTimeout:  timeout,
		logger:        logger,
	}
}

// RegisterRoutes wires the settings-llm endpoints under the given router
// group. All routes require authentication.
func (h *LLMSettingsHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	settings := rg.Group("/settings/llm", authMiddleware)
	settings.GET("", h.Get)
	settings.PUT("", h.Put)
	settings.DELETE("", h.Delete)
	settings.POST("/probe", h.Probe)

	// The onboarding consent endpoint is grouped here (rather than in
	// onboarding.go) because it writes the same user column that the
	// settings form reads, and keeping both near each other avoids a
	// cross-handler drift between consent semantics.
	onboarding := rg.Group("/onboarding", authMiddleware)
	onboarding.POST("/llm-consent", h.Consent)
}

// LLMSettingsDTO is the API response shape for GET /api/v1/settings/llm.
// Every provider-specific field is optional so an unconfigured user sees
// only the configured=false flag plus the consent state.
type LLMSettingsDTO struct {
	Configured                       bool    `json:"configured"`
	ProviderType                     *string `json:"providerType,omitempty"`
	BaseURL                          *string `json:"baseUrl,omitempty"`
	Model                            *string `json:"model,omitempty"`
	APIKeyHint                       *string `json:"apiKeyHint,omitempty"`
	UseSystemDefaultWhenUnconfigured bool    `json:"useSystemDefaultWhenUnconfigured"`
	FallbackToSystemDefaultOnFailure bool    `json:"fallbackToSystemDefaultOnFailure"`
	HealthStatus                     *string `json:"healthStatus,omitempty"`
	LastProbeAt                      *string `json:"lastProbeAt,omitempty"`
	LastProbeError                   *string `json:"lastProbeError,omitempty"`
}

// UpsertLLMRequest is the PUT /api/v1/settings/llm request body.
type UpsertLLMRequest struct {
	ProviderType                     string  `json:"providerType" binding:"required"`
	BaseURL                          *string `json:"baseUrl,omitempty"`
	APIKey                           string  `json:"apiKey" binding:"required"`
	Model                            string  `json:"model" binding:"required"`
	FallbackToSystemDefaultOnFailure bool    `json:"fallbackToSystemDefaultOnFailure"`
	Probe                            bool    `json:"probe"`
}

// ProbeResultDTO is the POST /api/v1/settings/llm/probe response.
type ProbeResultDTO struct {
	Healthy   bool    `json:"healthy"`
	Error     *string `json:"error,omitempty"`
	LatencyMs int64   `json:"latencyMs"`
}

// LLMConsentRequest is the POST /api/v1/onboarding/llm-consent body.
type LLMConsentRequest struct {
	UseSystemDefault bool `json:"useSystemDefault"`
}

// Get handles GET /api/v1/settings/llm. Returns the masked config state
// when configured, or an empty response with the consent flag when the
// user has no active row.
func (h *LLMSettingsHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	consent, err := h.consentRepo.GetUseSystemDefaultConsent(ctx, userID)
	if err != nil {
		h.logger.Warn("get llm consent failed", zap.Int64("user_id", userID), zap.Error(err))
		respondInternal(c, "failed to load llm consent")
		return
	}

	cfg, err := h.configRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, llm.ErrConfigNotFound) {
			// Echo the consent flag so the form can render its toggle
			// even before any provider has been configured.
			c.JSON(http.StatusOK, gin.H{
				"data": LLMSettingsDTO{
					Configured:                       false,
					UseSystemDefaultWhenUnconfigured: consent,
				},
			})
			return
		}
		h.logger.Warn("get llm config failed", zap.Int64("user_id", userID), zap.Error(err))
		respondInternal(c, "failed to load llm config")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": maskedLLMSettingsDTO(cfg, consent)})
}

// Put handles PUT /api/v1/settings/llm. Encrypts the api key, optionally
// probes the provider, and upserts the active config atomically via the
// repo transaction.
func (h *LLMSettingsHandler) Put(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	if h.crypto == nil {
		respondError(c, http.StatusServiceUnavailable, "CRYPTO_UNAVAILABLE",
			"llm config master key not configured")
		return
	}

	var req UpsertLLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	providerType, err := validateProviderType(req.ProviderType)
	if err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if strings.TrimSpace(req.Model) == "" {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "model is required")
		return
	}
	if strings.TrimSpace(req.APIKey) == "" {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "apiKey is required")
		return
	}

	// openai_compatible: base_url is required and must pass the SSRF
	// gate. claude/openai reject any base_url to keep the contract narrow
	// (they always hit the official endpoint).
	if providerType == model.ProviderOpenAICompatible {
		if req.BaseURL == nil || strings.TrimSpace(*req.BaseURL) == "" {
			respondError(c, http.StatusBadRequest, "VALIDATION_ERROR",
				"baseUrl is required for openai_compatible")
			return
		}
		if ssrfErr := llm.ValidateBaseURL(*req.BaseURL); ssrfErr != nil {
			respondError(c, http.StatusBadRequest, "SSRF_BLOCKED", ssrfErr.Error())
			return
		}
	}

	// Probe before persisting so a bad key / unreachable host surfaces
	// as a save failure rather than a silently stored zombie config.
	if req.Probe {
		probe := h.runLiveProbe(ctx, providerType, req.BaseURL, req.APIKey, req.Model)
		if !probe.Healthy {
			msg := "probe failed"
			if probe.Error != nil {
				msg = *probe.Error
			}
			respondError(c, http.StatusBadRequest, "PROBE_FAILED", msg)
			return
		}
	}

	ciphertext, nonce, err := h.crypto.Encrypt([]byte(req.APIKey))
	if err != nil {
		h.logger.Error("encrypt llm api key failed", zap.Int64("user_id", userID), zap.Error(err))
		respondInternal(c, "encrypt api key failed")
		return
	}

	actor := strconv.FormatInt(userID, 10)
	cfg := &model.LLMConfig{
		UserID:                           userID,
		ProviderType:                     providerType,
		BaseURL:                          req.BaseURL,
		APIKeyCipher:                     ciphertext,
		APIKeyNonce:                      nonce,
		APIKeyHint:                       maskAPIKey(req.APIKey),
		Model:                            req.Model,
		UseSystemDefaultWhenUnconfigured: false, // Always false when a
		// personal config exists; the column semantic only applies to
		// the "no config" branch which is gated by user consent instead.
		FallbackToSystemDefaultOnFailure: req.FallbackToSystemDefaultOnFailure,
		HealthStatus:                     model.HealthUnknown,
		Creator:                          actor,
		Modifier:                         actor,
	}

	// Probe result propagation: when the PUT body asked for a probe and
	// the probe passed, we already know the provider was live. Stamp
	// health=healthy so the first GET after save does not show "unknown".
	if req.Probe {
		cfg.HealthStatus = model.HealthHealthy
		now := time.Now()
		cfg.LastProbeAt = &now
	}

	if err := h.configRepo.Upsert(ctx, cfg); err != nil {
		h.logger.Error("upsert llm config failed",
			zap.Int64("user_id", userID),
			zap.String("config", cfg.Masked()),
			zap.Error(err),
		)
		respondInternal(c, "failed to save llm config")
		return
	}

	consent, cErr := h.consentRepo.GetUseSystemDefaultConsent(ctx, userID)
	if cErr != nil {
		h.logger.Warn("get llm consent after upsert failed",
			zap.Int64("user_id", userID), zap.Error(cErr))
		consent = false
	}
	c.JSON(http.StatusOK, gin.H{"data": maskedLLMSettingsDTO(cfg, consent)})
}

// Delete handles DELETE /api/v1/settings/llm. Idempotent: calling on a
// user with no active row is a no-op.
func (h *LLMSettingsHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	actor := strconv.FormatInt(userID, 10)
	if err := h.configRepo.SoftDelete(ctx, userID, actor); err != nil {
		h.logger.Error("delete llm config failed",
			zap.Int64("user_id", userID), zap.Error(err))
		respondInternal(c, "failed to delete llm config")
		return
	}
	c.Status(http.StatusNoContent)
}

// Probe handles POST /api/v1/settings/llm/probe. Loads the stored config,
// decrypts the key in-memory, runs a bounded probe, and persists the
// resulting health stamp. Returns 404 when the user has no active config.
func (h *LLMSettingsHandler) Probe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	if h.crypto == nil {
		respondError(c, http.StatusServiceUnavailable, "CRYPTO_UNAVAILABLE",
			"llm config master key not configured")
		return
	}

	cfg, err := h.configRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, llm.ErrConfigNotFound) {
			respondError(c, http.StatusNotFound, "NOT_FOUND", "no llm config")
			return
		}
		h.logger.Warn("probe: get llm config failed",
			zap.Int64("user_id", userID), zap.Error(err))
		respondInternal(c, "failed to load llm config")
		return
	}

	plaintext, err := h.crypto.Decrypt(cfg.APIKeyCipher, cfg.APIKeyNonce)
	if err != nil {
		h.logger.Warn("probe: decrypt failed",
			zap.Int64("user_id", userID),
			zap.String("config", cfg.Masked()),
			zap.Error(err),
		)
		respondError(c, http.StatusInternalServerError, "DECRYPT_FAILED", "decrypt failed")
		return
	}

	result := h.runLiveProbe(ctx, cfg.ProviderType, cfg.BaseURL, string(plaintext), cfg.Model)
	zeroBytes(plaintext)

	var healthErr *string
	if !result.Healthy {
		healthErr = result.Error
	}
	status := model.HealthHealthy
	if !result.Healthy {
		status = model.HealthFailing
	}
	if updateErr := h.configRepo.UpdateHealth(ctx, cfg.ConfigID, status, healthErr); updateErr != nil {
		h.logger.Warn("probe: update health failed",
			zap.Int64("config_id", cfg.ConfigID), zap.Error(updateErr))
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// Consent handles POST /api/v1/onboarding/llm-consent. Writes the users
// column used by the Resolver to decide whether to fall through to the
// system default LLM when a user has no personal config.
func (h *LLMSettingsHandler) Consent(c *gin.Context) {
	userID := middleware.GetUserID(c)
	ctx := c.Request.Context()

	var req LLMConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.consentRepo.SetUseSystemDefaultConsent(ctx, userID, req.UseSystemDefault); err != nil {
		h.logger.Error("set llm consent failed",
			zap.Int64("user_id", userID), zap.Error(err))
		respondInternal(c, "failed to save consent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nil})
}

// runLiveProbe constructs a temporary provider and performs a minimal
// ChatCompletion against it. The plaintext apiKey lives only in this
// goroutine's stack frame; callers that supplied it MUST zeroize their
// own copy after this returns.
func (h *LLMSettingsHandler) runLiveProbe(
	ctx context.Context,
	providerType model.LLMProviderType,
	baseURL *string,
	apiKey, chatModel string,
) ProbeResultDTO {
	// Re-validate base_url on every probe: DNS rebinding defense.
	if providerType == model.ProviderOpenAICompatible {
		if baseURL == nil || strings.TrimSpace(*baseURL) == "" {
			return probeFailure(0, errors.New("baseUrl is required"))
		}
		if ssrfErr := llm.ValidateBaseURL(*baseURL); ssrfErr != nil {
			return probeFailure(0, ssrfErr)
		}
	}

	provider, err := h.buildProvider(providerType, baseURL, apiKey, chatModel)
	if err != nil {
		return probeFailure(0, err)
	}

	probeCtx, cancel := context.WithTimeout(ctx, h.probeTimeout)
	defer cancel()

	start := time.Now()
	_, callErr := provider.ChatCompletion(probeCtx, llm.ChatRequest{
		SystemPrompt: "You are a ping utility. Respond with 'ok'.",
		UserPrompt:   "ping",
		MaxTokens:    16,
		Temperature:  0,
	})
	latency := time.Since(start).Milliseconds()
	if callErr != nil {
		return probeFailure(latency, callErr)
	}
	return ProbeResultDTO{Healthy: true, LatencyMs: latency}
}

// buildProvider translates a provider type + credentials into the live
// llm.Provider closure. Mirrors the Resolver's buildUserProvider so probe
// and live-call paths agree on which builder owns which provider type.
func (h *LLMSettingsHandler) buildProvider(
	providerType model.LLMProviderType,
	baseURL *string,
	apiKey, chatModel string,
) (llm.Provider, error) {
	switch providerType {
	case model.ProviderClaude:
		if h.claudeBuilder == nil {
			return nil, errors.New("claude builder not configured")
		}
		return h.claudeBuilder(apiKey, chatModel), nil
	case model.ProviderOpenAI:
		if h.openaiBuilder == nil {
			return nil, errors.New("openai builder not configured")
		}
		return h.openaiBuilder("", apiKey, chatModel), nil
	case model.ProviderOpenAICompatible:
		if h.openaiBuilder == nil {
			return nil, errors.New("openai builder not configured")
		}
		if baseURL == nil {
			return nil, errors.New("baseUrl is required for openai_compatible")
		}
		return h.openaiBuilder(*baseURL, apiKey, chatModel), nil
	default:
		return nil, fmt.Errorf("unknown provider type %q", providerType)
	}
}

// probeFailure converts a probe error into the DTO the handler returns to
// the client. The error message is stringified here so the caller never
// leaks typed provider errors over the wire.
func probeFailure(latencyMs int64, err error) ProbeResultDTO {
	msg := err.Error()
	return ProbeResultDTO{Healthy: false, Error: &msg, LatencyMs: latencyMs}
}

// maskedLLMSettingsDTO projects a persisted config onto the GET response.
// Every secret-adjacent field is already encrypted on disk; this layer
// pulls only the masked fields the frontend is allowed to see.
func maskedLLMSettingsDTO(cfg *model.LLMConfig, consent bool) LLMSettingsDTO {
	providerType := string(cfg.ProviderType)
	chatModel := cfg.Model
	hint := cfg.APIKeyHint
	healthStatus := string(cfg.HealthStatus)
	var lastProbeAt *string
	if cfg.LastProbeAt != nil {
		t := cfg.LastProbeAt.UTC().Format(time.RFC3339)
		lastProbeAt = &t
	}
	return LLMSettingsDTO{
		Configured:                       true,
		ProviderType:                     &providerType,
		BaseURL:                          cfg.BaseURL,
		Model:                            &chatModel,
		APIKeyHint:                       &hint,
		UseSystemDefaultWhenUnconfigured: consent,
		FallbackToSystemDefaultOnFailure: cfg.FallbackToSystemDefaultOnFailure,
		HealthStatus:                     &healthStatus,
		LastProbeAt:                      lastProbeAt,
		LastProbeError:                   cfg.LastProbeError,
	}
}

// maskAPIKey returns a short hint suitable for display: "..." + last 4
// characters. Short keys (<= 4 chars) collapse to "****" so we never
// reveal the entire secret on a misconfiguration.
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 4 {
		return "****"
	}
	return "..." + apiKey[len(apiKey)-4:]
}

// validateProviderType returns the typed enum value for a caller-supplied
// string, or an error with a stable 400-friendly message.
func validateProviderType(raw string) (model.LLMProviderType, error) {
	switch model.LLMProviderType(raw) {
	case model.ProviderClaude, model.ProviderOpenAI, model.ProviderOpenAICompatible:
		return model.LLMProviderType(raw), nil
	default:
		return "", fmt.Errorf("providerType must be claude, openai, or openai_compatible")
	}
}

// zeroBytes overwrites b with zeros so decrypted plaintext does not linger
// on the heap any longer than the probe window requires. Kept as a local
// helper rather than importing the one in llm/resolver.go to avoid making
// that symbol public.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// respondError writes a standard error envelope with the given status.
func respondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

// respondInternal writes a 500 error envelope with a generic message so
// internal failures do not leak server-side details to the client.
func respondInternal(c *gin.Context, loggedMessage string) {
	respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", loggedMessage)
}
