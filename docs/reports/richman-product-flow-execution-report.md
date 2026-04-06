# Richman 产品动线实施执行报告

## 文档定位

本报告记录 `docs/plans/richman-product-flow-plan.md` 及其 21 个 step 的实施过程中出现的所有问题、修复结果和遗留决策项。每完成一个 step 后追加一段记录，最终由用户验收。

## 执行方式

- 隔离模式：git worktree 在 `.claude/worktrees/product-flow`，分支 `product-flow-redesign`
- 执行流：Subagent 驱动开发（superpowers:subagent-driven-development）
- 每个 step 流程：implementer → spec compliance review → code quality review → 必要修复循环 → 任务标记完成
- 冷却规则（最终版本）：
  - Step 03 完成后冷却 **1 小时**
  - 冷却结束后一次性连续执行 Step 04 至 Step 21 全部 18 个 step，**中间无冷却**
  - 冷却实现：`sleep 3600` 后台命令 + `run_in_background=true`，启动后立即结束 turn，等通知回来继续
  - 历史规则变更：先后经历过"按 phase 边界 20 min"、"Step 03 之间 50 min"、"phase 边界 30 min"等多版本，最终以"只在 Step 03 完成后冷却 1 小时"为准

## 全局规则

1. **冲突一次性升级**：遇到字段/API/命名冲突时，一次性全部升级到最新方案，彻底弃用旧字段，不保留兼容层
2. **问题追踪**：所有过程问题记录在本报告中，能修复的必须修复，无法决策的保留到最后验收
3. **零 AI 痕迹**：commit message、分支名、代码注释、文件内容均不出现 AI/Claude/Anthropic 相关信息
4. **Lint 必过**：每次修改后立即 lint，通过后才能进入下一步
5. **本地环境缺 golangci-lint 与 sqlc**：所有 step 的 lint 验证用 `go build ./... && go vet ./... && go test ./...` 作为替代

## Step 01 数据库迁移

### 实施结果
- Commit: `9019772` feat(db): add migrations 006-008 for product flow redesign
- 创建 6 个迁移文件（006/007/008 的 up/down 对）
- 迁移回滚 → 再应用 roundtrip 通过
- 所有 ADD COLUMN / DROP COLUMN / CREATE INDEX / DROP CONSTRAINT 均带 IF NOT EXISTS / IF EXISTS
- risk_preference CHECK 约束包含 conservative/neutral/aggressive 三值

### 已修复问题
- 无（一次通过）

### 已记录但未修复的观察项（code quality review 标注的 minor 建议）
1. **007 缺少意图注释**：`risk_preference` 默认 `'neutral'` 的原因（与 CHECK 约束兼容 backfill）没写在 migration 里。留待后续维护时补注释即可，非功能影响
2. **badge_state 无 CHECK 约束**：当前 `badge_state VARCHAR(32) DEFAULT 'none'` 是自由文本。8 种 enum 在 Step 02 落到 Go 代码里已稳定，可以在后续 migration 补一个 CHECK 约束硬约束
3. **execution_fingerprint 用空串默认值而非 NULL**：查询"无指纹"要写 `= ''` 而非 `IS NULL`。非 NULL 是为了方便 Step 03 生成时总有值，但语义上有点奇怪
4. **prev_card_id 无外键约束**：自引用但没声明 `REFERENCES decision_cards(decision_card_id)`。与项目现有风格一致（001-005 migration 都不用外键），所以刻意保留
5. **categories JSONB 不约束为 array 类型**：理论上可以存入任意 JSON 值。实际应用层会保证是数组，DB 层未做强校验

**判断**：以上 5 项全部是加固项，不是 bug。按 YAGNI 暂不处理，如果后续 step 需要再补。

### 无法决策项
- 无

### Review 结果
- Spec compliance: ✅ Pass
- Code quality: ✅ Approve

## Step 02 Recommendation 类型与徽章 Diff 算法

### 实施结果
- Commits:
  - `2d59a74` feat(analysis): add recommendation types and execution fingerprint
  - `0acf057` feat(analysis): add badge diff algorithm with state machine
- 创建 `backend/internal/analysis/recommendation/` 包（types.go / fingerprint.go + 测试）
- 创建 `backend/internal/analysis/diff/` 包（badge.go + 测试）
- 测试覆盖率：recommendation 100%、diff 100%
- 依赖：recommendation 包仅 stdlib；diff 包仅依赖 stdlib + recommendation（使用 ConfidenceShiftThreshold 常量）
- 8 种 BadgeState 各有 happy path 测试，5 个多状态优先级测试覆盖"同时命中取最高"

### 已修复问题
- 无（一次通过）

### 已记录但未修复的观察项（code quality review 标注的 minor 建议）

1. **Fingerprint 不含 `Execution.ValidDays`**
   - 位置：`fingerprint.go:28-49`
   - 现状：只对 Type、TargetPositionPct、StopLoss、TakeProfit、Steps 计算指纹，ValidDays 被排除
   - 影响：如果 LLM 保持同一计划但延长有效期从 7 天到 14 天，指纹不变，不会触发 plan_adjust 徽章
   - 处理：TRD §3.3 明确列出的稳定字段里不含 ValidDays，implementer 按 TRD 执行正确。如果 ValidDays 需要参与 diff，需要先改 TRD。**暂不修改，记录待用户决策**

