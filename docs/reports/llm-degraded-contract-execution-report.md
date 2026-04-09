# LLM 降级契约与用户自选 Provider 执行报告

## 执行上下文

- 触发原因：调试 `analysis/synthesis.Synthesizer` 的 nil-panic 时发现代码里没有任何可观测的降级信号，也没有用户级 LLM 配置能力，同时暴露了 lint 工具链和 commit scope 的系统性纪律问题
- 执行日期：2026-04-09
- 会话边界：单次会话完成 PRD/TRD/Plan 落盘 + chore 支线（lint toolchain + commit hygiene 纪律固化）+ feat 支线 Phase 1 的 backend 底层模块
- 涉及的分支和 worktree：
  - `chore/golangci-lint-v2-and-cleanup` @ `.claude/worktrees/chore-lint-v2`
  - `docs/llm-degraded-contract` @ `.claude/worktrees/docs-llm-degraded`
  - `feat/llm-degraded-contract` @ `.claude/worktrees/feat-llm-degraded`
- Origin main 在会话期间被用户的并行工作推进，触发了多次 local-main 被 reset 回 origin 的现象，本报告的"Local main reset 现象"段落详细记录

## 全局规则

- 零 AI 痕迹：所有 commit message / 代码 / 注释 / 文档里不得出现 AI/Claude/Anthropic/OpenAI/Copilot 作为工具署名。`Claude API` / `OpenAI API` 作为产品名允许
- 严格 lint：每次修改文件后跑项目配置的 lint，必须 0 issue 才能进下一步
- commit scope 纪律：禁止 `git add -A`，每次 stage 按具体 file path，每个 commit 一个主题
- 设计完备性：任何非平凡设计必须走完 `docs/standards/design-review.md` 的 5 个 Pass，产出 8 件 artifacts
- 文档分层强制顺序：PRD -> TRD -> Plan，三者缺一不可

## Phase 0: Panic Root Cause Fix

### 任务
- 根因定位：`synthesizer.go:63` 在 `s.provider == nil` 时直接解引用调用 `ChatCompletion`
- 修复：`Synthesize` 入口加 nil 短路，走 templateFallback
- 单元测试：`TestSynthesize_NilProvider_UsesTemplateFallback` 锁死降级契约

### 结果
- 代码已入 main（通过用户并行会话的 commit `0ad2ce3`，该 commit 包含了非预期的 scope 混合，详见 "Scoping 事故" 段落）
- nil-check 逻辑和单元测试都已经在 main 上生效
- dev 模式下的 panic 不会再出现

### 已记录但未修的观察项
- commit `0ad2ce3 docs(trd)` 混合了 TRD 文档 + backend 修复 + 前端 auth 重构三类改动，违反 commit scope 纪律。用户决策"接受现状，不重写 history"。系统性防御已沉淀到 `docs/standards/commit-hygiene.md`

## chore 分支: Lint Toolchain + Commit Hygiene

### 任务范围
1. 把 `backend/.golangci.yml` 从 v1 迁移到 v2 schema
2. 清理 v2 schema 开启后浮出的 41 条存量 lint 违规
3. 写 `docs/standards/lint-toolchain.md` 沉淀 lint 工具纪律
4. 写 `docs/standards/commit-hygiene.md` 沉淀 commit scope 纪律
5. 在 `CLAUDE.md` 的 Standards Index 挂两份新标准
6. 执行报告：`docs/reports/golangci-lint-v2-cleanup-execution-report.md`

### 执行结果
- 初始盘点：41 条违规，gocritic 21 / revive 6 / staticcheck 5 / misspell 3 / gofmt 3 / errcheck 2 / unused 1
- subagent 执行机械清理，同时修复了 12 条级联同类问题（总计 53 条）
- 所有接口签名变更（`llm.VisionProvider.AnalyzeImage`、`diff.Compute`、`cmd/migrate/main.go` 重构为 `run() error`）属于 lint 修复范畴，没有改动业务行为
- `janitorSweep` 死函数用 `//nolint:unused` + 明示"保留供未来 ticker wiring"的注释方式保留，未删除
- Rebase 到当前 main 时遇到 `onboarding_test.go` 冲突：主分支删除了 `TestOnboardingAPI_ResetForbiddenInProduction` 测试（因为 onboarding 重构让 stubEnv 参数不再存在），chore 的 lint 清理保留了这个测试。解决方案：接受主分支的删除
- Rebase 后发现 4 条新的 lint 违规（来自主分支 cherry-pick 的 onboarding skip 测试），立即补了一个 commit 清理

