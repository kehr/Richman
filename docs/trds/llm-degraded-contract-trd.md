# LLM 降级契约与用户自选 Provider 技术设计文档

本 TRD 承载 `docs/prds/llm-degraded-contract-prd.md` 的代码级实现细节。PRD 描述"做什么"和"为什么"，TRD 描述"怎么做"，所有字段命名、接口签名、SQL 语句、加密参数、错误码、前端组件 props 都在这里。

## 架构概览

```
+----------------------+
| FE Settings Page     |
|  llm-section.tsx     |
+----------+-----------+
           | HTTP
           v
+----------+-----------+
| API /settings/llm    |
|  settings_llm.go     |
+----------+-----------+
           | Service call
           v
+----------+-----------+      +---------------------+
| llm.Resolver         |<---->| llm_config_repo     |
|  resolver.go         |      |  llm_config_repo.go |
+----------+-----------+      +----------+----------+
           |                             | SQL
           | provider abstraction        v
           v                     +-------+-------+
+----------+-----------+         | llm_configs   |
| llm.Provider         |         +---------------+
|  claude/client.go    |
|  openai/client.go    |
+----------------------+

+----------------------+
| synthesis.Synthesizer |
|  depends on Resolver  |
|  returns *Meta        |
+----------+-----------+
           |
           v
+----------+-----------+      +---------------------+
| service.Analysis     |<---->| decision_card_repo  |
|  writes source+used  |      |  new columns        |
+----------------------+      +---------------------+
```

## 数据库设计

### Migration 010: `llm_configs` 表

文件：`backend/db/migration/010_llm_configs.up.sql`

```sql
CREATE TABLE llm_configs (
    config_id                              BIGSERIAL PRIMARY KEY,
    user_id                                BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    provider_type                          VARCHAR(32)  NOT NULL
        CHECK (provider_type IN ('claude', 'openai', 'openai_compatible')),
    base_url                               VARCHAR(512),
    api_key_cipher                         BYTEA        NOT NULL,
    api_key_nonce                          BYTEA        NOT NULL,
    api_key_hint                           VARCHAR(16)  NOT NULL,
    model                                  VARCHAR(128) NOT NULL,
    use_system_default_when_unconfigured   BOOLEAN      NOT NULL DEFAULT FALSE,
    fallback_to_system_default_on_failure  BOOLEAN      NOT NULL DEFAULT FALSE,
    health_status                          VARCHAR(16)  NOT NULL DEFAULT 'unknown'
        CHECK (health_status IN ('healthy', 'failing', 'unknown')),
    last_probe_at                          TIMESTAMPTZ,
    last_probe_error                       TEXT,
    created_at                             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at                             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    is_deleted                             BOOLEAN      NOT NULL DEFAULT FALSE
);

CREATE UNIQUE INDEX idx_llm_configs_active_user
    ON llm_configs (user_id)
    WHERE is_deleted = FALSE;

CREATE INDEX idx_llm_configs_health_status
    ON llm_configs (health_status)
    WHERE is_deleted = FALSE;

COMMENT ON TABLE llm_configs IS
    'Per-user LLM provider configuration. Exactly one active config per user enforced by partial unique index.';
COMMENT ON COLUMN llm_configs.api_key_cipher IS
    'AES-256-GCM ciphertext of the plaintext api key. Master key from env LLM_CONFIG_MASTER_KEY.';
COMMENT ON COLUMN llm_configs.api_key_nonce IS
    'GCM nonce, 12 bytes, randomly generated on every save. Must be stored together with cipher.';
COMMENT ON COLUMN llm_configs.api_key_hint IS
    'Last 4 characters of the plaintext api key, prefixed with ".." e.g. "..abcd". Safe to log and display.';
```

文件：`backend/db/migration/010_llm_configs.down.sql`

```sql
DROP INDEX IF EXISTS idx_llm_configs_health_status;
DROP INDEX IF EXISTS idx_llm_configs_active_user;
DROP TABLE IF EXISTS llm_configs;
```

### Migration 011: `decision_cards` 新列

文件：`backend/db/migration/011_decision_cards_synthesis_source.up.sql`

