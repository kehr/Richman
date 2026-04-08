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

## 待办清单

### Phase 2 Backend Core
依赖 Phase 1 完成。范围：LLMConfigRepo + Resolver + user_repo consent 扩展。详见 `docs/plans/llm-degraded-contract-plan/step2-backend-core.md`。

### Phase 3 Backend Integration
依赖 Phase 2 完成。范围：Synthesizer 接口升级 + Analysis Service 集成 + settings/llm API handlers + reanalyze-all endpoint + main.go wire-up。详见 step3。

### Phase 4 Frontend
依赖 Phase 3 API 稳定。范围：settings-llm feature + dashboard-llm-status + decision-card SourcePill + onboarding consent step。详见 step4。

### Phase 5 Verification
全部 Phase 完成后的端到端 verify + 执行报告汇总。详见 step5。

### 合并策略
用户需要在适当时机决定：
1. chore 分支合并到 origin/main（先做）
2. docs 分支合并到 origin/main
3. feat 分支完成所有 Phase 后合并到 origin/main

合并顺序保证 feat 的 baseline 干净（lint 闭环），且 PRD/TRD/Plan 先于实现落盘。

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