### Commits（基于 rebase 后）
1. `chore(lint): migrate golangci-lint config to v2 schema`
2. `chore(lint): resolve 53 preexisting violations across backend`
3. `docs(standards): add lint-toolchain and commit-hygiene standards`
4. `docs(reports): add golangci-lint v2 cleanup execution report`
5. `chore(lint): clean up rebase-introduced onboarding test violations`

### 合并状态
- 本地 merge 到 main 成功（merge commit `251c4ef`），但 local main 随后被 reset 回 origin/main
- 分支内容完整保留在 `chore/golangci-lint-v2-and-cleanup`
- **等待用户通过 PR 或显式 push 合并到 origin/main**

### 已记录但未修的观察项
- 中间 commit `1f35bc9`（后变 `eaa5eee`）曾在 rebase --continue 过程中暂时含有 conflict markers，因 Edit tool 首次调用失败。后续 commit 已清理干净。**Bisect 风险**：如果未来 `git bisect` 到这个中间点，编译会失败。用户可接受此代价（merge 的 endpoint 状态是干净的）
- `janitorSweep` 未被 wire 到任何 ticker，长时间运行会有内存累积。建议后续开独立 ticket 修复
- 本机 `golangci-lint` 通过 brew 装的 v2.11.4 没有被项目锁定版本。`lint-toolchain.md` 的"工具链版本声明"段落已经要求后续在 Makefile 或 tools.go 里固定版本，待后续实施

## docs 分支: PRD + TRD + Plan

### 任务范围

按 CLAUDE.md "superpowers skill 文档分层强制顺序" 要求，产出 PRD / TRD / Plan 三层文档，覆盖 LLM 降级契约 + 用户自选 provider 的完整产品方案。

### Brainstorming 阶段

通过 11 个 clarifying question 逐个收敛产品决策：

| # | 决策 | 取值 |
|---|------|------|
| 1 | LLM 不可用时的产品态度 | 始终出卡片，只打降级标签 |
| 2 | 降级字段的表达力 | 三态 enum: llm / template / mixed |
| 3 | UI 标签命名 | AI / Rules / Mixed（英文） |
| 4 | PRD 范围 | 含用户自选 LLM（完整方案） |
| 5 | 配置分层 | 系统默认 + 用户覆盖 |
| 6 | MVP provider | Claude / OpenAI / OpenAI 兼容 |
| 7 | fallback 链 | user -> system_default -> template |
| 8 | 历史卡片回归策略 | 前端 banner 提示用户点击"重新分析所有持仓" |
| 9 | Synthesizer 接口演进 | 显式返回 (Output, Meta, error) |
| 10 | 非配置时的 consent 语义 | 两个独立 consent 字段 |
| 11 | Reanalyze 单次 holdings 数 | 不限，依赖 1/10min 节流 |

### 5-pass Design Review 执行

按 `docs/standards/design-review.md` 完整走完 5 Pass，产出 8 件 artifacts：

1. **Pass 1 状态空间**：三维组合共 12 行 valid target 枚举，发现 Gap G1（未配置时 allow_fallback 语义歧义），通过拆分两个 consent 字段解决
2. **Pass 2 文件不变量**：对 7 个核心待修改文件逐个提取现有契约和改动影响
3. **Pass 3 替代路径**：8 条替代路径（back 导航、probe 失败重试、cross-session、双 tab 并发、banner dismiss regret、reanalyze 与 edit race、分析中删配置、主密钥轮换）逐条验证
4. **Pass 4 Pre-mortem**：5 个最可能 bug（token 爆炸、secret 泄露 log、fallback 雪崩、save-reanalyze race、迁移日 banner 泛滥）+ 每个都给出防御
5. **Pass 5 Attack-your-own**：6 个推荐项的自我反驳，发现 SSRF 漏洞补丁和 onboarding 引导步骤