```sql
ALTER TABLE decision_cards
    ADD COLUMN synthesis_source VARCHAR(16)
        CHECK (synthesis_source IN ('llm', 'template', 'mixed')),
    ADD COLUMN provider_used    VARCHAR(32)
        CHECK (provider_used IN ('user', 'system_default', 'none'));

-- Optimistic backfill: historical deployments were assumed to be LLM-driven.
-- New columns are nullable so this UPDATE is idempotent if re-run.
UPDATE decision_cards
SET synthesis_source = 'llm',
    provider_used    = 'user'
WHERE synthesis_source IS NULL;

CREATE INDEX idx_decision_cards_synthesis_source
    ON decision_cards (synthesis_source)
    WHERE is_deleted = FALSE;

COMMENT ON COLUMN decision_cards.synthesis_source IS
    'Source of the synthesized content: llm (full AI), template (fallback), mixed (LLM text + template recommendation).';
COMMENT ON COLUMN decision_cards.provider_used IS
    'Which provider layer served this analysis: user, system_default, or none.';
```

文件：`backend/db/migration/011_decision_cards_synthesis_source.down.sql`

```sql
DROP INDEX IF EXISTS idx_decision_cards_synthesis_source;
ALTER TABLE decision_cards
    DROP COLUMN IF EXISTS synthesis_source,
    DROP COLUMN IF EXISTS provider_used;
```

## Go 数据结构

### `internal/model/llm_config.go` (新)

```go
package model

import "time"

type LLMProviderType string

const (
    ProviderClaude           LLMProviderType = "claude"
    ProviderOpenAI           LLMProviderType = "openai"
    ProviderOpenAICompatible LLMProviderType = "openai_compatible"
)

type LLMHealthStatus string

const (
    HealthHealthy LLMHealthStatus = "healthy"
    HealthFailing LLMHealthStatus = "failing"
    HealthUnknown LLMHealthStatus = "unknown"
)

// LLMConfig is the persistence model for a user's LLM provider configuration.
// api_key_cipher and api_key_nonce are NEVER serialized to JSON.
type LLMConfig struct {
    ConfigID                          int64           `db:"config_id"`
    UserID                            int64           `db:"user_id"`
    ProviderType                      LLMProviderType `db:"provider_type"`
    BaseURL                           *string         `db:"base_url"`
    APIKeyCipher                      []byte          `db:"api_key_cipher"`
    APIKeyNonce                       []byte          `db:"api_key_nonce"`
    APIKeyHint                        string          `db:"api_key_hint"`
    Model                             string          `db:"model"`
    UseSystemDefaultWhenUnconfigured  bool            `db:"use_system_default_when_unconfigured"`
    FallbackToSystemDefaultOnFailure  bool            `db:"fallback_to_system_default_on_failure"`
    HealthStatus                      LLMHealthStatus `db:"health_status"`
    LastProbeAt                       *time.Time      `db:"last_probe_at"`
    LastProbeError                    *string         `db:"last_probe_error"`
    CreatedAt                         time.Time       `db:"created_at"`
    UpdatedAt                         time.Time       `db:"updated_at"`
    IsDeleted                         bool            `db:"is_deleted"`
}

// Masked returns a log-safe string representation that NEVER includes the
// plaintext api key or the ciphertext bytes. This is the only method callers
// may use to serialize LLMConfig for structured logs, errors, and metrics.
func (c *LLMConfig) Masked() string {
    baseURL := ""
    if c.BaseURL != nil {
        baseURL = *c.BaseURL
    }
    return fmt.Sprintf(
        "LLMConfig{user=%d type=%s model=%s base_url=%s key_hint=%s health=%s}",
        c.UserID, c.ProviderType, c.Model, baseURL, c.APIKeyHint, c.HealthStatus,
    )
}
```

`String()` 方法不实现，避免 `%v` 和 `%+v` 意外打印完整结构体。

### `internal/model/decision_card.go` 扩展

```go
type DecisionCard struct {
    // ... existing fields ...
    SynthesisSource *string // nullable; "llm" | "template" | "mixed"
    ProviderUsed    *string // nullable; "user" | "system_default" | "none"
}
```

## 加密模块

### `internal/llm/crypto.go` (新)

