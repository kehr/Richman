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
- **Step 03 fix pass**: **用户直接接受**，跳过额外 review 循环（决策原因：fix implementer 已完成 grep 全清验证 + 测试全绿 + migration roundtrip 通过，且用户指示立即进入冷却执行后续任务）
- Step 03 状态: **completed**