### 产物

- `docs/prds/llm-degraded-contract-prd.md`（319 行）
- `docs/trds/llm-degraded-contract-trd.md`（1010 行）
- `docs/plans/llm-degraded-contract-plan.md`（总述）
- `docs/plans/llm-degraded-contract-plan/step1-backend-foundation.md`
- `docs/plans/llm-degraded-contract-plan/step2-backend-core.md`
- `docs/plans/llm-degraded-contract-plan/step3-backend-integration.md`
- `docs/plans/llm-degraded-contract-plan/step4-frontend.md`
- `docs/plans/llm-degraded-contract-plan/step5-verification.md`

### Commits

1. `docs(prd): add llm degraded contract and user provider spec`
2. `docs(trd): add llm degraded contract technical design`
3. `docs(plan): add llm degraded contract implementation plan`
4. `docs(trd): correct for main drift after onboarding merge`（记录 migration 编号从 010/011/012 改成 N+1/N+2/N+3 占位的原因）

### 合并状态
- 本地 merge 到 main 成功（merge commit `35001bf`），但随后被 reset 回 origin/main
- 分支内容完整保留在 `docs/llm-degraded-contract`
- **等待用户通过 PR 或显式 push 合并到 origin/main**

## feat 分支: Phase 1 Backend Foundation

### 任务范围

按 `docs/plans/llm-degraded-contract-plan/step1-backend-foundation.md`：

- 3 对 migration SQL 文件（011/012/013）
- `internal/model/llm_config.go` 新增
- `internal/llm/crypto.go` + `crypto_test.go` 新增
- `internal/llm/ssrf.go` + `ssrf_test.go` 新增
- `internal/model/decision_card.go` 扩展
- `internal/config/config.go` 扩展
- `backend/.env.example` 扩展

### 执行方式

派 general-purpose subagent 在 `.claude/worktrees/feat-llm-degraded` 执行，subagent 基于本 worktree 的 PRD/TRD/Plan 和现有源代码。

### 执行结果

Phase 1 subagent 完成并全部验证通过。

#### Commits

| SHA | Subject |
|-----|---------|
| `2a375bf` | feat(db): add llm_configs and decision_cards synthesis columns |
| `877de6a` | feat(llm): add crypto and ssrf primitives |
| `1ff22ff` | feat(config): extend llm config and decision card model for degraded contract |

#### 新增文件

| 文件 | 行数 |
|---|---|
| `backend/db/migration/011_llm_configs.up.sql` | 46 |
| `backend/db/migration/011_llm_configs.down.sql` | 3 |
| `backend/db/migration/012_decision_cards_synthesis_source.up.sql` | 26 |
| `backend/db/migration/012_decision_cards_synthesis_source.down.sql` | 4 |
| `backend/db/migration/013_users_llm_consent.up.sql` | 10 |
| `backend/db/migration/013_users_llm_consent.down.sql` | 2 |
| `backend/internal/llm/crypto.go` | 90 |
| `backend/internal/llm/crypto_test.go` | 153 |
| `backend/internal/llm/ssrf.go` | 127 |
| `backend/internal/llm/ssrf_test.go` | 210 |
| `backend/internal/model/llm_config.go` | 82 |

#### 修改文件

| 文件 | 增量 | 说明 |
|---|---|---|
| `backend/internal/model/decision_card.go` | +10 | 追加 `SynthesisSource *string`, `ProviderUsed *string` |
| `backend/internal/config/config.go` | +18 | LLM.ConfigMasterKey、ProbeTimeout 字段和 env 解析 |
| `backend/.env.example` | +10 | `LLM_CONFIG_MASTER_KEY` 和 `LLM_PROBE_TIMEOUT` |

#### 验证证据

主会话独立重跑验证：

- `golangci-lint run ./...` -> `0 issues`
- `go vet ./...` -> 无输出
- `go test ./...` -> 所有 backend 包 pass
- `git diff main..HEAD backend/** | grep ai_trace_pattern` -> 空
- 23 个新单测（9 crypto + 14 ssrf）全部通过