```go
package llm

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/hex"
    "errors"
    "io"
)

const (
    MasterKeyBytes = 32 // AES-256
    NonceBytes     = 12 // GCM standard
)

var (
    ErrMasterKeyInvalid = errors.New("llm: master key must be 32 bytes (64 hex chars)")
    ErrDecryptFailed    = errors.New("llm: decrypt failed")
)

// Crypto wraps a GCM cipher loaded once at startup.
type Crypto struct {
    aead cipher.AEAD
}

func NewCryptoFromHex(masterKeyHex string) (*Crypto, error) {
    if len(masterKeyHex) != MasterKeyBytes*2 {
        return nil, ErrMasterKeyInvalid
    }
    key, err := hex.DecodeString(masterKeyHex)
    if err != nil {
        return nil, ErrMasterKeyInvalid
    }
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    return &Crypto{aead: aead}, nil
}

// Encrypt returns (ciphertext, nonce, error).
func (c *Crypto) Encrypt(plaintext []byte) ([]byte, []byte, error) {
    nonce := make([]byte, NonceBytes)
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, nil, err
    }
    ct := c.aead.Seal(nil, nonce, plaintext, nil)
    return ct, nonce, nil
}

func (c *Crypto) Decrypt(ciphertext, nonce []byte) ([]byte, error) {
    if len(nonce) != NonceBytes {
        return nil, ErrDecryptFailed
    }
    pt, err := c.aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, ErrDecryptFailed
    }
    return pt, nil
}
```

主密钥从 `cfg.LLM.ConfigMasterKey` 加载；`config.Load()` 里新增该字段，来源 env `LLM_CONFIG_MASTER_KEY`。启动时 `NewCryptoFromHex` 校验失败直接 `log.Fatal`。

## SSRF 防护

### `internal/llm/ssrf.go` (新)

```go
package llm

import (
    "errors"
    "net"
    "net/url"
    "strings"
)

var (
    ErrSSRFBadScheme    = errors.New("llm: base_url must use https scheme")
    ErrSSRFHostBlocked  = errors.New("llm: base_url hostname blocked")
    ErrSSRFPrivateIP    = errors.New("llm: base_url resolves to private IP range")
    ErrSSRFMetadataHost = errors.New("llm: base_url is a cloud metadata endpoint")
)

var blockedHosts = map[string]bool{
    "localhost":                   true,
    "metadata.google.internal":    true,
    "169.254.169.254":             true,
    "metadata":                    true,
}

var privateCIDRs []*net.IPNet

func init() {
    for _, cidr := range []string{
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
        "127.0.0.0/8",
        "169.254.0.0/16",
        "fc00::/7",
        "::1/128",
        "fe80::/10",
    } {
        _, block, _ := net.ParseCIDR(cidr)
        privateCIDRs = append(privateCIDRs, block)
    }
}

// ValidateBaseURL enforces the SSRF policy documented in the PRD.
// Must be called both on save and on every actual provider call.
func ValidateBaseURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return ErrSSRFBadScheme
    }
    if u.Scheme != "https" {
        return ErrSSRFBadScheme
    }
    host := strings.ToLower(u.Hostname())
    if host == "" {
        return ErrSSRFHostBlocked
    }
    if strings.HasSuffix(host, ".local") {
        return ErrSSRFHostBlocked
    }
    if blockedHosts[host] {
        return ErrSSRFMetadataHost
    }
    // DNS resolve and check every A/AAAA record
    ips, err := net.LookupIP(host)
    if err != nil || len(ips) == 0 {
        return ErrSSRFHostBlocked
    }
    for _, ip := range ips {
        for _, block := range privateCIDRs {
            if block.Contains(ip) {
                return ErrSSRFPrivateIP
            }
        }
    }
    return nil
}
```

调用点：
1. `settings_llm.go` 的 PUT handler 在 save 前调用一次
2. `llm.Resolver` 每次构造用户 provider 前再调用一次（防 DNS rebinding）
3. `POST /settings/llm/probe` 在 probe 前调用一次

## LLM Provider 接口

### 现有 `internal/llm/provider.go` 保留

```go
type Provider interface {
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    Name() string
}
```

### 新增 `internal/llm/resolver.go`