2. **Fingerprint 不含 `TriggerPayload`**
   - 位置：`fingerprint.go:42-45`
   - 现状：只对每个 Step 的 TriggerType / TriggerValue / DeltaPct 计算指纹
   - 影响：如果 `TriggerPayload.PriceValue` 与显示文案 `TriggerValue` 不一致，会出现语义不同但指纹相同的情况
   - 假设：TRD 约定 `TriggerValue` 是 `TriggerPayload` 的规范化渲染结果（即两者总是同步），所以只 hash 文案即可
   - 处理：按 TRD 执行，保持现状。**记录待用户验证这一假设**

3. **`%.6f` 浮点格式化不防 NaN/Inf**
   - 位置：`fingerprint.go:32,43,58`
   - 现状：`fmt.Sprintf("%.6f", NaN)` 会输出 "NaN"，如果上游传入异常浮点会导致所有卡哈希相同
   - 处理：上游应在调用前校验输入。目前未修，记录

4. **nil vs 0.0 stopLoss 的指纹差异未单测**
   - 位置：`fingerprint_test.go:84-98`
   - 现状：存在覆盖，但没有一个用例明确对比 `stopLoss = nil` 与 `stopLoss = &0.0`
   - 处理：代码行为正确（"nil" vs "0.000000"），只是测试不够严谨。留作后续补测

5. **Compute 对 `TargetPositionPct` 使用 float 直接相等比较**
   - 位置：`badge.go:101`
   - 现状：`cur.TargetPositionPct != prev.TargetPositionPct`
   - 影响：如果 JSON round-trip 出现浮点漂移会误触发 plan_adjust
   - 处理：实际值是 LLM 输出的小数（50.0、62.5 等），能在 float64 里精确表示，MVP 可接受。记录

6. **TestCompute_ActionDowngrade 缺 delta 断言**
   - 位置：`badge_test.go:58-66`
   - 现状：只断言 state，未断言 confidence delta
   - 处理：与 upgrade 测试不对称。**留作后续补测**

7. **CardSnapshot 的 direction 字段是 plain string**（Important — code reviewer 特别强调）
   - 位置：`badge.go:36-38`
   - 现状：TrendDirection / PositionDirection / CatalystDirection 是 plain string，没有 typed enum
   - 影响：Step 03 wire 起来时，如果 service 层传入的字符串值与 test 数据不一致（例如 "upward" vs "up"）会静默产生 signal_flip
   - 处理：**Step 03 派发时已明确要求 service 层用 `string(trendResult.Direction)` 显式转换，并写 `buildCardSnapshot` helper 集中管理**。已在 Step 03 中修复

**判断**：1-6 全部是 minor / suggestion。7 已在 Step 03 中规避。

### 无法决策项
- **Fingerprint 是否包含 ValidDays / TriggerPayload**（待用户确认 TRD 的设计意图是否需要调整）

### Review 结果
- Spec compliance: ✅ Pass
- Code quality: ✅ Approve

## Step 03 Synthesis 扩展与分析管线集成

### 实施结果
- Commits:
  - `25726b7` feat(repo): expose new decision_card structured fields
  - `d2868b0` feat(synthesis): generate structured recommendation with template fallback
  - `50d525f` feat(analysis): integrate badge diff into card persistence pipeline
- 修改 model.DecisionCard 加入 7 个新字段 + `RecommendationDetailJSON()` helper
- 修改 decision_card_repo：重写为 scanCardRow / insertDecisionCardOn 共享 helper，新增 Pool() / CreateDecisionCardTx / GetLatestByHolding / GetLatestByHoldingTx
- 修改 synthesizer：SynthesisOutput 新增 Recommendation 字段、prompt 追加 recommendation 指令段、template fallback 按 action 类型生成默认执行计划
- 修改 AnalyzeHolding：新增 persistDecisionCardWithDiff tx 包装、computeCardDiff / buildCardSnapshot 纯函数 helper、card 构造填入所有新字段
- 新建 recommendation_prompt.go（拆分 prompt 构造与 fallback）
- 测试：6 个 synthesizer 测试 + 9 个 service 测试，全部通过

### 已修复问题
- **变量名冲突**：AnalyzeHolding 内部原有局部变量 `recommendation` 与新导入的 `recommendation` 包冲突，implementer 把局部变量重命名为 `rec` 解决
- **Commit 顺序**：implementer 发现 plan 建议的 synthesis → analysis → repo 顺序会导致中间 commit 不能独立编译，自行调整为 repo → synthesis → analysis，每个 commit 都能独立 build

### 待修复问题（按规则 A 必须在 Step 03 内一次性升级）

**JSON tag 冲突（Critical - 需要立即修复）**

- 问题：`model.DecisionCard.Recommendation string` 旧字段已占用 `json:"recommendation"` tag，implementer 为了不破坏兼容性把新的结构化字段改成了 `json:"recommendation_detail"`
- 按**规则 A**：不保留兼容层，彻底弃用旧字段
- 需要的修复：
  1. 新增 migration 009：`ALTER TABLE decision_cards DROP COLUMN recommendation;`（配套 down 恢复列）
  2. 删除 `model.DecisionCard.Recommendation string` 字段
  3. 重命名 `model.DecisionCard.RecommendationJSON` 的 JSON tag 从 `"recommendation_detail"` 回 `"recommendation"`
  4. 更新 `decision_card_repo.go`：INSERT 列表和 scan 里移除 recommendation 字段
  5. 更新 `service/analysis/service.go`：停止设置 card.Recommendation
  6. 更新 `synthesizer.go` 的 templateFallback：不再依赖 legacy recommendation string
  7. 更新所有现有使用 `card.Recommendation` 的代码路径（notification adapters、API handlers 等）
  8. 更新测试