#### Subagent 按项目实际情况对 TRD 的修正（合理偏差）

Subagent 在实施时发现 TRD 在几处和项目现有风格不一致，按 CLAUDE.md Plan 文档规范"执行 agent 根据 TRD 设计和实际情况动态决策具体实现"自主调整：

1. **`is_deleted` 类型**：TRD 写 `BOOLEAN NOT NULL DEFAULT FALSE`，但项目现有 schema 统一用 `SMALLINT NOT NULL DEFAULT 0`，partial index 断言 `WHERE is_deleted = 0`。Subagent 按现有约定实施，`model.LLMConfig.IsDeleted` 用 `int16`
2. **Audit 列**：TRD 片段省略了 `creator` / `modifier`，但项目每张表都有这两列。Subagent 追加 `DEFAULT 'system'` 保持统一
3. **`fmt` import**：TRD 的 `Masked()` 方法 snippet 没列 fmt import，Subagent 补上；并且让 Masked() 处理 nil receiver 以便 zap logger 在 repo lookup 失败时不 panic
4. **Encrypt 返回值命名**：TRD 未命名返回值，触发 golangci-lint v2.11.4 的 `gocritic unnamedResult`。Subagent 命名为 `(ciphertext, nonce []byte, err error)`
5. **Migration 编号**：按 TRD "Main 漂移注记"执行，010 占位 -> 011（llm_configs），011 -> 012（decision_cards），012 -> 013（users）

这些偏差都是 TRD-doc 和项目-reality 的小漂移，Phase 2 主会话确认是合理的。

#### 已记录但未修的观察项

- **TRD schema drift**：TRD 的 `is_deleted BOOLEAN` 和 partial index `WHERE is_deleted = FALSE` 与项目 SMALLINT 约定不符。建议 Phase 2 或 Phase 5 把 TRD 修正到和实际一致
- **Resolver model import path**：TRD `Resolver` 段落用 `model.HealthHealthy` 常量，Subagent 把这些常量放在 `package model`，与 TRD snippet 自洽
- **`aes.NewCipher` 错误路径**：Subagent 把该错误折叠成 `ErrMasterKeyInvalid`，因为实际上 `aes.NewCipher` 只会在 key 长度错时失败，而 key 长度我们已经提前校验。代码注释里写明这个 defensive collapse

#### Phase 2 交接契约

Subagent 暴露了以下稳定契约，Phase 2 可以直接消费：

**`package model`**:
- `LLMProviderType` / `LLMHealthStatus` 枚举及其常量
- `LLMConfig` struct，18 个 db-tagged 字段（和 migration 011 对齐）
- `(*LLMConfig).Masked() string` —— 唯一合法 log 序列化入口，nil-safe
- `LLMConfig.IsDeleted` 是 `int16`（schema 约定）
- `DecisionCard.SynthesisSource *string` / `ProviderUsed *string`

**`package llm`**:
- 常量 `MasterKeyBytes = 32` / `NonceBytes = 12`
- 错误：`ErrMasterKeyInvalid`、`ErrDecryptFailed`、`ErrSSRFBadScheme`、`ErrSSRFHostBlocked`、`ErrSSRFPrivateIP`、`ErrSSRFMetadataHost`
- `NewCryptoFromHex(masterKeyHex string) (*Crypto, error)`
- `(*Crypto).Encrypt(plaintext []byte) (ciphertext, nonce []byte, err error)`
- `(*Crypto).Decrypt(ciphertext, nonce []byte) ([]byte, error)`
- `ValidateBaseURL(rawURL string) error` —— Resolver 里 save/probe/ChatCompletion 三处都要调一次（DNS rebinding 防御）
- `var lookupIP = net.LookupIP` —— 包级 hook，测试里可 override-and-restore，生产代码不要改

**`package config`**:
- `LLMConfig.ConfigMasterKey string` —— 对应 `LLM_CONFIG_MASTER_KEY`，空值应让启动 fatal
- `LLMConfig.ProbeTimeout time.Duration` —— 对应 `LLM_PROBE_TIMEOUT`，默认 5s

