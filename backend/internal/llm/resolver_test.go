package llm

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/richman/backend/internal/model"
)

// -------- test doubles --------

// fakeConfigRepo is a narrow stand-in for the LLMConfigRepo interface. It
// records every UpdateHealth call so tests can assert the Resolver writes
// the correct health transition back after a live provider call.
type fakeConfigRepo struct {
	cfg       *model.LLMConfig
	lookupErr error

	mu             sync.Mutex
	healthCalls    []fakeHealthCall
	updateHealthOK error
}

type fakeHealthCall struct {
	configID  int64
	status    model.LLMHealthStatus
	lastError *string
}

func (f *fakeConfigRepo) GetActiveByUserID(
	_ context.Context, _ int64,
) (*model.LLMConfig, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if f.cfg == nil {
		return nil, ErrConfigNotFound
	}
	copyCfg := *f.cfg
	return &copyCfg, nil
}

func (f *fakeConfigRepo) UpdateHealth(
	_ context.Context,
	configID int64,
	status model.LLMHealthStatus,
	lastError *string,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	var errCopy *string
	if lastError != nil {
		s := *lastError
		errCopy = &s
	}
	f.healthCalls = append(f.healthCalls, fakeHealthCall{
		configID: configID, status: status, lastError: errCopy,
	})
	return f.updateHealthOK
}

// fakeConsentRepo backs GetUseSystemDefaultConsent.
type fakeConsentRepo struct {
	consent bool
	err     error
}

func (f *fakeConsentRepo) GetUseSystemDefaultConsent(
	_ context.Context, _ int64,
) (bool, error) {
	return f.consent, f.err
}

// stubProvider is an in-memory Provider for driving the ChatCompletion path.
type stubProvider struct {
	name  string
	resp  *ChatResponse
	err   error
	calls int32
}

