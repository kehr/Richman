# Step 1: Backend Foundation

## 任务目标

为 LLM 降级契约铺设 backend 的数据和底层工具依赖：DB schema、Go model、加密模块、SSRF 校验。这一步完成后，后续 Repo 和 Resolver 才有地基。

## 涉及文件

### 新增

- `backend/db/migration/010_llm_configs.up.sql`
- `backend/db/migration/010_llm_configs.down.sql`
- `backend/db/migration/011_decision_cards_synthesis_source.up.sql`
- `backend/db/migration/011_decision_cards_synthesis_source.down.sql`
- `backend/db/migration/012_users_llm_consent.up.sql`
- `backend/db/migration/012_users_llm_consent.down.sql`
- `backend/internal/model/llm_config.go`
- `backend/internal/llm/crypto.go`
- `backend/internal/llm/crypto_test.go`
- `backend/internal/llm/ssrf.go`
- `backend/internal/llm/ssrf_test.go`

### 修改

- `backend/internal/model/decision_card.go`：追加 `SynthesisSource *string` 和 `ProviderUsed *string`
- `backend/internal/config/config.go`：新增 `LLMConfig.ConfigMasterKey`、`ProbeTimeout`
- `backend/.env.example`：新增 `LLM_CONFIG_MASTER_KEY`、`LLM_PROBE_TIMEOUT`

## 设计依据

- PRD "数据模型" 段落：`llm_configs` 表结构、`decision_cards` 扩展、historical 回填
- PRD "安全/密钥存储"：AES-256-GCM、master key 从 env 加载
- PRD "安全/SSRF 防护"：https-only、CIDR block list、hostname 黑名单
- TRD "数据库设计/Migration 010-012"：完整 SQL
- TRD "Go 数据结构/llm_config.go"：struct 字段、`Masked()` 方法
- TRD "加密模块"：Crypto 类型定义
- TRD "SSRF 防护/ssrf.go"：ValidateBaseURL 实现

## 验证标准

- 3 个 migration 文件 up/down 成对存在，`make migrate-up` + `make migrate-down` + `make migrate-up` 可以 roundtrip 无错
- 010 migration 包含 partial unique index，重复 insert 同 user 的第二条 active 配置会被 DB 拒绝
- 011 migration 执行后 decision_cards 表有 `synthesis_source` 和 `provider_used` 两列，已存在的行被回填为 `'llm' / 'user'`
- 012 migration 添加 `use_system_default_llm_consent BOOLEAN NOT NULL DEFAULT FALSE`
- `model.LLMConfig.Masked()` 方法 roundtrip 测试：给定 config，masked 字符串里不包含任何 api_key_cipher 或 api_key_nonce 字节
- `llm.Crypto` 单元测试：
  - encrypt/decrypt roundtrip 返回原文
  - decrypt 带错误 nonce 返回 `ErrDecryptFailed`
  - NewCryptoFromHex 对非 64-char hex 返回 `ErrMasterKeyInvalid`
- `llm.ValidateBaseURL` 单元测试覆盖每种 block case：
  - `http://example.com` → ErrSSRFBadScheme
  - `https://localhost/api` → ErrSSRFHostBlocked
  - `https://169.254.169.254/` → ErrSSRFMetadataHost
  - `https://10.0.0.1/api` → ErrSSRFPrivateIP
  - `https://api.anthropic.com/` → nil
- `go vet ./...` 无 warning
- `golangci-lint run ./...` 0 issues
- `go test ./internal/llm/...` 全绿

## 依赖

无（是整个链路的起点）。但需要 postgres 实例才能跑 migration；如果本地 docker-compose 未启动，step 开始前先 `docker-compose up -d`。

## 偏差处理

- 如果 `config.Load()` 已经有 LLM 相关段落，扩展现有段，不要创建平行段落
- 如果 `decision_card.go` 里 `DecisionCard` 结构体字段已按 sqlc 生成，新增字段要同步 sqlc 命名约定（check `backend/sqlc.yaml` 配置）
- SSRF 校验里的 DNS resolve 在测试环境需要 mock 掉 `net.LookupIP`，避免单测依赖网络；实现时把函数引用成包级 var 方便 test 替换

## 预期产出

- 3 对 migration SQL 文件
- 1 个 model 文件 + 1 个 decision_card.go 扩展
- 2 个 llm 底层文件 (crypto + ssrf) + 对应测试
- 1 个 config.go 扩展 + 1 个 .env.example 更新
- commit: `feat(db): add llm_configs table and decision_cards synthesis columns`（只含 migration 文件）
- commit: `feat(llm): add crypto and ssrf primitives`（含 crypto/ssrf 和对应测试）
- commit: `feat(model): extend models for llm config and card synthesis source`（model + config + .env.example）