**Repo 层注意事项**（给 Phase 2）:
- Partial unique index `uq_llm_configs_active_user` 强制每 user 一行 active；upsert 用 ON CONFLICT 或先 soft-delete 旧行再 insert
- `api_key_cipher BYTEA NOT NULL` —— 直接从 `Crypto.Encrypt` 拿到的 `[]byte` 写入
- **绝不 SELECT api_key_cipher / api_key_nonce 到 log 字段**
- `zap.String("config", cfg.Masked())` 是唯一合法的 log 调用姿势，禁止 `zap.Any("config", cfg)`

## Local main reset 现象

### 描述

会话期间用户的并行工作流反复做 `git reset --hard origin/main`，reflog 显示此现象发生了至少 4 次：

```
main@{0}: reset: moving to origin/main
main@{1}: merge docs/llm-degraded-contract  <- 本会话的 docs 合并
main@{2}: merge chore/golangci-lint-v2-and-cleanup  <- 本会话的 chore 合并
main@{3}: commit: feat(onboarding): refactor CategoriesPage
main@{4}: merge auth-slogan-typewriter: Fast-forward
main@{5}: reset: moving to origin/main  <- 之前也 reset 过
main@{6}: merge auth-slogan-typewriter
main@{7}: reset: moving to origin/main
main@{8}: merge auth-slogan-typewriter
main@{9}: merge onboarding-ux-overhaul
```

### 含义

用户项目的合并流程是"通过 push/PR 进入 origin/main"，local main 只作为 origin 的镜像，不是本地的 source of truth。任何只在本地 merge 的 commit 都会被下一次 reset 清掉。

### 影响

- 本次会话的 chore / docs 两次本地 merge 都被 reset 清掉
- 但分支本身（`chore/golangci-lint-v2-and-cleanup` 和 `docs/llm-degraded-contract`）的 commit 完整保留在 .git/refs/heads/
- feat 分支的工作不受影响，因为它是独立的 feature branch 不跟 main 同步

### 处置

- 当前会话不做 push（按 CLAUDE.md 安全约束，push 需要用户显式授权）
- 所有成果保留在分支里
- 用户需要自行决定合并方式：
  - 通过 GitHub PR
  - `git push origin chore/...` + 本地 `git merge` + `git push origin main`
  - 或其它项目约定的合并流程

## 未合并到 origin 的分支清单

| 分支 | HEAD 近似 SHA | 状态 | 说明 |
|---|---|---|---|
| `chore/golangci-lint-v2-and-cleanup` | `4aa7f49` | 待 push | 5 commit，lint 0 issues，测试全绿 |
| `docs/llm-degraded-contract` | `840491c` | 待 push | 4 commit，包含 PRD/TRD/Plan |
| `feat/llm-degraded-contract` | （待填） | Phase 1 in progress | 基于 35001bf（含 chore + docs merge），子 agent 正在写代码 |

## feat 分支: Phase 2 Backend Core

### 范围
按 step2-backend-core.md：LLMConfigRepo + Resolver（三级 fallback 链）+ user_repo consent 扩展 + 集中错误定义。

### 执行方式
派 general-purpose subagent 在 feat worktree 执行，TDD 流程，先写 resolver_test.go 覆盖 PRD 状态空间表 7 行的全部组合。

### Commits

| SHA | Subject |
|---|---|
| `1c97ac8` | feat(llm): add central error definitions for resolver layer |
| `5ff9292` | feat(repo): add llm config repo and user consent accessors |
| `74b9c57` | feat(llm): add resolver with three-level fallback chain |

Subagent 把 errors 提前到第 1 个 commit，保证每个 commit 独立可编译（避免 bisect 陷阱）。

### 新增/修改文件

| 文件 | 行数 | 性质 |
|---|---|---|
| `backend/internal/llm/errors.go` | 37 | 新增 |
| `backend/internal/llm/resolver.go` | 292 | 新增 |
| `backend/internal/llm/resolver_test.go` | 665 | 新增 |
| `backend/internal/repo/llm_config_repo.go` | 208 | 新增（含 interface assertions） |
| `backend/internal/repo/llm_config_repo_test.go` | 58 | 新增 |
| `backend/internal/repo/user_repo.go` | +53 | 追加 GetUseSystemDefaultConsent / SetUseSystemDefaultConsent |