func (s *stubProvider) ChatCompletion(
	_ context.Context, _ ChatRequest,
) (*ChatResponse, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func (s *stubProvider) Name() string {
	if s.name == "" {
		return "stub"
	}
	return s.name
}

// -------- helpers --------

// testCrypto returns a Crypto seeded with a deterministic 32-byte master
// key so tests can Encrypt and Decrypt without relying on process env.
func testCrypto(t *testing.T) *Crypto {
	t.Helper()
	keyHex := hex.EncodeToString(make([]byte, MasterKeyBytes))
	c, err := NewCryptoFromHex(keyHex)
	if err != nil {
		t.Fatalf("NewCryptoFromHex: %v", err)
	}
	return c
}

// encryptKey seals a plaintext api key and returns the cipher/nonce pair
// so tests can construct realistic LLMConfig fixtures without duplicating
// the aead wiring in every test.
func encryptKey(t *testing.T, c *Crypto, plaintext string) (cipher, nonce []byte) {
	t.Helper()
	ct, n, err := c.Encrypt([]byte(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return ct, n
}

// makeHealthyCfg builds a baseline user LLMConfig with a Claude provider
// type and a valid ciphertext. Individual tests override fields as needed.
func makeHealthyCfg(t *testing.T, c *Crypto) *model.LLMConfig {
	t.Helper()
	cipher, nonce := encryptKey(t, c, "sk-ant-test-key")
	return &model.LLMConfig{
		ConfigID:                         99,
		UserID:                           42,
		ProviderType:                     model.ProviderClaude,
		APIKeyCipher:                     cipher,
		APIKeyNonce:                      nonce,
		APIKeyHint:                       "..-key",
		Model:                            "claude-sonnet-4-6",
		UseSystemDefaultWhenUnconfigured: false,
		FallbackToSystemDefaultOnFailure: true,
		HealthStatus:                     model.HealthHealthy,
	}
}

// resolverHarness bundles a Resolver with its fakes so tests can peek at
// health updates, stub provider call counts, and the builder closures.
type resolverHarness struct {
	resolver       Resolver
	configRepo     *fakeConfigRepo
	consentRepo    *fakeConsentRepo
	systemDefault  *stubProvider
	userProvider   *stubProvider
	claudeBuilderN int
	openaiBuilderN int
	openaiBaseURL  string
}

// newHarness wires up a Resolver with default fakes. Tests mutate the
// returned fakes before invoking ResolvedChatCompletion.
func newHarness(t *testing.T) *resolverHarness {
	t.Helper()
	h := &resolverHarness{
		configRepo:    &fakeConfigRepo{},
		consentRepo:   &fakeConsentRepo{},
		systemDefault: &stubProvider{name: "system_default"},
		userProvider:  &stubProvider{name: "user"},
	}
	claudeBuilder := func(_ /*apiKey*/, _ /*model*/ string) Provider {
		h.claudeBuilderN++
		return h.userProvider
	}
	openaiBuilder := func(baseURL, _ /*apiKey*/, _ /*model*/ string) Provider {
		h.openaiBuilderN++
		h.openaiBaseURL = baseURL
		return h.userProvider
	}
	h.resolver = NewResolver(
		h.configRepo,
		h.consentRepo,
		testCrypto(t),
		h.systemDefault,
		claudeBuilder,
		openaiBuilder,
		100*time.Millisecond,
		zap.NewNop(),
	)
	return h
}

// -------- state space coverage tests --------

// Row 1: user healthy -> Layer user.
func TestResolver_UserHealthy_PicksUserLayer(t *testing.T) {
	h := newHarness(t)
	h.configRepo.cfg = makeHealthyCfg(t, testCrypto(t))
	h.userProvider.resp = &ChatResponse{Content: "ok"}

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{UserPrompt: "hi"},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.Layer != LayerUser {
		t.Fatalf("expected LayerUser, got %+v", got)
	}
	if got.Response == nil || got.Response.Content != "ok" {
		t.Errorf("expected response from user provider, got %+v", got.Response)
	}
}

// Row 1 secondary: healthy call should stamp HealthHealthy on the config.
func TestResolver_UserHealthy_UpdatesHealthOnSuccess(t *testing.T) {
	h := newHarness(t)
	h.configRepo.cfg = makeHealthyCfg(t, testCrypto(t))
	h.userProvider.resp = &ChatResponse{Content: "ok"}

	_, _ = h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if len(h.configRepo.healthCalls) != 1 {
		t.Fatalf("expected exactly one UpdateHealth call, got %d",
			len(h.configRepo.healthCalls))
	}
	call := h.configRepo.healthCalls[0]
	if call.status != model.HealthHealthy {
		t.Errorf("expected HealthHealthy, got %q", call.status)
	}
	if call.lastError != nil {
		t.Errorf("expected nil lastError on success, got %q", *call.lastError)
	}
	if call.configID != 99 {
		t.Errorf("expected configID=99, got %d", call.configID)
	}
}

// Row 2: user failing, fallback=on, sys available -> LayerSystemDefault.
func TestResolver_UserFailing_FallbackOn_UsesSystemDefault(t *testing.T) {
	h := newHarness(t)
	cfg := makeHealthyCfg(t, testCrypto(t))
	cfg.FallbackToSystemDefaultOnFailure = true
	h.configRepo.cfg = cfg
	h.userProvider.err = errors.New("429 rate limit")
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.Layer != LayerSystemDefault {
		t.Fatalf("expected LayerSystemDefault, got %+v", got)
	}
	if got.Response.Content != "sys" {
		t.Errorf("expected system_default response, got %q", got.Response.Content)
	}
}

// Row 2 secondary: failing call should stamp HealthFailing with error text.
func TestResolver_UserFailing_UpdatesHealthOnFailure(t *testing.T) {
	h := newHarness(t)
	cfg := makeHealthyCfg(t, testCrypto(t))
	cfg.FallbackToSystemDefaultOnFailure = true
	h.configRepo.cfg = cfg
	h.userProvider.err = errors.New("dial tcp: timeout")
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	_, _ = h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if len(h.configRepo.healthCalls) != 1 {
		t.Fatalf("expected one UpdateHealth call, got %d",
			len(h.configRepo.healthCalls))
	}
	call := h.configRepo.healthCalls[0]
	if call.status != model.HealthFailing {
		t.Errorf("expected HealthFailing, got %q", call.status)
	}
	if call.lastError == nil || !strings.Contains(*call.lastError, "dial tcp") {
		t.Errorf("expected error text to contain 'dial tcp', got %v", call.lastError)
	}
}

// Row 3: user failing, fallback=off -> error propagated, no fallback attempt.
func TestResolver_UserFailing_FallbackOff_ReturnsErr(t *testing.T) {
	h := newHarness(t)
	cfg := makeHealthyCfg(t, testCrypto(t))
	cfg.FallbackToSystemDefaultOnFailure = false
	h.configRepo.cfg = cfg
	h.userProvider.err = errors.New("boom")
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil resolved response, got %+v", got)
	}
	if h.systemDefault.calls != 0 {
		t.Errorf("expected 0 system_default calls, got %d", h.systemDefault.calls)
	}
}

// Row 4: user failing, no system_default -> ErrAllLayersFailed (or callErr).
func TestResolver_UserFailing_NoSystemDefault(t *testing.T) {
	h := newHarness(t)
	cfg := makeHealthyCfg(t, testCrypto(t))
	cfg.FallbackToSystemDefaultOnFailure = true
	h.configRepo.cfg = cfg
	h.userProvider.err = errors.New("user down")

	// Rebuild harness with nil system default to exercise the "no fallback
	// available" branch.
	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, testCrypto(t),
		nil,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil resolved response, got %+v", got)
	}
	if !errors.Is(err, ErrAllLayersFailed) {
		t.Errorf("expected ErrAllLayersFailed, got %v", err)
	}
}

// Row 5: user absent, consent=on, sys available -> LayerSystemDefault.
func TestResolver_UserUnconfigured_ConsentOn_UsesSystemDefault(t *testing.T) {
	h := newHarness(t)
	h.configRepo.cfg = nil // GetActiveByUserID returns ErrConfigNotFound
	h.consentRepo.consent = true
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.Layer != LayerSystemDefault {
		t.Fatalf("expected LayerSystemDefault, got %+v", got)
	}
	if h.userProvider.calls != 0 {
		t.Errorf("user provider must not be called, got %d calls", h.userProvider.calls)
	}
}

// Row 6: user absent, consent=off -> ErrConsentDenied.
func TestResolver_UserUnconfigured_ConsentOff_ConsentDenied(t *testing.T) {
	h := newHarness(t)
	h.configRepo.cfg = nil
	h.consentRepo.consent = false

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected ErrConsentDenied, got nil")
	}
	if !errors.Is(err, ErrConsentDenied) {
		t.Errorf("expected ErrConsentDenied, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil resolved response, got %+v", got)
	}
	if h.systemDefault.calls != 0 {
		t.Errorf("system_default must not be called on consent denied")
	}
}