- **这项修复会作为 Step 03 的 followup commit，spec + code quality review 通过后立即派发**

### 已记录但未修复的观察项

1. **DataSourceDegraded 恒为 false**
   - 位置：`service/analysis/service.go` 的 `computeCardDiff` / `buildCardSnapshot`
   - 现状：`datasource.AssetData` 没有 Degraded 标志字段，Input.DataSourceDegraded 被硬编码为 false
   - 影响：`BadgeDataDegraded` 在生产环境里永远不会触发
   - 处理：需要在 `datasource.AssetData` 加 Degraded 字段并由 fetcher 正确填写。属于独立改动，**留待后续单独的 step 或 step21 e2e verify 时处理**

2. **无真实 DB 集成测试**
   - 位置：`persistDecisionCardWithDiff` tx 路径
   - 现状：只有纯函数 `computeCardDiff` 被单测覆盖，tx 实际行为（FOR UPDATE、Commit、Rollback）未做端到端测试
   - 处理：需要 CI 级别的 Postgres 测试环境。**留待后续 CI 搭建时处理**

### 无法决策项
- 无（JSON 冲突已按规则 A 明确如何修复）

### Code quality review 反馈

Reviewer 结论：**Approve with minor follow-ups requested**。无 Critical，5 条 Important + 10 条 Minor。

#### 将在 Step 03 fix implementer 中一次性修复的项

| 编号 | 问题 | 修复策略 |
|---|---|---|
| Rule A | legacy `Recommendation string` 字段与 JSON tag `"recommendation"` 冲突 | 新增 migration 009 drop 列；删除 model 字段；新字段 JSON tag 改回 `"recommendation"`；更新 repo/service/synthesizer/tests |
| Important #1 | `persistDecisionCardWithDiff` 副作用 mutate 输入 card pointer（即使 rollback 也污染 caller） | 在 tx 内用 local copy，不污染输入 |
| Important #2 | `scanCardRow` 静默吞掉 recommendation_json 反序列化错误，产生零值 Recommendation 静默生成错误徽章 | 解析失败时用 zap.Warn 记日志（含 card_id）而不是静默 |
| Important #5 | `DataSourceDegraded` 硬编码 false 缺少 TODO 标记 | 在 call site 加 `// TODO(degraded)` 指向 `datasource.AssetData.Degraded` 未来字段 |
| Minor #1 | `debugRecommendation` 无调用（dead code） | 删除 |
| Minor #3 | `scanCard` 是 `scanCardRow` 的 0-value wrapper（dead code） | 删除 |
| Minor #4 | `buildCardSnapshot` 的 `string()` cast 是 no-op（model 字段已经是 string） | 移除 cast，加注释说明为何不用 typed Direction |
| Minor #10 | tx `Rollback` 用可能被取消的 ctx，pgx 会额外 "context canceled" 错误 | 改为 `_ = tx.Rollback(context.Background())` |

#### 延后项（非阻塞，记录待用户验收时决策或在后续 step 中处理）

| 编号 | 问题 | 延后原因 | 建议处理时机 |
|---|---|---|---|
| Important #3 | `DecisionCardRepo.Pool()` 方法破坏 repo 层封装 | reviewer 明确"worth doing in small follow-up" | Step 09 API DTO alignment 阶段重构为 `WithTx` helper |
| Important #4 | tx 路径零单元/集成测试覆盖 | 需要 Postgres 测试环境（docker-compose 已有），集成测试方案独立 | Step 21 E2E verification 或专门 CI step |
| Minor #2 | `extractJSON` 二次解析冗余 | 性能优化，非功能 | 不修复，可接受 |
| Minor #5 | 迁移边界首次分析普遍产生 plan_adjust 徽章（旧卡 TargetPositionRatio=0） | 一次性适应成本可接受 | 不修复，作为预期行为 |
| Minor #6 | `recommendationText` 放在 `synthesizer.go` 但只被 `recommendation_prompt.go` 使用 | 风格项 | 可后续 step 中顺手迁移 |
| Minor #7 | 局部变量 `rec` 命名不够自文档 | naming polish | 不修复 |
| Minor #8 | `insertDecisionCardSQL` 的 `$1..$30` 手工维护 | YAGNI，只有下次加列时才值得 | 等下次加列时一起 |
| Minor #9 | tx 错误 wrap 缺 `holding_id` 上下文 | 日志便利性，非功能 | 不修复 |

### Step 03 fix implementer 执行结果

**Commits:**
- `16537ec` feat(decision-card): drop legacy recommendation column and harden persistence
- `29128d9` chore(analysis): drop dead debugRecommendation helper and update test name

**Fixes 全部落地:**
- Fix 1 (Rule A): migration 009 drop legacy column；model 字段重命名为 `Recommendation recommendation.Recommendation` 带 `json:"recommendation"` tag；repo/service/tests 全部同步
- Fix 2: `persistDecisionCardWithDiff` 用 `toPersist := *card` 局部副本，不污染 caller 输入
- Fix 3: `DecisionCardRepo` 新增 `*zap.Logger` 字段，`NewDecisionCardRepo(pool, logger)`，recommendation_json 解析失败记 zap.Warn（含 card_id / holding_id），cmd/server/main.go 已同步传入
- Fix 4: computeCardDiff call site 加 `// TODO(degraded):` 注释
- Fix 5: `debugRecommendation` 删除（含相关 `fmt` import）
- Fix 6: `scanCard` wrapper 删除，全部 caller 改为直接调 `scanCardRow`
- Fix 7: `buildCardSnapshot` 去掉 no-op string cast，加说明注释；测试名同步更新
- Fix 8: `defer func() { _ = tx.Rollback(context.Background()) }()` 保证 rollback 不受 ctx 取消影响