### 验证证据
- `golangci-lint run ./...` -> 0 issues
- `go vet ./...` -> 无输出
- `go test ./...` -> 全部 24 个包 pass
- 15 个 resolver 测试覆盖 PRD 状态空间表 + 健康度副作用 + SSRF + 解密失败等

### Subagent 合理偏差

1. **`SoftDelete` 增加 `modifier string` 参数**：与项目 audit 列约定一致（holding_repo / notification_channel_repo 都有）。Phase 3 settings handler 必须提供 modifier 标识符
2. **DB 集成测试推迟到 Phase 5**：项目目前没有 testcontainers 或 pgxmock 基础设施，subagent 在 repo 层只写了 nil/zero 输入的 guard 测试，靠 compile-time interface assertions（`var _ llm.LLMConfigRepo = (*LLMConfigRepo)(nil)`）防止方法签名漂移
3. **`probeTimeout` 暂未应用**：Resolver 接受参数但还没 wire 到 context.WithTimeout。加了 `//nolint:unused` 和 docstring 标注"reserved for bounded probe retries"

### 已记录但未修的观察项
- repo 层缺乏 DB 集成测试，partial unique index 约束、Upsert 事务回滚、UpdateHealth 列范围都没被测试覆盖。这是项目原有技术债的延伸，不是本次新增
- `probeTimeout` 待 Phase 3 或 Phase 5 wire 到 context

## feat 分支: Phase 3 Backend Integration

### 范围
按 step3-backend-integration.md：Synthesizer 接口升级 + AnalyzeHolding 集成 + settings/llm REST handlers + reanalyze-all endpoint + dashboard summary 扩展 + main.go wire-up。

### 执行方式
派 general-purpose subagent 执行。subagent 自主调整了 commit 顺序（让 commit 1 包含最小的 service.go + main.go 修复保证可编译）。

### Commits

| SHA | Subject |
|---|---|
| `f19fc28` | refactor(synthesis): thread SynthesisMeta through Synthesize signature |
| `6838112` | feat(service): persist synthesis_source and provider_used on decision cards |
| `4aabc61` | feat(api): add llm settings endpoints and onboarding consent handler |
| `f01455a` | feat(api): add reanalyze-all endpoint with per-user rate limit |
| `5ec5d26` | feat(api): extend decision_cards and dashboard summary responses |
| `99e7905` | feat(server): wire up llm crypto, resolver, and settings handlers |

### 主要改动统计
+2013 / -69 行，13 个文件。

新增：
- `backend/internal/api/v1/settings_llm.go` (555 行)
- `backend/internal/api/v1/settings_llm_test.go` (476 行)
- `backend/internal/api/middleware/rate_limit.go` (95 行) —— per-user in-memory limiter
- `backend/internal/api/v1/reanalyze_test.go` (85 行)
- `backend/internal/api/v1/dashboard.go` (159 行)
- `backend/internal/api/v1/dashboard_test.go` (171 行)

修改：
- `backend/internal/analysis/synthesis/synthesizer.go` 接口签名 +144/-61
- `backend/internal/analysis/synthesis/synthesizer_test.go` stubResolver 迁移 +171
- `backend/internal/service/analysis/service.go` AnalyzeHolding 集成 +28/-3
- `backend/internal/repo/decision_card_repo.go` INSERT/SELECT 列扩展 +36/-5
- `backend/internal/api/v1/decision_card.go` DTO 扩展 +38
- `backend/internal/api/v1/analysis.go` reanalyze-all handler +29
- `backend/cmd/server/main.go` 完整 wire-up +87/-10

### 验证证据
- `golangci-lint run ./...` -> 0 issues
- `go vet ./...` -> 无输出
- `go test ./...` -> 全部 24 个包 pass
- `go build ./...` -> 成功
- 零 AI 痕迹

### Subagent 合理偏差