// Row 7: user absent, no system_default, consent irrelevant because we
// short-circuit before probing sys. But still: ensure we return err.
func TestResolver_UserUnconfigured_NoSystemDefault(t *testing.T) {
	h := newHarness(t)
	h.configRepo.cfg = nil
	h.consentRepo.consent = true

	// Rebuild with nil system default.
	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, testCrypto(t),
		nil,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil response, got %+v", got)
	}
	if !errors.Is(err, ErrAllLayersFailed) {
		t.Errorf("expected ErrAllLayersFailed, got %v", err)
	}
}

// -------- provider build-failure tests --------

// SSRF rebuild check: openai_compatible with a blocked base_url should fail
// build and still be allowed to fall back to system_default when the user
// has FallbackToSystemDefaultOnFailure=true. This defends against a user
// saving a previously-valid URL that later resolves to a metadata IP.
func TestResolver_UserFailing_SSRFBlocked_FallsBack(t *testing.T) {
	h := newHarness(t)
	c := testCrypto(t)
	cipher, nonce := encryptKey(t, c, "sk-openai-test")
	blockedURL := "https://metadata.google.internal"
	h.configRepo.cfg = &model.LLMConfig{
		ConfigID:                         77,
		UserID:                           42,
		ProviderType:                     model.ProviderOpenAICompatible,
		BaseURL:                          &blockedURL,
		APIKeyCipher:                     cipher,
		APIKeyNonce:                      nonce,
		APIKeyHint:                       "..test",
		Model:                            "gpt-4",
		FallbackToSystemDefaultOnFailure: true,
	}
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	// Rebuild resolver with the real crypto that matches the cfg's cipher.
	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, c,
		h.systemDefault,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got err: %v", err)
	}
	if got == nil || got.Layer != LayerSystemDefault {
		t.Fatalf("expected LayerSystemDefault after SSRF-block + fallback, got %+v", got)
	}
	if h.userProvider.calls != 0 {
		t.Errorf("user provider must not be reached when build fails, got %d calls",
			h.userProvider.calls)
	}
}

// SSRF block with fallback=off should propagate the build error as-is so
// the settings page can surface a specific SSRF-class message.
func TestResolver_UserBuildFails_FallbackOff_PropagatesErr(t *testing.T) {
	h := newHarness(t)
	c := testCrypto(t)
	cipher, nonce := encryptKey(t, c, "sk-openai-test")
	blockedURL := "https://169.254.169.254"
	h.configRepo.cfg = &model.LLMConfig{
		ConfigID:                         78,
		UserID:                           42,
		ProviderType:                     model.ProviderOpenAICompatible,
		BaseURL:                          &blockedURL,
		APIKeyCipher:                     cipher,
		APIKeyNonce:                      nonce,
		APIKeyHint:                       "..test",
		Model:                            "gpt-4",
		FallbackToSystemDefaultOnFailure: false,
	}
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, c,
		h.systemDefault,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil response, got %+v", got)
	}
	if h.systemDefault.calls != 0 {
		t.Errorf("system_default must not be called when fallback=off")
	}
}