```go
package llm

import (
    "context"
    "errors"
    "time"

    "go.uber.org/zap"
)

// ProviderLayer identifies which layer of the fallback chain served a request.
type ProviderLayer string

const (
    LayerUser          ProviderLayer = "user"
    LayerSystemDefault ProviderLayer = "system_default"
    LayerNone          ProviderLayer = "none"
)

// ResolvedResponse wraps the underlying ChatResponse with the layer that
// actually served the request.
type ResolvedResponse struct {
    Response *ChatResponse
    Layer    ProviderLayer
}

// Resolver encapsulates the three-level fallback chain:
//   user -> system_default -> none (caller falls back to template)
type Resolver interface {
    // ResolvedChatCompletion tries the fallback chain for a specific user.
    // Returns (*ResolvedResponse, nil) if any layer succeeds. Returns
    // (nil, err) only when ALL layers fail or when consent denies fallback.
    // The caller interprets a nil response as "use template fallback".
    ResolvedChatCompletion(ctx context.Context, userID int64, req ChatRequest) (*ResolvedResponse, error)
}

// resolverImpl is the production Resolver implementation.
type resolverImpl struct {
    configRepo     LLMConfigRepo
    crypto         *Crypto
    systemDefault  Provider // may be nil
    claudeBuilder  func(apiKey, model string) Provider
    openaiBuilder  func(baseURL, apiKey, model string) Provider
    probeTimeout   time.Duration
    logger         *zap.Logger
}

// LLMConfigRepo is the narrow interface Resolver depends on. Kept separate
// from the full repo to minimize coupling and simplify tests.
type LLMConfigRepo interface {
    GetActiveByUserID(ctx context.Context, userID int64) (*model.LLMConfig, error)
    UpdateHealth(ctx context.Context, configID int64, status model.LLMHealthStatus, lastError *string) error
}

func NewResolver(
    configRepo LLMConfigRepo,
    crypto *Crypto,
    systemDefault Provider,
    claudeBuilder func(apiKey, model string) Provider,
    openaiBuilder func(baseURL, apiKey, model string) Provider,
    probeTimeout time.Duration,
    logger *zap.Logger,
) Resolver {
    return &resolverImpl{
        configRepo:    configRepo,
        crypto:        crypto,
        systemDefault: systemDefault,
        claudeBuilder: claudeBuilder,
        openaiBuilder: openaiBuilder,
        probeTimeout:  probeTimeout,
        logger:        logger,
    }
}

func (r *resolverImpl) ResolvedChatCompletion(
    ctx context.Context,
    userID int64,
    req ChatRequest,
) (*ResolvedResponse, error) {
    cfg, err := r.configRepo.GetActiveByUserID(ctx, userID)
    if err != nil && !errors.Is(err, ErrConfigNotFound) {
        r.logger.Warn("llm config lookup failed",
            zap.Int64("user_id", userID),
            zap.Error(err),
        )
    }

    // Layer 1: user
    if cfg != nil {
        userProvider, buildErr := r.buildUserProvider(cfg)
        if buildErr == nil {
            resp, callErr := userProvider.ChatCompletion(ctx, req)
            if callErr == nil {
                _ = r.configRepo.UpdateHealth(ctx, cfg.ConfigID, model.HealthHealthy, nil)
                return &ResolvedResponse{Response: resp, Layer: LayerUser}, nil
            }
            errStr := callErr.Error()
            _ = r.configRepo.UpdateHealth(ctx, cfg.ConfigID, model.HealthFailing, &errStr)
            r.logger.Warn("user provider failed",
                zap.Int64("user_id", userID),
                zap.String("config", cfg.Masked()),
                zap.Error(callErr),
            )
            // Decide whether to fall back to system default
            if !cfg.FallbackToSystemDefaultOnFailure {
                return nil, callErr
            }
        } else {
            r.logger.Warn("user provider build failed",
                zap.Int64("user_id", userID),
                zap.Error(buildErr),
            )
            if !cfg.FallbackToSystemDefaultOnFailure {
                return nil, buildErr
            }
        }
    } else {
        // No user config — the caller's consent field is not on a user config
        // row (because there is none). Require consent via a default-side
        // lookup. For MVP we store this consent in the user's onboarding
        // profile, not in llm_configs. See UserRepo.GetUseSystemDefaultConsent.
        consent, consentErr := r.getUseSystemDefaultConsent(ctx, userID)
        if consentErr != nil || !consent {
            return nil, ErrConsentDenied
        }
    }

    // Layer 2: system_default
    if r.systemDefault != nil {
        resp, callErr := r.systemDefault.ChatCompletion(ctx, req)
        if callErr == nil {
            return &ResolvedResponse{Response: resp, Layer: LayerSystemDefault}, nil
        }
        r.logger.Warn("system_default provider failed", zap.Error(callErr))
    }

    // Layer 3: none -> caller will use template fallback
    return nil, ErrAllLayersFailed
}

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
        if err := ValidateBaseURL(*cfg.BaseURL); err != nil {
            return nil, err
        }
        return r.openaiBuilder(*cfg.BaseURL, string(plaintext), cfg.Model), nil
    default:
        return nil, ErrConfigDamaged
    }
}

func zeroBytes(b []byte) {
    for i := range b {
        b[i] = 0
    }
}
```