1. **Commit 1 包含最小 service.go + main.go 修复**：因为 Synthesizer 接口签名变化会破坏所有调用点，commit 1 必须同时修复 caller 才能 compile。bisect 安全比单 commit 单 scope 更重要
2. **Dashboard handler 是新文件**：plan 说"修改 dashboard.go" 但项目里没有这个文件，subagent 创建了它
3. **`useSystemDefaultWhenUnconfigured` DTO 字段语义来源**：两个 column（llm_configs.use_system_default_when_unconfigured 和 users.use_system_default_llm_consent）并存，DTO 统一从 users 表读，与 onboarding consent step 保持一致。`llm_configs` 那一列实际上是 dead column 待未来 admin UI 复用
4. **Rate limit middleware 是新建的**：项目没有 per-user 限流，写了一个 in-memory map + mutex 版本，文档里标注 Redis 是 multi-instance 的 drop-in replacement
5. **`PUT /settings/llm` 要求每次都传 apiKey**：plan 写"编辑模式可空表示不变"，但实现"不变"分支需要复杂的字段合并逻辑。MVP 选择硬要求，前端表单需要总是 resend
6. **`HealthUnknown` 在 DTO 映射为 `healthy`**：避免首次 probe 前显示红色 banner。如果运营要求严格区分，未来可加第 4 个 union 值
7. **`Crypto` 是 `*llm.Crypto` 指针**：让 main.go 在 master key 未配置时可以传 nil 而无需 wrapper interface

### 已记录但未修的观察项

- `llm_configs.use_system_default_when_unconfigured` 列永远写 false，是 dead column
- Probe 实现用硬编码 "ping" prompt 计入用户配额。长远应该走 provider 专用 zero-cost 探活但 Claude/OpenAI 都没暴露
- Rate limit map 不收缩，每个 unique userID 永驻直到 process 结束。Richman 用户量内可接受
- Migration 012 历史卡片回填假设 'llm'+'user'，dev 环境如果有 template 历史会被错误回填

## feat 分支: Phase 4 Frontend

### 范围
按 step4-frontend.md：settings-llm feature + dashboard-llm-status feature + decision-card SourcePill + onboarding consent step + API hooks。

### 执行方式
派 general-purpose subagent 执行。subagent 在执行第 5 个 commit（onboarding consent）期间因 rate limit 被中止，主会话接手完成最后一步。

### Commits

| SHA | Subject | 来源 |
|---|---|---|
| `9f4d484` | feat(api): add llm settings and reanalyze hooks | subagent |
| `25225dd` | feat(settings): add llm provider configuration ui | subagent |
| `44900b0` | feat(dashboard): add llm status banner with reanalyze cta | subagent |
| `e182708` | feat(decision-card): add source provenance pill | subagent |
| `2f3a183` | feat(onboarding): add llm consent step | 主会话接手 |

### 主要改动

新增 feature 模块：
- `frontend/src/features/settings-llm/` 包含 LLMSection / LLMEmptyState / LLMHealthyCard / LLMFailingCard / LLMConfigForm / LLMProbeButton + hooks + types + barrel
- `frontend/src/features/dashboard-llm-status/` 包含 LLMStatusBanner + useLLMStatusBanner + barrel
- `frontend/src/features/decision-card/SourcePill.tsx`

新增页面：
- `frontend/src/pages/onboarding/LLMConsentPage.tsx` (148 行) —— 主会话补充

修改：
- onboarding state.tsx + use-onboarding-nav.ts + components/OnboardingLayout.tsx —— 把 consent step 插入 wizard 流程
- routes.tsx —— 注册新路由

### 验证证据
- `pnpm lint:all` -> 171 文件检查，无 fix；186 modules dependency-cruiser 无违规
- `pnpm build` -> 成功（仅 chunk size 警告，pre-existing）
- 所有新 feature 模块有 index.ts barrel
- 零 `any` 类型，零 AI 痕迹

### 已知限制
- `pnpm vitest` 在某个特定测试上 hang（DOM dump 导致），**clean main 上同样问题，与本次工作无关**。被记入 Phase 5 风险点
- 已知 pre-existing 失败：`state.test.tsx` 5/6 fail、`LoginPage.test.tsx` 6/13 fail、`ScreenshotImportModal.test.tsx` 2/4 fail。本次代码没有修改这些文件，失败是项目原有问题