// OpenAI-compatible with a valid https host should pass the SSRF gate and
// propagate the resolved base URL into the openaiBuilder closure. This
// locks the "re-validate every call" behavior so a future refactor cannot
// cache a prior validation result past its DNS TTL.
func TestResolver_OpenAICompatible_ValidatesBaseURL(t *testing.T) {
	h := newHarness(t)
	c := testCrypto(t)
	cipher, nonce := encryptKey(t, c, "sk-openai-test")
	goodURL := "https://api.openai.com"
	h.configRepo.cfg = &model.LLMConfig{
		ConfigID:     55,
		UserID:       42,
		ProviderType: model.ProviderOpenAICompatible,
		BaseURL:      &goodURL,
		APIKeyCipher: cipher,
		APIKeyNonce:  nonce,
		APIKeyHint:   "..test",
		Model:        "gpt-4o",
	}
	h.userProvider.resp = &ChatResponse{Content: "ok"}

	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, c,
		h.systemDefault,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(baseURL, _ /*apiKey*/, _ /*model*/ string) Provider {
			h.openaiBaseURL = baseURL
			return h.userProvider
		},
		100*time.Millisecond, zap.NewNop(),
	)

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.Layer != LayerUser {
		t.Fatalf("expected LayerUser, got %+v", got)
	}
	if h.openaiBaseURL != goodURL {
		t.Errorf("expected openai builder to receive %q, got %q", goodURL, h.openaiBaseURL)
	}
}

// Openai-compatible with nil BaseURL -> ErrConfigDamaged.
func TestResolver_OpenAICompatible_NilBaseURL_Damaged(t *testing.T) {
	h := newHarness(t)
	c := testCrypto(t)
	cipher, nonce := encryptKey(t, c, "sk-test")
	h.configRepo.cfg = &model.LLMConfig{
		ConfigID:                         60,
		UserID:                           42,
		ProviderType:                     model.ProviderOpenAICompatible,
		BaseURL:                          nil,
		APIKeyCipher:                     cipher,
		APIKeyNonce:                      nonce,
		APIKeyHint:                       "..test",
		Model:                            "gpt-4",
		FallbackToSystemDefaultOnFailure: false,
	}

	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, c,
		h.systemDefault,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	_, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigDamaged) {
		t.Errorf("expected ErrConfigDamaged, got %v", err)
	}
}

// Unknown provider_type -> ErrConfigDamaged.
func TestResolver_UnknownProviderType_Damaged(t *testing.T) {
	h := newHarness(t)
	c := testCrypto(t)
	cipher, nonce := encryptKey(t, c, "sk-test")
	h.configRepo.cfg = &model.LLMConfig{
		ConfigID:     61,
		UserID:       42,
		ProviderType: "mystery",
		APIKeyCipher: cipher,
		APIKeyNonce:  nonce,
		APIKeyHint:   "..test",
		Model:        "x",
	}

	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, c,
		h.systemDefault,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	_, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrConfigDamaged) {
		t.Errorf("expected ErrConfigDamaged, got %v", err)
	}
}

// Decrypt failure (garbled cipher) -> ErrDecryptFailed and, with
// fallback=true, the Resolver still attempts the system default. This
// ensures a corrupted-at-rest ciphertext does not permanently black out
// cards if the user opted into fallback.
func TestResolver_DecryptFailure_FallsBack(t *testing.T) {
	h := newHarness(t)
	c := testCrypto(t)
	// Garble the cipher so GCM Open fails.
	cipher, nonce := encryptKey(t, c, "sk-test")
	cipher[0] ^= 0xFF
	h.configRepo.cfg = &model.LLMConfig{
		ConfigID:                         62,
		UserID:                           42,
		ProviderType:                     model.ProviderClaude,
		APIKeyCipher:                     cipher,
		APIKeyNonce:                      nonce,
		APIKeyHint:                       "..test",
		Model:                            "claude-sonnet-4-6",
		FallbackToSystemDefaultOnFailure: true,
	}
	h.systemDefault.resp = &ChatResponse{Content: "sys"}

	h.resolver = NewResolver(
		h.configRepo, h.consentRepo, c,
		h.systemDefault,
		func(_ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		func(_ /*baseURL*/, _ /*apiKey*/, _ /*model*/ string) Provider { return h.userProvider },
		100*time.Millisecond, zap.NewNop(),
	)

	got, err := h.resolver.ResolvedChatCompletion(
		context.Background(), 42, ChatRequest{},
	)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}
	if got == nil || got.Layer != LayerSystemDefault {
		t.Fatalf("expected LayerSystemDefault after decrypt fail + fallback, got %+v", got)
	}
}