**`getUseSystemDefaultConsent` 的存储位置决策**：本 MVP 把"未配置时是否用系统默认"的 consent 放在 `users` 表新增一列 `use_system_default_llm_consent BOOLEAN`，而不是 `llm_configs` 里。因为 `llm_configs` 的语义是"用户配置了自己的 provider"，如果用户没配那一行不存在。onboarding 里勾选后直接写到 `users` 表，简单直接。

### Migration 012: `users.use_system_default_llm_consent`

```sql
ALTER TABLE users
    ADD COLUMN use_system_default_llm_consent BOOLEAN NOT NULL DEFAULT FALSE;
```

## Synthesizer 接口演进

### `internal/analysis/synthesis/synthesizer.go` 改造

```go
// SynthesisMeta carries provenance information about how the output was
// produced. Both fields are persisted on decision_cards.
type SynthesisMeta struct {
    Source       string // "llm" | "template" | "mixed"
    ProviderUsed string // "user" | "system_default" | "none"
    LatencyMs    int64  // wall-clock time of the resolver call, 0 for template
}

// Synthesizer generates structured decision card content using the LLM
// resolver. When the resolver is nil or returns an error, the output falls
// back to a deterministic template and SynthesisMeta reflects the actual
// layer that served the request.
type Synthesizer struct {
    resolver llm.Resolver
    logger   *zap.Logger
}

func NewSynthesizer(resolver llm.Resolver, logger *zap.Logger) *Synthesizer {
    return &Synthesizer{resolver: resolver, logger: logger}
}

// Synthesize generates the decision card content for one holding.
// Returns (output, meta, nil) on success or degraded fallback. Returns
// (nil, nil, err) only when both the LLM path and the template path fail,
// which should be impossible in practice.
func (s *Synthesizer) Synthesize(
    ctx context.Context,
    input *SynthesisInput,
    userID int64,
) (*SynthesisOutput, *SynthesisMeta, error) {
    if s.resolver == nil {
        return templateFallback(input), templateMeta(), nil
    }

    start := time.Now()
    resolved, err := s.resolver.ResolvedChatCompletion(
        ctx, userID, llm.ChatRequest{
            SystemPrompt: synthesisSystemPrompt,
            UserPrompt:   buildSynthesisPrompt(input),
            MaxTokens:    2048,
            Temperature:  0.4,
        },
    )
    if err != nil || resolved == nil {
        s.logger.Info("synthesize falling back to template",
            zap.String("asset", input.AssetCode),
            zap.Error(err),
        )
        return templateFallback(input), templateMeta(), nil
    }

    output, parseErr := parseSynthesisResponse(resolved.Response.Content)
    if parseErr != nil {
        s.logger.Warn("synthesis response unparseable",
            zap.String("asset", input.AssetCode),
            zap.Error(parseErr),
        )
        return templateFallback(input), &SynthesisMeta{
            Source:       "template",
            ProviderUsed: string(resolved.Layer),
            LatencyMs:    time.Since(start).Milliseconds(),
        }, nil
    }

    // Recommendation sub-object: llm or mixed?
    source := "llm"
    if parsed := parseRecommendation(extractJSON(resolved.Response.Content)); parsed != nil {
        ensureRecommendation(parsed, input)
        output.Recommendation = *parsed
    } else {
        source = "mixed"
        output.Recommendation = fallbackRecommendation(input)
    }

    return output, &SynthesisMeta{
        Source:       source,
        ProviderUsed: string(resolved.Layer),
        LatencyMs:    time.Since(start).Milliseconds(),
    }, nil
}

func templateMeta() *SynthesisMeta {
    return &SynthesisMeta{Source: "template", ProviderUsed: "none"}
}
```

现有的 `templateFallback`、`fallbackRecommendation`、`parseRecommendation`、`parseSynthesisResponse`、`extractJSON` 全部保留不变。

### 测试更新

`synthesizer_test.go` 里的 stubProvider 被替换为 stubResolver：