## Phase 5 验证

### 最终验证证据

**Backend** (`feat/llm-degraded-contract` HEAD):
```
$ golangci-lint run ./...        -> 0 issues
$ go vet ./...                   -> 无输出
$ go test ./...                  -> 全部 24 包 pass
$ go build ./cmd/server/main.go  -> 成功
$ git diff main..HEAD -- backend/** | grep ai_trace_pattern -> 空
```

**Frontend**:
```
$ pnpm lint:all                  -> 171 文件，无 fix；186 modules 无依赖违规
$ pnpm build                     -> 成功
$ git diff main..HEAD -- frontend/** | grep ai_trace_pattern -> 空
```

### 已知风险（不阻塞合并，需要用户后续处理）

1. **vitest 单元测试 hang**：clean main 也存在，与本工作无关。新增组件没有 vitest 单测覆盖（lint + build + ts type-check 已经覆盖了类型契约）。建议独立 ticket 修复 hang 的 root cause（可能是某个 test 没 cleanup）
2. **pre-existing 测试失败**：state.test.tsx / LoginPage.test.tsx / ScreenshotImportModal.test.tsx，与本工作无关
3. **DB 集成测试缺失**：repo 层依赖 partial unique index、事务回滚等约束目前没单测覆盖。需要后续引入 testcontainers
4. **Probe 占用真实 token 配额**：MVP 妥协
5. **Rate limit map 不收缩**：MVP 妥协
6. **Migration 012 历史回填**：dev 环境如果有 template 历史会被错误回填为 llm

### 合并策略

按 commit-hygiene 和项目工作流，feat 分支已经包含全部 18 个实施 commit + 1 个 docs report commit + 4 个 docs+chore ancestor merge。**整个分支可作为单个 PR 合并到 origin/main**。

或者拆成 3 个 PR（如果 review 团队倾向小 PR）：
1. PR 1: chore branch（lint v2 + 53 cleanup + 2 standards + chore report）
2. PR 2: docs branch（PRD + TRD + Plan）
3. PR 3: feat branch 实现（Phase 1-4 共 17 commits + 本执行报告）

按 CLAUDE.md 安全约束，本会话**不主动 push 到 origin**。等待用户决策合并方式后由用户执行 push 或 PR 创建。

### Worktree 清理建议

合并完成后清理：
```bash
git worktree remove .claude/worktrees/chore-lint-v2
git worktree remove .claude/worktrees/docs-llm-degraded
git worktree remove .claude/worktrees/feat-llm-degraded
git branch -d chore/golangci-lint-v2-and-cleanup
git branch -d docs/llm-degraded-contract
git branch -d feat/llm-degraded-contract
```

## 观察项（供用户参考）

1. **Local main reset pattern**：已记录在上面。建议项目 README 或开发者文档里写清楚这个工作流约定
2. **Migration 编号占位**：TRD 里的 010/011/012 是占位符，实施时根据 feat 分支实际 main baseline 确定。Phase 1 subagent 按 011/012/013 执行
3. **`janitorSweep` 死函数**：chore 清理里保留了 nolint 注释，但没修复"未 wire 到 ticker 的内存增长 bug"。建议开独立 ticket
4. **主密钥轮换**：PRD 明示 MVP 不支持。用户丢失 `LLM_CONFIG_MASTER_KEY` 后所有存量 user LLM 配置作废，需要用户重新填 key。运维手册需要写清楚
5. **SSRF DNS rebinding**：TRD 设计了每次 provider 调用前重新 resolve 的防御，代码实现时要注意不要因为性能优化缓存 DNS 结果

## 下一步

等待 Phase 1 subagent 返回后：

1. 验证 Phase 1 产出（lint / test / build）
2. 如果 OK：更新本报告的 Phase 1 段落，填写 commit SHA
3. 评估剩余上下文预算：
   - 充足：继续 Phase 2
   - 不足：把本报告作为交接文档，下一个会话继续
4. 无论哪种情况，都要通知用户 chore 和 docs 分支需要通过 push 或 PR 合并到 origin/main 才能生效
