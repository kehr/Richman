package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/richman/backend/internal/api/middleware"
	"github.com/richman/backend/internal/llm"
	"github.com/richman/backend/internal/model"
	"go.uber.org/zap"
)

// fakeLLMConfigRepo stores a single active config in-memory keyed by
// user id. It satisfies v1.LLMConfigRepo so the handler can be driven
// through its public surface without touching pgx.
type fakeLLMConfigRepo struct {
	byUser map[int64]*model.LLMConfig
	upErr  error
	delErr error
}

func newFakeLLMConfigRepo() *fakeLLMConfigRepo {
	return &fakeLLMConfigRepo{byUser: map[int64]*model.LLMConfig{}}
}

func (f *fakeLLMConfigRepo) GetActiveByUserID(
	_ context.Context, userID int64,
) (*model.LLMConfig, error) {
	cfg, ok := f.byUser[userID]
	if !ok {
		return nil, llm.ErrConfigNotFound
	}
	cp := *cfg
	return &cp, nil
}

func (f *fakeLLMConfigRepo) Upsert(_ context.Context, cfg *model.LLMConfig) error {
	if f.upErr != nil {
		return f.upErr
	}
	cfg.ConfigID = int64(len(f.byUser) + 1)
	now := time.Now()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now
	cp := *cfg
	f.byUser[cfg.UserID] = &cp
	return nil
}

func (f *fakeLLMConfigRepo) SoftDelete(_ context.Context, userID int64, _ string) error {
	if f.delErr != nil {
		return f.delErr
	}
	delete(f.byUser, userID)
	return nil
}

func (f *fakeLLMConfigRepo) UpdateHealth(
	_ context.Context, configID int64, status model.LLMHealthStatus, lastError *string,
) error {
	for _, cfg := range f.byUser {
		if cfg.ConfigID != configID {
			continue
		}
		cfg.HealthStatus = status
		cfg.LastProbeError = lastError
		now := time.Now()
		cfg.LastProbeAt = &now
		return nil
	}
	return nil
}

// fakeConsentRepo stores a single boolean per user.
type fakeConsentRepo struct {
	byUser map[int64]bool
}

func newFakeConsentRepo() *fakeConsentRepo {
	return &fakeConsentRepo{byUser: map[int64]bool{}}
}

func (f *fakeConsentRepo) GetUseSystemDefaultConsent(
	_ context.Context, userID int64,
) (bool, error) {
	return f.byUser[userID], nil
}

func (f *fakeConsentRepo) SetUseSystemDefaultConsent(
	_ context.Context, userID int64, consent bool,
) error {
	f.byUser[userID] = consent
	return nil
}

// fakeProbeProvider lets tests assert the probe code path without any
// network traffic. The name is returned verbatim so assertions on which
// builder was invoked stay trivial.
type fakeProbeProvider struct {
	name string
	err  error
}

func (f *fakeProbeProvider) ChatCompletion(
	_ context.Context, _ llm.ChatRequest,
) (*llm.ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &llm.ChatResponse{Content: "ok"}, nil
}

func (f *fakeProbeProvider) Name() string { return f.name }

// newTestCrypto builds a Crypto instance from a fixed hex key so tests
// do not depend on environment state.
func newTestCrypto(t *testing.T) *llm.Crypto {
	t.Helper()
	// 32 bytes hex-encoded. Value is test-only; never reused in prod.
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	c, err := llm.NewCryptoFromHex(key)
	if err != nil {
		t.Fatalf("new crypto: %v", err)
	}
	return c
}

type llmHandlerBuilder struct {
	configRepo  *fakeLLMConfigRepo
	consentRepo *fakeConsentRepo
	crypto      *llm.Crypto
	probeErr    error
}

func (b *llmHandlerBuilder) build(t *testing.T) *LLMSettingsHandler {
	t.Helper()
	if b.configRepo == nil {
		b.configRepo = newFakeLLMConfigRepo()
	}
	if b.consentRepo == nil {
		b.consentRepo = newFakeConsentRepo()
	}
	if b.crypto == nil {
		b.crypto = newTestCrypto(t)
	}
	return NewLLMSettingsHandler(LLMSettingsDeps{
		ConfigRepo:  b.configRepo,
		ConsentRepo: b.consentRepo,
		Crypto:      b.crypto,
		ClaudeBuilder: func(_, _ string) llm.Provider {
			return &fakeProbeProvider{name: "claude", err: b.probeErr}
		},
		OpenAIBuilder: func(_, _, _ string) llm.Provider {
			return &fakeProbeProvider{name: "openai", err: b.probeErr}
		},
		ProbeTimeout: time.Second,
		Logger:       zap.NewNop(),
	})
}