```go
type stubResolver struct {
    resp  *llm.ResolvedResponse
    err   error
}

func (s *stubResolver) ResolvedChatCompletion(
    _ context.Context, _ int64, _ llm.ChatRequest,
) (*llm.ResolvedResponse, error) {
    return s.resp, s.err
}
```

已有的 5 个测试（LLMSuccess、LLMFailure、MalformedJSON、MissingRecommendation、NilProvider）全部改为传 resolver 实例，断言时同步检查返回的 meta 字段。新增一个测试 `TestSynthesize_SystemDefaultFallback_RecordsLayer`：stubResolver 返回 `Layer=system_default`，断言输出的 meta.ProviderUsed == "system_default"。

## Service 层改造

### `internal/service/analysis/service.go` 改动

1. `Deps` 字段 `Synthesizer *synthesis.Synthesizer` 的构造入口改为注入 `llm.Resolver`（通过 main.go）
2. `AnalyzeHolding` 内部调用 `synthesizer.Synthesize(ctx, input, userID)` 并接收 meta
3. `model.DecisionCard` 赋值时把 meta 的两字段写进 `SynthesisSource` 和 `ProviderUsed`
4. `persistDecisionCardWithDiff` 函数签名不变，但 card 里多两个字段

### `TriggerReanalyzeAll` 新方法

```go
// TriggerReanalyzeAll re-runs analysis for every active holding of the user.
// Implemented on top of the existing TriggerAnalysis by reusing its
// goroutine and task store. Returns the task_id.
func (s *Service) TriggerReanalyzeAll(ctx context.Context, userID int64) (string, error) {
    taskID := uuid.New().String()
    s.TriggerAnalysis(ctx, userID, taskID)
    return taskID, nil
}
```

（如果 `TriggerAnalysis` 已经覆盖了"全部持仓"的语义，则 `TriggerReanalyzeAll` 可以就是它的 alias。实际实现时看 `TriggerAnalysis` 的行为决定。）

## Repo 层

### `internal/repo/llm_config_repo.go` (新)

关键方法签名：

```go
type LLMConfigRepo struct {
    pool *pgxpool.Pool
}

func NewLLMConfigRepo(pool *pgxpool.Pool) *LLMConfigRepo

// GetActiveByUserID returns the single active config for a user, or
// ErrConfigNotFound if the user has no active config.
func (r *LLMConfigRepo) GetActiveByUserID(ctx context.Context, userID int64) (*model.LLMConfig, error)

// Upsert creates or updates the single active config for a user.
// Enforces the one-active-per-user invariant at the DB level via the
// partial unique index.
func (r *LLMConfigRepo) Upsert(ctx context.Context, cfg *model.LLMConfig) error

// SoftDelete marks the user's active config as deleted.
func (r *LLMConfigRepo) SoftDelete(ctx context.Context, userID int64) error

// UpdateHealth updates only the health-related fields, leaving key
// cipher/nonce untouched.
func (r *LLMConfigRepo) UpdateHealth(
    ctx context.Context, configID int64,
    status model.LLMHealthStatus, lastError *string,
) error
```

SQL 模板用 sqlc 或手写 pgx。为了和现有 repo 风格一致，MVP 阶段手写 pgx。

### `internal/repo/decision_card_repo.go` 扩展

- `CreateDecisionCard` / `CreateDecisionCardTx` 的 INSERT 列表追加 `synthesis_source, provider_used`
- `GetByID` / `GetLatestByHolding` 的 SELECT 列表追加两列
- `ListBySources` 新方法：按 `synthesis_source ∈ (...)` 过滤，供 dashboard summary 判断 needsReanalysis

### `internal/repo/user_repo.go` 扩展

- `GetUseSystemDefaultConsent(ctx, userID int64) (bool, error)`：读 users.use_system_default_llm_consent
- `SetUseSystemDefaultConsent(ctx, userID int64, consent bool) error`：写该字段，伴随 updated_at 刷新

## API 层

### `internal/api/v1/settings_llm.go` (新)