**Grep 验证全清:**
- `card.Recommendation` 作为 legacy string 字段：0 match
- `recommendation_detail` JSON tag：0 match
- `debugRecommendation`：0 match
- `scanCard`（独立函数）：0 match
- `string(card.TrendDirection)` 系列 no-op cast：0 match

**验证结果:**
- `make migrate-down` → `make migrate-up` roundtrip 通过
- `go build ./... && go vet ./... && go test ./...` 全绿

**Implementer 发现的 latent bug（超出本次 fix 范围，记录待后续处理）:**
- 项目 migration runner 的 `internal/migration/runner.go` `splitStatements` 函数不会剥离 down migration 文件开头的 `--` 注释行，导致带头注释的 down migration 会在 pgx 层语法错误
- 这次 009_*.down.sql 初版带注释失败，按 001-008 down 文件的无注释惯例改回
- **建议:** 后续独立 issue 修复 runner 的 splitStatements 逻辑，使其与 up 文件保持一致的注释处理

### Review 结果
- Spec compliance (main Step 03): ✅ Pass
- Code quality (main Step 03): ✅ Approve with minor follow-ups（已在 fix pass 中全部处理）
- Spec compliance (Step 03 fix pass): ✅ Pass — 所有 8 项 fix 全部通过 grep + 测试 + 代码检查
- Code quality (Step 03 fix pass): ✅ Approve with minor follow-ups

### Step 03 fix pass 的 code quality review 观察项

**2 项 Important（已在 controller inline 处理）:**
1. `RecommendationDetailJSON()` 方法名在 Rule A 字段重命名后 stale（"Detail" 已无意义）
   - **已修复**: 重命名为 `RecommendationJSONBytes()`，更新 `decision_card_repo.go:132` 的唯一 caller，build + vet + test 通过
2. **API JSON 合约变更**: `model.DecisionCard` 的 `json:"recommendation"` tag 现在序列化的是完整的结构化 `recommendation.Recommendation` 对象（旧版是 plain string）
   - **当前 grep 确认** `backend/internal/api` 和 `backend/internal/notification` 没有任何地方依赖旧的 string 形态
   - **Step 09 API DTO alignment 必须知晓**: Dashboard / 详情页 / 推送渠道的 JSON 响应里 `"recommendation"` 字段从 string 变成了对象，前端消费时需要按新 schema 渲染
   - **记录此项供 Step 09 参考**，不再额外修改

**9 项 Minor（延后 / 不修复）:**
1. shallow copy `toPersist := *card` 的 slice 共享问题：当前只 mutate 标量字段所以安全；注释可以更明确"only scalar diff fields are mutated"。**不改**
2. TODO(degraded) 注释位置在 call site 上方而非 `false` 字面量旁：readability micro-polish。**不改**
3. Commit `16537ec` bundle 了 7 个 fix 导致难以单独 revert：接受现状，下次 fix pass 拆更细
4. Migration 009 down 恢复的 DEFAULT `''` 与原列定义（无 DEFAULT）不完全一致：无语义影响，可接受
5-9. reviewer 确认的"non-blocking" 观察项，详见上方 code quality review 原文

### Step 03 状态: **COMPLETED** ✅
- Commits (ordered): `25726b7` → `d2868b0` → `50d525f` → `16537ec` → `29128d9` → `9ebb00b`

## Step 04 权重管理器 risk_preference bias

### 实施结果
- Commit: `70f14e4` feat(weight): apply risk_preference bias on top of base weights
- 修改 `model/user.go`：新增 `RiskPreference string` 字段 + 3 个 enum 常量
- 修改 `repo/user_repo.go`：集中 `userSelectColumns` 常量，所有 `GetUserBy*` 方法同步读新列；新增 `GetRiskPreference(ctx, userID) (string, error)` 单列查询 helper
- 修改 `analysis/weight/manager.go`：新增 `ApplyRiskBias(current, assetType, pref)` 方法，与已有 `GetBaseWeights` / `AdjustWeights` 并存
- 修改 `service/analysis/service.go`：Deps 新增 `UserRepo`，AnalyzeHolding 按 `GetBaseWeights → ApplyRiskBias` 两步流水运行
- 修改 `cmd/server/main.go`：wire userRepo 进 analysisService.Deps
- 测试：12 个测试函数、30+ 个子测试，覆盖 neutral identity、sum-to-one、direction、truncation、range bounds

### Review 反馈与修复（controller inline）

Code quality review 返回 **Request minor changes**（3 Important + 9 Minor）。3 Important 已 inline 修复：

| 编号 | 问题 | 修复 |
|---|---|---|
| I1 | neutral 路径不是真 no-op，会走 clamp+normalize 造成浮点漂移风险 | `default:` 分支改为 `return current` 直接 early return |
| I2 | `math.Max(x, 0.01)` 在生产路径是死代码，在 unknown asset type 路径会静默改写合法输入 | 删除 ApplyRiskBias 里的 floor 语句（AdjustWeights 保留不动） |
| I3 | switch 用字符串字面量 `"conservative"` 而不是常量，与 model 包有漂移风险 | 新增包内常量 `prefConservative/Neutral/Aggressive`；测试文件加 compile-time 断言验证与 `model.RiskPreferenceXxx` 值一致；switch 改用常量 |
| M1 | `trendDelta` 声明但恒为 0 | 删除，`result.Trend = current.Trend` 显式表达 |