func newLLMTestRouter(h *LLMSettingsHandler, authedUserID int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		if authedUserID <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "auth required"},
			})
			return
		}
		c.Set(middleware.ContextKeyUserID, authedUserID)
		c.Next()
	}
	h.RegisterRoutes(r.Group("/api/v1"), auth)
	return r
}

func decodeLLMSettings(t *testing.T, body []byte) LLMSettingsDTO {
	t.Helper()
	var envelope struct {
		Data LLMSettingsDTO `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return envelope.Data
}

func decodeProbeResult(t *testing.T, body []byte) ProbeResultDTO {
	t.Helper()
	var envelope struct {
		Data ProbeResultDTO `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, string(body))
	}
	return envelope.Data
}

// mustServe runs a single request against the router and returns the
// recorder so each test can assert on status and body.
func mustServe(t *testing.T, r *gin.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	var req *http.Request
	if reader == nil {
		req = httptest.NewRequest(method, path, http.NoBody)
	} else {
		req = httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLLMSettings_GetUnconfigured(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	b.consentRepo.byUser[7] = true
	r := newLLMTestRouter(h, 7)

	w := mustServe(t, r, http.MethodGet, "/api/v1/settings/llm", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200 got %d body=%s", w.Code, w.Body.String())
	}
	got := decodeLLMSettings(t, w.Body.Bytes())
	if got.Configured {
		t.Error("expected configured=false")
	}
	if !got.UseSystemDefaultWhenUnconfigured {
		t.Error("expected consent echoed as true")
	}
}

func TestLLMSettings_PutAndGetClaude(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	body := []byte(`{
        "providerType": "claude",
        "apiKey": "sk-ant-abcd1234",
        "model": "claude-sonnet-4-20250514",
        "fallbackToSystemDefaultOnFailure": true,
        "probe": true
    }`)
	w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body)
	if w.Code != http.StatusOK {
		t.Fatalf("put status: want 200 got %d body=%s", w.Code, w.Body.String())
	}
	got := decodeLLMSettings(t, w.Body.Bytes())
	if !got.Configured {
		t.Error("expected configured=true after put")
	}
	if got.APIKeyHint == nil || *got.APIKeyHint != "...1234" {
		t.Errorf("expected hint=...1234, got %+v", got.APIKeyHint)
	}
	if got.HealthStatus == nil || *got.HealthStatus != string(model.HealthHealthy) {
		t.Errorf("expected healthy after probe, got %+v", got.HealthStatus)
	}

	// A subsequent GET should return the masked form.
	w2 := mustServe(t, r, http.MethodGet, "/api/v1/settings/llm", nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("get status: want 200 got %d", w2.Code)
	}
	got2 := decodeLLMSettings(t, w2.Body.Bytes())
	if !got2.Configured {
		t.Error("expected configured=true on subsequent get")
	}
}

func TestLLMSettings_PutOpenAICompatible_BadScheme(t *testing.T) {
	// openai_compatible uses the relaxed SSRF validator: http and https are
	// both accepted, but non-HTTP schemes (ftp, file, etc.) must be rejected.
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	body := []byte(`{
        "providerType": "openai_compatible",
        "baseUrl": "ftp://example.com/v1",
        "apiKey": "sk-local-1234",
        "model": "llama3",
        "fallbackToSystemDefaultOnFailure": false,
        "probe": false
    }`)
	w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for ftp scheme, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLLMSettings_PutOpenAICompatible_HTTP_LocalhostAllowed(t *testing.T) {
	// openai_compatible providers like Ollama run on http and localhost;
	// both should be accepted by the relaxed SSRF validator.
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	body := []byte(`{
        "providerType": "openai_compatible",
        "baseUrl": "http://localhost:11434/v1",
        "apiKey": "sk-local-1234",
        "model": "llama3",
        "fallbackToSystemDefaultOnFailure": false,
        "probe": false
    }`)
	w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 for http localhost (self-hosted), got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLLMSettings_PutProbeFailed(t *testing.T) {
	b := &llmHandlerBuilder{probeErr: errors.New("auth failed")}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	body := []byte(`{
        "providerType": "claude",
        "apiKey": "sk-bad",
        "model": "claude-sonnet-4-20250514",
        "fallbackToSystemDefaultOnFailure": false,
        "probe": true
    }`)
	w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 on probe failure, got %d body=%s", w.Code, w.Body.String())
	}
	if len(b.configRepo.byUser) != 0 {
		t.Errorf("expected no config persisted after probe failure")
	}
}

func TestLLMSettings_PutMissingRequired(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	body := []byte(`{"providerType":"claude","apiKey":"sk-only"}`)
	w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 on missing model, got %d", w.Code)
	}
}

func TestLLMSettings_Delete(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	// Seed an existing config via Put.
	body := []byte(`{
        "providerType": "claude",
        "apiKey": "sk-ant-abcd1234",
        "model": "claude-sonnet-4-20250514",
        "fallbackToSystemDefaultOnFailure": false,
        "probe": false
    }`)
	if w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body); w.Code != http.StatusOK {
		t.Fatalf("seed put failed: %d body=%s", w.Code, w.Body.String())
	}
	if len(b.configRepo.byUser) != 1 {
		t.Fatalf("seed expected 1 config, got %d", len(b.configRepo.byUser))
	}

	w := mustServe(t, r, http.MethodDelete, "/api/v1/settings/llm", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if len(b.configRepo.byUser) != 0 {
		t.Errorf("expected 0 configs after delete, got %d", len(b.configRepo.byUser))
	}

	// Repeating the DELETE is idempotent and still returns 204.
	w2 := mustServe(t, r, http.MethodDelete, "/api/v1/settings/llm", nil)
	if w2.Code != http.StatusNoContent {
		t.Fatalf("want 204 on idempotent delete, got %d", w2.Code)
	}
}

func TestLLMSettings_ProbeSuccess(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	// Seed via Put.
	body := []byte(`{
        "providerType": "claude",
        "apiKey": "sk-ant-abcd1234",
        "model": "claude-sonnet-4-20250514",
        "fallbackToSystemDefaultOnFailure": false,
        "probe": false
    }`)
	if w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body); w.Code != http.StatusOK {
		t.Fatalf("seed put failed: %d body=%s", w.Code, w.Body.String())
	}

	w := mustServe(t, r, http.MethodPost, "/api/v1/settings/llm/probe", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("probe want 200 got %d body=%s", w.Code, w.Body.String())
	}
	res := decodeProbeResult(t, w.Body.Bytes())
	if !res.Healthy {
		t.Errorf("expected healthy=true, got %+v", res)
	}
}