```go
// GET /api/v1/settings/llm
// Response:
// {
//   "configured": true,
//   "providerType": "claude",
//   "baseUrl": null,
//   "model": "claude-sonnet-4-6",
//   "apiKeyHint": "..abcd",
//   "useSystemDefaultWhenUnconfigured": false,
//   "fallbackToSystemDefaultOnFailure": true,
//   "healthStatus": "healthy",
//   "lastProbeAt": "2026-04-09T12:34:56Z",
//   "lastProbeError": null
// }
//
// If unconfigured: {"configured": false, "useSystemDefaultWhenUnconfigured": <from users table>}

// PUT /api/v1/settings/llm
// Request:
// {
//   "providerType": "claude",
//   "baseUrl": null,
//   "apiKey": "sk-ant-...",
//   "model": "claude-sonnet-4-6",
//   "fallbackToSystemDefaultOnFailure": true,
//   "probe": true
// }
// Response: same as GET
// Behavior:
//   1. validate base_url if provider_type = openai_compatible (SSRF check)
//   2. if probe=true, construct a temporary provider and do a minimal probe
//   3. encrypt api_key with crypto
//   4. repo.Upsert in a transaction
//   5. return 200 with masked GET response

// DELETE /api/v1/settings/llm
// Response: 204

// POST /api/v1/settings/llm/probe
// Request: (none, uses the stored config)
// Response:
// {
//   "healthy": true,
//   "error": null,
//   "latencyMs": 523
// }
```

Handler 代码里只使用 `LLMConfig.Masked()` 和 `APIKeyHint` 给前端，**永远不返回 api_key 明文**。

### `internal/api/v1/onboarding.go` 扩展

`POST /api/v1/onboarding/llm-consent` 新 handler，写 users.use_system_default_llm_consent。

### `internal/api/v1/analysis.go` 扩展

`POST /api/v1/analysis/reanalyze-all` 新 handler：

```go
func (h *Handler) ReanalyzeAll(c *gin.Context) {
    userID := middleware.GetUserID(c)
    taskID, err := h.svc.TriggerReanalyzeAll(c.Request.Context(), userID)
    if err != nil {
        respondError(c, http.StatusInternalServerError, err)
        return
    }
    c.JSON(http.StatusOK, gin.H{"taskId": taskID})
}
```

Rate limiting 由 gin middleware 处理：`rateLimit(1, 10*time.Minute)` 以 userID 为 key。

### `internal/api/v1/decision_cards.go` 扩展

`GET /api/v1/decision-cards/:id` 和列表响应追加 `synthesisSource` 和 `providerUsed` 字段。现有的 DTO 类型扩展两个字段，repo → dto 映射处理 nullable → "unknown" 转换。

### `internal/api/v1/dashboard.go` 扩展

dashboard summary 响应追加 `llmStatus` 子对象，字段语义见 PRD "UX 表面/Dashboard Banner" 段落。后端计算 `needsReanalysis` 的 SQL：

```sql
SELECT EXISTS (
  SELECT 1 FROM decision_cards
  WHERE user_id = $1 AND is_deleted = FALSE
    AND synthesis_source IN ('template', 'mixed')
) AS needs_reanalysis;
```

## 前端架构

### 新增 Feature: `features/settings/llm/`

```
features/settings/llm/
  LLMSection.tsx              // top-level
  LLMEmptyState.tsx           // unconfigured
  LLMHealthyCard.tsx          // configured + healthy
  LLMFailingCard.tsx          // configured + failing
  LLMConfigForm.tsx           // add/edit form
  LLMProbeButton.tsx
  hooks.ts                    // TanStack Query hooks
  types.ts                    // DTO types
  index.ts                    // barrel export
```

关键 hooks：
- `useLLMConfig()` → `{ data, isLoading, refetch }`
- `useUpsertLLMConfig()` → `mutation`，onSuccess invalidate `llm-config` + `dashboard-summary`
- `useDeleteLLMConfig()` → `mutation`
- `useProbeLLMConfig()` → `mutation`
- `useReanalyzeAll()` → `mutation`，onSuccess invalidate `decision-cards` + `dashboard-summary`

表单校验：
- providerType 必填，三选一
- base_url 仅当 providerType = openai_compatible 时必填，必须 https
- api_key 必填，创建时必填，编辑时可空（空表示不变）
- model 必填
- consent 勾选：字段 `fallbackToSystemDefaultOnFailure` 默认 false，明示隐私 tradeoff

### 新增 Feature: `features/dashboard/llm-status/`

```
features/dashboard/llm-status/
  LLMStatusBanner.tsx         // top banner
  useLLMStatusBanner.ts       // reads dashboard-summary
  index.ts
```

banner 展示条件：`dashboardSummary.llmStatus.needsReanalysis === true`。关闭按钮调用 sessionStorage，key `llm-status-banner-dismissed`。"重新分析所有持仓"按钮触发 `useReanalyzeAll` mutation。