修复验证：
- `go test ./internal/analysis/weight/... -v` 全绿（原有 12 个测试函数 + 30 子测试均保持通过）
- `go test ./...` 全绿
- `go vet ./...` 无警告
- Compile-time 断言：如果 weight 包的 `prefXxx` 与 `model.RiskPreferenceXxx` 发生漂移，build 会直接 fail

### 延后项（6 条 Minor 记录不修复）

| 编号 | 问题 | 原因 |
|---|---|---|
| M2 | Unknown asset type 的 "skip clamp" 行为无测试 | 生产路径只有 4 种已知 asset type，unknown 是 test-friendly 保留路径 |
| M3 | 文档 comment "keeps the function total" typo | inline 改为 "keeps the function forgiving" 时顺手处理，已修 |
| M4 | `TestApplyRiskBiasDirection` 只覆盖 `a_share_broad` | 后续补 |
| M5 | SumsToOne 测试可以更 table-driven | 风格项 |
| M6 | 空串/unknown 值的 sum-to-one 覆盖 | 已在 M1 修复的 early return 下等效覆盖 |
| M7 | service 集成路径（nil repo / error / empty pref）的单测缺失 | 需要构造 mock userRepo，留到 Step 21 E2E 统一补 |
| M8 | `ApplyRiskBias` 命名可更明确 `BiasByRiskPreference` | 接受 reviewer "fine if kept" |
| M9 | service pipeline 注释可更明确 | 风格项 |

### Step 04 状态: **COMPLETED** ✅
- Commits: `70f14e4`（主实施）+ `d7b8374`（I1/I2/I3/M3 修复）

## Step 05 user_settings 服务与隐私守卫

### 实施结果
- Commits:
  - `4dcfff3` feat(model): add user profile fields for settings
  - `f861f03` feat(repo): expose user profile fields for settings
  - `2608780` feat(user_settings): add service with validation and reflection helpers