func TestLLMSettings_ProbeWithoutConfig(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	w := mustServe(t, r, http.MethodPost, "/api/v1/settings/llm/probe", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestLLMSettings_Consent(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 7)

	body := []byte(`{"useSystemDefault": true}`)
	w := mustServe(t, r, http.MethodPost, "/api/v1/onboarding/llm-consent", body)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !b.consentRepo.byUser[7] {
		t.Error("expected consent recorded as true")
	}
}

func TestLLMSettings_Unauthorized(t *testing.T) {
	b := &llmHandlerBuilder{}
	h := b.build(t)
	r := newLLMTestRouter(h, 0)

	w := mustServe(t, r, http.MethodGet, "/api/v1/settings/llm", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestLLMSettings_PutWithNilCrypto(t *testing.T) {
	// Dev env without a master key: every mutating call must 503.
	h := NewLLMSettingsHandler(LLMSettingsDeps{
		ConfigRepo:   newFakeLLMConfigRepo(),
		ConsentRepo:  newFakeConsentRepo(),
		Crypto:       nil,
		ProbeTimeout: time.Second,
		Logger:       zap.NewNop(),
	})
	r := newLLMTestRouter(h, 7)

	body := []byte(`{
        "providerType": "claude",
        "apiKey": "sk-ant-abcd1234",
        "model": "claude-sonnet-4-20250514",
        "fallbackToSystemDefaultOnFailure": false,
        "probe": false
    }`)
	w := mustServe(t, r, http.MethodPut, "/api/v1/settings/llm", body)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d body=%s", w.Code, w.Body.String())
	}
}