### 扩展 Feature: `features/decision-card/`

`SourcePill.tsx` 新组件：根据 `card.synthesisSource` 渲染不同颜色的 pill 和 tooltip。集成到 `DecisionCardHeader.tsx`。

### 扩展 API 类型：`api/decisionCards.ts`

DecisionCardDTO 类型追加：

```ts
interface DecisionCardDTO {
  // ... existing ...
  synthesisSource: 'llm' | 'template' | 'mixed' | 'unknown';
  providerUsed: 'user' | 'system_default' | 'none' | 'unknown';
}
```

后端返回 null 时前端映射为 `'unknown'`，SourcePill 对 unknown 不渲染。

## 配置扩展

### `internal/config/config.go`

```go
type LLMConfig struct {
    DefaultProviderType string // claude | openai | openai_compatible | empty
    DefaultBaseURL      string
    DefaultAPIKey       string
    DefaultModel        string
    ConfigMasterKey     string // 64 hex chars
    ProbeTimeout        time.Duration
}
```

env 变量：
```
LLM_DEFAULT_PROVIDER_TYPE=claude
LLM_DEFAULT_API_KEY=sk-ant-xxx
LLM_DEFAULT_MODEL=claude-sonnet-4-6
LLM_CONFIG_MASTER_KEY=<64 hex chars>
LLM_PROBE_TIMEOUT=5s
```

`.env.example` 补上述行。

## Wire-up (main.go)

`cmd/server/main.go` 的 LLM 初始化段改写为：

```go
// Initialize llm crypto
crypto, err := llm.NewCryptoFromHex(cfg.LLM.ConfigMasterKey)
if err != nil {
    zapLogger.Fatal("llm crypto init failed", zap.Error(err))
}

// System default provider (may be nil)
var systemDefault llm.Provider
systemDefault, err = llm.NewProvider(cfg, zapLogger)
if err != nil {
    zapLogger.Warn("system default llm unavailable", zap.Error(err))
}

// Provider builders closures over shared http client
httpClient := &http.Client{Timeout: 30 * time.Second}
claudeBuilder := func(apiKey, model string) llm.Provider {
    return claude.NewClient(apiKey, model, httpClient, zapLogger)
}
openaiBuilder := func(baseURL, apiKey, model string) llm.Provider {
    return openai.NewClient(baseURL, apiKey, model, httpClient, zapLogger)
}

llmConfigRepo := repo.NewLLMConfigRepo(dbPool)
resolver := llm.NewResolver(
    llmConfigRepo, crypto, systemDefault,
    claudeBuilder, openaiBuilder,
    cfg.LLM.ProbeTimeout, zapLogger,
)

llmSynthesizer := synthesis.NewSynthesizer(resolver, zapLogger)
```

## 非功能性要求

### 性能

- Resolver 解析一次配置平均 ~2ms（DB query 1ms + decrypt 0.5ms + build provider 0.5ms）
- 一次 AnalyzeHolding 端到端增加的开销：~5ms（resolver + metrics emit）
- 不对主流程引入新的 I/O 层（不缓存、不轮询）

### 并发

- Resolver 可安全被多 goroutine 共享
- `resolverImpl` 的字段全部只读或线程安全
- `buildUserProvider` 里构造的 `Provider` 实例是单次使用，生命周期 = 本次调用

### 监控

Prometheus metrics 定义在 `internal/metrics/llm.go`，由 resolver 和 synthesizer 分别 emit。

### 测试

单元测试覆盖：
- Crypto encrypt/decrypt roundtrip + corrupted nonce + wrong master key
- ValidateBaseURL 每种 block rule
- Resolver 所有 12 种状态空间组合
- Synthesizer 6 种分支（包括新的 SystemDefaultFallback）
- LLMConfigRepo 的 Upsert / SoftDelete / UpdateHealth
- SSRF check 每个黑名单 case

集成测试：
- Handler 级别的 PUT /settings/llm：合法、SSRF 违规、probe 失败、无 auth
- `/api/v1/analysis/reanalyze-all` rate limiting
- Dashboard summary 的 needsReanalysis 计算

## 实现顺序

Plan 文档会拆成若干 step，顺序保证 backend 先 DB → Provider → Resolver → Synthesizer → Service → API，前端在 backend API 稳定后开始。详见 `docs/plans/llm-degraded-contract-plan/`。