- 新增 6 个文件：service/service_test、money/money_test、privacy_guard/privacy_guard_test
- 修改 2 个文件：model/user.go（+3 字段 + 4 AssetType 常量）、repo/user_repo.go（userSelectColumns 扩展、scanUser 集中、UpdateUserSettings、MarkOnboardingCompleted）
- 关键设计：
  - UserRepo interface 定义在 service 包以便 fake 测试
  - PatchUserSettings 指针字段 + `ClearTotalCapitalCNY bool` tri-state 清空语义
  - AttachAmounts 反射：识别 `*Pct` 后缀 + `PositionRatio`/`TargetPositionRatio` 别名；top-level only；接受 float64/float32/*float64/*float32
  - AssertNoCapitalLeakage 运行时反射守卫，不使用 build tag（全构建启用）
- 测试覆盖率：91.7%
- `go test ./...` / `go build ./...` / `go vet ./...` 全绿

### Review 反馈与修复（controller inline）

Spec compliance: ✅ Pass。Code quality: **Request minor changes**（4 Important + 10 Minor）。

**4 Important 全部 inline 修复:**

| 编号 | 问题 | 修复 |
|---|---|---|
| I1 | `"amount"` 子串匹配过宽，会误报 `paymentAmount` / `minAmount` 等良性字段 | forbiddenSubstrings 改为具体列表：`totalcapital` / `positionamount` / `targetpositionamount` / `unrealizedamount` / `realizedamount`；错误消息增加"重命名或 `json:\"-\"`"的 escape hatch 提示 |
| I2 | 空 patch 仍触发 UPDATE 圆转 + 无意义 bump `updated_at` | service.PatchUserSettings 加 `isEmptyPatch` 短路：所有字段 nil + !ClearTotalCapitalCNY → 直接返回 `GetUserSettings` 结果 |
| I3 | walkGuard 无 visited set，对环形引用会 stack overflow | 文档加"Precondition: tree-shaped DTO only"前置条件说明（visited set 属于过度防御，API 层产出 DTO 保证无环） |
| I4 | walkStruct / walkType 重复 ~30 行 tag-check 逻辑 | 抽取 `checkFieldTag(sf, path)` + `tagViolation(tag, fieldPath)` 2 个共享 helper，DRY |
| 测试 | 空 patch 短路 / ClearFlag 覆盖 value 的 tri-state 语义无断言 | 新增 `TestPatchUserSettings_EmptyPatchShortCircuit` + `TestPatchUserSettings_ClearFlagOverridesValue`；原 `TestPatchUserSettings_RepoError` 改用非空 patch 保证打到 repo 路径 |

**Plan deviation 处理:**
- 原 step05 plan 写的 "守卫只在 `-tags debug` 构建中启用" 与实施方式冲突
- 按 Rule A 一次性升级：**更新 plan 文件**将相关描述改为"全构建启用，handler 层显式调用"，与实施保持一致

### 延后项（10 条 Minor 中的剩余部分）

| 编号 | 问题 | 处理 |
|---|---|---|
| M5 | `OnboardingCompletedAt` 在 service 返回 `*string` RFC3339 而非 `*time.Time` | 可接受，Step 09 统一 DTO 时再决定 |
| M6 | `MarkOnboardingCompleted` 本 step 无 consumer | 预期 Step 08 Onboarding service 调用 |
| M7 | `amountFieldFor("Pct")` 返回 `"Amount"` 的 edge case | 已在测试里 pin 住行为，不修改 |
| M8 | `AttachAmounts` 覆盖已填充的 Amount 字段未文档化 | 追加一行文档即可，不影响正确性 |
| M9 | 负 pct 无 warning | 预期语义（未实现亏损），不修复 |
| M11 | scanUser 维护成本（后续加字段时多处同步） | 未来可迁移 sqlc |

### Step 05 状态: **COMPLETED** ✅
- Commits: `4dcfff3` → `f861f03` → `2608780` → `1283f52`（inline review fixes）

## Step 06 LLM Vision provider

### 实施结果
- Commits:
  - `b3de0f3` feat(llm): add vision provider abstraction
  - `7f971ec` feat(llm): add claude vision implementation with mock tests
- 创建 `llm/vision.go`、`llm/vision_factory.go`、`llm/claude/vision.go`、`llm/claude/vision_test.go`
- 修改 `llm/claude/register.go`、`config/config.go`、`.env.example`
- 关键设计：独立 visionRegistry（与文本 Provider 解耦）；7 个 typed sentinel errors；ctx.Err() 重校正保证 timeout 分类；VISION_API_KEY → CLAUDE_API_KEY 单 vendor 回退
- 测试：7 个 httptest 场景（success / 5xx / 429 / timeout / malformed / 4xx / invalid request）全部通过

### Review 反馈与修复（controller inline）

Spec: ✅ Pass。Code quality: **Approve with follow-ups**（0 Critical, 5 Important, 7 Minor）

**Important 中的 I2 / I3 / I5 inline 修复**（Step 07 依赖 ErrVisionInvalidRequest 合约）:

| 编号 | 问题 | 修复 |
|---|---|---|
| I2 | MIME 类型未按 Claude allowlist 预先校验，可能浪费 API 请求 | 新增 `allowedVisionMIMEs` map（jpeg/png/gif/webp），不在列表的 MIME 在 client 层直接返回 `ErrVisionInvalidRequest` |
| I3 | 图像字节数无本地 cap，5MB+ 载荷会白白打到 Claude 再被拒 | 新增 `maxVisionImageBytes = 5 MB` 常量，超过直接返 `ErrVisionInvalidRequest` 含明确字节数错误消息 |
| I5 | `VISION_API_KEY → CLAUDE_API_KEY` 回退无测试 | 新增 `TestRegisterVision_FallsBackToClaudeKey` 构造最小 config 验证 apiKey 字段正确继承；同时补两个 MIME/size 拒绝测试 |

**Important 中的 I1 / I4 延后**:
- I1：`LLM_VISION_TIMEOUT_SECONDS=0` 被 WithVisionTimeout 静默忽略 → 记录待 config.validate() 统一处理
- I4：error response body 日志无脱敏 → 记录（API 响应不太可能回显 base64 图像数据，生产风险可控）

### 延后项（5 条 Minor）

| 编号 | 问题 | 处理 |
|---|---|---|
| M1 | visionRegistry 写入无 mutex（init-time only） | 加 "init-time only" 注释即可，风险极低 |
| M2 | `VisionResponse.Model` 略 leaky（Claude 特定） | 与 ChatResponse.Model 一致，保持 |
| M3 | 7 个 error 类别数量辩护 | 延后 |
| M4 | `visionMessage.Content` 字段对齐 gofmt | gofmt 自动处理 |
| M5 | `UsageHint` 用 map[string]any | 可考虑后续改为 typed struct |
| M6 | `config.validate` 未校验 VisionTimeout >= 0 | 后续 config 集中整理时补 |

### Step 06 状态: **COMPLETED** ✅
- Commits: `b3de0f3` → `7f971ec` → `ee2c708`（inline fixes）

## Step 07 截图 OCR 服务与 API

### 实施结果
- Commits:
  - `ea5d712` feat(screenshot): add recognition service with confidence thresholds
  - `bf5e218` feat(api): add import-screenshot endpoint with multipart upload
- 创建 screenshot service 包（service/prompts/parser + tests）
- 创建 api/v1/screenshot.go handler + test
- 修改 cmd/server/main.go wire VisionProvider + screenshotService
- 关键设计：
  - 内存 fixed-window rate limiter（10/hour/user，sync.Mutex，Options.Now 可注入便于测试）
  - 任意字段 >= ConfidenceLow (0.60) → "ok"；全部低 → "low_quality"；vision fail / invalid JSON → "failed" + 中文 warning
  - Vision error HTTP 200 + status=failed（TRD §4.6 降级策略，前端无需检查 HTTP 状态码）
  - 可选 VisionProvider（nil 时 graceful degrade）
- 测试：service 11 个测试 + handler 6 个测试 + parser 测试，全部通过

### Review 反馈与修复（controller inline）

Spec: ✅ Pass。Code quality: **Approve with follow-ups**（0 Critical, 4 Important, 7 Minor）

**4 Important 全部 inline 修复:**

| 编号 | 问题 | 修复 |
|---|---|---|
| I1 | Rate limiter 对"一次性用户"内存无限增长（allow() 只对返回用户剪裁） | 新增 `janitorSweep()` 辅助方法可由后台 ticker 或测试调用扫描所有 stale 用户 entries。保持 `allow()` 不变以维持其最小锁窗口原则 |
| I2 | Parser 非严格模式，LLM prompt drift 时静默产生空 holdings | 改为 `json.NewDecoder(...).DisallowUnknownFields()`，drift 直接变 `ErrInvalidJSON` → failed 状态 |
| I3 | `MaxBytesReader` 触发时 FormFile 返回的错误被映射为 400 VALIDATION_ERROR 而不是 413 | 使用 `errors.As(&http.MaxBytesError)` 检测并映射到 413 FILE_TOO_LARGE |
| I4 | 信任客户端 multipart Content-Type header，恶意客户端可伪造 MIME | 改为始终 `http.DetectContentType(data)` 嗅探真实字节，不再回退到 fileHeader.Header。测试 fixture 改用真实 PNG magic bytes（0x89 50 4E 47 ...）|

**测试调整:**
- screenshot_test.go 新增 `fakePNGBytes()` helper，所有 fixture 从 `[]byte("fake")` 改为含真实 PNG 签名的字节序列

### 延后项（7 条 Minor）

| 编号 | 问题 | 处理 |
|---|---|---|
| M5 | stripCodeFence 对嵌套 fence 不处理 | 嵌套 fence 会在 JSON decode 层失败成 ErrInvalidJSON，功能正确 |
| M6 | ConfidenceHigh 常量在 gradeStatus 中未使用 | 前端消费，保留常量作为 JSON schema 文档 |
| M7 | maxUploadFormBytes 64KB envelope 注释 | 已存在说明 |
| M8 | Vision error classification 未对所有 7 个 sentinel 显式 case | 不影响行为 |
| M9 | AssetTypeGuess 无 confidence 字段 | schema 故意设计 |
| M10 | io.ReadAll 完整加载到内存 | 5MB cap 已限制 |
| M11 | newTestRouter 直接写 ContextKeyUserID | 测试辅助可接受 |

### Step 07 状态: **COMPLETED** ✅
- Commits: `ea5d712` → `bf5e218` → `0f12a48`（inline fixes）

## Step 08 Onboarding 服务与 API

### 实施结果
- Commits:
  - `e98f5fe` feat(onboarding): add status service with mark and reset
  - `f9533ea` feat(api): expose onboarding endpoints with prod-mode reset guard
- 创建 `service/onboarding/service.go` + test（GetStatus/MarkCompleted/Reset 3 个方法）
- 创建 `api/v1/onboarding.go` + test（GET/POST/DELETE 路由）
- 修改 `repo/user_repo.go` 新增 `ClearOnboardingCompleted`
- 修改 `config/config.go` 新增 `IsProduction()` fail-closed helper
- 修改 `cmd/server/main.go` wire onboardingService
- 关键设计：
  - `EnvGuard` 窄接口（1 方法），`*config.Config` 直接满足
  - `UserRepo` 窄接口（3 方法），测试用 fake
  - Reset 在 production 直接返 `ONBOARDING_RESET_FORBIDDEN` 403
  - Mark/Reset 返回 `*Status` 让 handler 无需二次 GET
- 测试：11 个 service 测试 + 6 个 API 测试 + IsProduction/IsDev 专项测试

### Review 反馈与修复（controller inline）

Spec: ✅ Pass。Code quality: **Approve with follow-ups**（0 Critical, 2 Important, 8 Minor）。

**2 Important + 1 Minor 已 inline 修复:**

| 编号 | 问题 | 修复 |
|---|---|---|
| I1 | `IsProduction` 大小写敏感，`APP_ENV=DEV` 会被误判为 prod | 改为 `strings.ToLower(c.App.Env)` 比较；`IsDev` 同步改为 `strings.EqualFold`。新增 `config_test.go`，12 组用例覆盖 dev/test/staging/prod/空/大小写变体 |
| I2 | Reset 的 `s.env != nil` 防御让 nil env 变成"永远允许重置" | `NewService` 改为 fail-fast：nil users / nil env 都直接 panic；`Reset` 移除 nil check；新增 `TestNewService_PanicsOnNilDeps` + `TestReset_ProductionTakesPrecedenceOverNotFound` 两个测试固化契约 |
| Minor #3 | Reset "prod 优先于 not-found" 的顺序未固化测试 | 新增 `TestReset_ProductionTakesPrecedenceOverNotFound` 覆盖"用户不存在 + prod 环境 → 403 而非 404"（防止通过错误码泄露用户存在） |

### 延后项（7 条 Minor）

| 编号 | 处理 |
|---|---|
| M4 | `*Status` 返回值（ergonomic，保留） |
| M5 | router.go 未修改（handler 自注册，与其他 handler 一致） |
| M6 | fakeUserRepo 的 copy 语义注释 |
| M7 | `statusFromUser` 的 ts copy 注释 |
| M8-M10 | 风格项 |

### Step 08 状态: **COMPLETED** ✅
- Commits: `e98f5fe` → `f9533ea` → `61f6501`（inline fixes）

## Step 09 API DTO alignment

### 实施结果
- Commits:
  - `d294b85` feat(holding): expose category column on model and repo
  - `4d4f2cf` feat(api): project holdings to DTO with category and position amount
  - `d2fa59e` feat(api): expose structured recommendation fields and amount projection on decision cards
  - `b7863a3` feat(api): add user settings endpoints and total capital lookup
  - `4f907dc` test(notification): assert adapter Message carries no capital info
- 关键设计：
  - DTO 定义在 api/v1 package（handler 旁边），不污染 model 层
  - `CapitalProvider` 窄接口（单方法）在 consumer package 声明，service 满足
  - 优雅降级：capital 读取失败时返回 nil，Amount 字段 omitempty
  - notification adapter.Message 审计后确认 4 个 string 字段全部合规，无需重构，测试锁定契约
- 测试：5 个 user_settings API 测试 + notification leakage 测试 + 全部现有测试通过

### Review 反馈与修复（controller inline）

Spec: ✅ Pass。Code quality: **Approve with minor fixes**（0 Critical, 3 Important, 7 Minor）。

**3 Important 全部 inline 修复:**

| 编号 | 问题 | 修复 |
|---|---|---|
| I1 | `categoryArg any` 写入隐式依赖 pgx 把 untyped nil 映射为 SQL NULL | 改为 `sql.NullString` 对称读写，与 scan 侧一致，加注释说明 |
| I2 | `GetTotalCapitalCNY` 名义 "cheap read" 但实际调 `GetUserByID` 加载完整用户行 | 在 `repo/user_repo.go` 新增专用 `GetTotalCapitalCNY(ctx, userID) (*float64, error)` 单列查询；service 改为直接调新方法；更新 UserRepo interface + 两个 fake repo |
| I3 | `ClearTotalCapitalCNY=true + TotalCapitalCNY=非 nil` 的矛盾组合被 service 静默接受 | `validatePatch` 新增明确拒绝，返回 400 `INVALID_TOTAL_CAPITAL`；将原 `TestPatchUserSettings_ClearFlagOverridesValue` 改写为 `TestPatchUserSettings_ClearAndSetConflictRejected` 验证 400 响应和 repo 未被调用 |

### 延后项（7 条 Minor）

| 编号 | 处理 |
|---|---|
| M4 | portfolio.go / decision_card.go 行数增长，后续 step 可拆 dto.go |
| M5 | CapitalProvider 可迁到独立 file |
| M6 | resolveCapital 在两个 handler 重复 |
| M7 | 缺直接的 DTO 投影测试 |
| M8 | resolveCapital signature 略不一致（context.Context vs *gin.Context） |
| M9 | main.go wiring 顺序注释 |
| M10 | user_settings handler PATCH 400 分支未走 handleServiceError |

### Step 09 状态: **COMPLETED** ✅
- Commits: `d294b85` → `4d4f2cf` → `d2fa59e` → `b7863a3` → `4f907dc` → `da76c74`（inline fixes）
- **Phase 3（后端阶段）全部完成**

## Step 10 前端路由与守卫重构（Phase 4 前端基础第 1 步）

### 实施结果
- Commits:
  - `a257310` feat(auth): add onboarding guard
  - `bf3e8be` refactor(routes): collapse menu to dashboard, portfolio, settings
  - `c6aaf35` chore(pages): remove deprecated analysis, decision-cards-list, notifications pages
  - `2e3fdc3` review inline fixes
- 创建 OnboardingGuard + 临时 useOnboardingStatus hook + 4 个 onboarding 占位页 + HelpPage 占位
- 重构 routes.tsx 使用 OnboardingShell / AppShell 两层 shell 组件
- MainLayout 菜单精简到 3 项 + menuFooterRender 帮助入口
- 删除 analysis / decision-cards-list / notifications 页面与相关 features
- 修复 pre-existing 的 test/setup.ts 和 test/utils.tsx 的 Biome 错误
- tsconfig.json 加 vitest/jest-dom types 让 tsc 识别测试 globals

### Review 反馈与修复（controller inline）

Spec: ✅ Pass。Code quality: **Approve with follow-ups**（0 Critical, 4 Important, 6 Minor）。

**4 Important 全部 inline 修复:**

| 编号 | 问题 | 修复 |
|---|---|---|
| I1 | OnboardingGuard 的 redirect 分支在 Step 11 接入前完全未被测试 | 新增 `onboarding-guard.test.tsx` 5 个用例：loading spinner、pre-onboarding redirect、onboarding pass-through、post-onboarding redirect、main app pass-through。Mock useOnboardingStatus + useNavigate |
| I2 | MainLayout 的 `collapsed` useState 是 dead code（sider 被强制展开） | 删除 useState；`menuFooterRender` 简化为单一展开样式（无 collapsed 分支） |
| I3 | i18n nav.analysis / nav.decisionCards / nav.notifications 过时 key 未清理 | zh.json + en.json 精简 nav section 至 `dashboard / portfolio / settings / help` 四项 |
| I4 | `/portfolio/:id/transactions` 静默 alias 到 PortfolioEditPage，无注释说明 | 加 `TODO(step17)` 注释指向 PRD §4.4 要求 |

### 延后项（6 条 Minor）

| 编号 | 处理 |
|---|---|
| M5 | tsconfig.json 的 types 字段可能需要 tsconfig.test.json 拆分 | 不修复，当前配置验证通过 |
| M6 | useOnboardingStatus hook 的 error 字段未被消费 | Step 11 接入真实 API 后一并处理 |
| M7-M10 | 风格项 | 不修复 |

### 验证
- `pnpm lint:all` PASS（Biome + tsc + depcruise 全绿，75 modules / 169 deps）
- `pnpm build` PASS（vite build 3.37s）
- `pnpm test onboarding-guard` PASS（5/5）

### Step 10 状态: **COMPLETED** ✅
- Commits: `a257310` → `bf3e8be` → `c6aaf35` → `2e3fdc3`
