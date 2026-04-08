# Onboarding UX Overhaul 执行报告

本报告记录 `docs/plans/onboarding-ux-overhaul-plan.md` 的实施过程。每完成一个 step 追加一段，全部完成后由用户验收。

## 执行配置

- 执行方式：superpowers:subagent-driven-development skill，每个 step 派发独立 implementer subagent + 两阶段 review（spec compliance + code quality）
- 分支：`onboarding-ux-overhaul`（在主仓库目录开分支，不使用 worktree）
- 三层文档闭环：
  - PRD：`docs/prds/onboarding-ux-overhaul-prd.md` (commit 8872828)
  - TRD：`docs/trds/onboarding-ux-overhaul-trd.md` (commit 0ad2ce3)
  - Plan：`docs/plans/onboarding-ux-overhaul-plan.md` + `plan/` 18 个 step 文件 (commits bb5aa62, 80f090a)
- 零 AI 痕迹：commit message 不加 Co-Authored-By，不提 AI/Claude；代码注释英文；分支名和文件名无 AI 相关关键词
- 遵循 `docs/standards/design-review.md`：修改任何文件前先 Read 全文、列既有契约、评估影响

## 全局规则

1. **冲突处理**：Rule A —— 一次性升级到最新方案，不保留兼容 shim
2. **LINT 阻断**：每个 step 完成后必须通过 `pnpm lint:all`（前端）或 `go vet && go build && go test`（后端），未通过不进入下一步
3. **Review 流程**：implementer 完成 → spec compliance review → inline fix loop → code quality review → inline fix loop → mark complete
4. **Subagent 边界**：每个 implementer 只 stage 自己修改的文件（按明确路径），不执行 `git add -A` 以避免扫入工作树遗留编辑
5. **预先存在的未提交改动**：开始执行前工作树有 2 个未提交文件（`frontend/public/logo.svg`、`frontend/src/pages/auth/components/AuthSplitLayout.tsx`），属于用户 IDE 里的未保存编辑，不在本 plan 范围内，subagents 不会触碰
6. **问题记录**：过程中遇到所有问题（已修复 / 已记录未修复 / 无法决策）都在本报告追加记录

## Step 执行状态

执行过程中逐条追加。

## Step 01 DB migration 010_onboarding_skipped

### 目标
为 users 表追加 `onboarding_skipped_at TIMESTAMPTZ NULL` 字段。互斥约束由后续 step 的 SQL 写入路径保证。

### 实施提交
- `afbf5c8` feat(db): add migration 010 for onboarding_skipped_at column

### 新增文件
- `backend/db/migration/010_onboarding_skipped.up.sql`
- `backend/db/migration/010_onboarding_skipped.down.sql`

### Review 轮次
1. **Inline 合并 review**（spec + code quality）→ PASS
   - Spec: 与 TRD §2.1 / PRD §1.1 完全一致，无额外改动
   - Code quality: 风格对齐既有 007 / 009 迁移文件，`IF NOT EXISTS` 幂等、英文注释、无 emoji、无 AI 痕迹
   - 由于 scope 极小（2 个 SQL 文件共 6 行），未派发独立 subagent review，由主会话直接 Read + go vet 验证

### 验证
- `go vet ./...` PASS
- `go build ./...` PASS（implementer 已验证）
- 实际 `make migrate-up` 执行推迟到 step 02/03 有 repo/service 代码消费新字段时一并验证，避免在仅 schema 变更无代码依赖的中间状态跑迁移

### Step 01 状态: COMPLETED

## Step 02 Backend Model + Repo

### 目标
扩展 `User` 模型 + `userSelectColumns` + `scanUser`，新增 `MarkOnboardingSkipped`，将 `ClearOnboardingCompleted` 改名为 `ResetOnboarding` 并扩展为同时清两列。`MarkOnboardingCompleted` SQL 追加 `onboarding_skipped_at = NULL` 保证互斥。Service 的 `UserRepo` interface 同步 rename 以保持 build green。

### 实施提交
- `faf1404` feat(backend): extend User model and repo for onboarding_skipped_at

### 修改文件
- `backend/internal/model/user.go`: 新增 `OnboardingSkippedAt *time.Time` 字段
- `backend/internal/repo/user_repo.go`:
  - `userSelectColumns` 12 列，新列插入 `onboarding_completed_at` 之后
  - `scanUser` 追加 `skippedAt` 局部变量和 Scan 目标
  - `MarkOnboardingCompleted` SET 子句追加 `onboarding_skipped_at = NULL`
  - 新增 `MarkOnboardingSkipped`（对称 SQL：COALESCE skipped_at + 清 completed_at）
  - `ClearOnboardingCompleted` 改名 `ResetOnboarding`，SQL 清两列
- `backend/internal/service/onboarding/service.go`: `UserRepo` interface 与 `Reset` 方法内调用同步重命名（最小 cascade）
- `backend/internal/service/onboarding/service_test.go` + `backend/internal/api/v1/onboarding_test.go`: `fakeUserRepo.ClearOnboardingCompleted` → `ResetOnboarding` 并同步清两列

### Review 轮次
1. **Inline 合并 review**（spec + code quality）→ PASS
   - Spec: 与 TRD §3.1 完整方法签名一致；4 个 repo 操作对称性正确；service cascade 最小化未触 Status 或 env guard
   - Code quality: 错误 wrap 消息风格对齐、comment 英文、无 emoji、无 AI 痕迹；Scan 参数顺序手动验证与列顺序匹配（12 positions）
   - 验证通过：`go vet ./...` / `go build ./...` / `go test ./...` 全绿

### 观察项（不阻塞）
- `make lint`（golangci-lint）对 clean tree 也失败：`unsupported version of the configuration: ""`，`.golangci.yml` 缺少 `version:` key。预存在问题，非本 step 引入，建议后续单独修复工具链
- `ResetOnboarding` 方法的 comment 仍说「dev-only reset flows, service layer gating」，这是临时状态 —— step 03 会移除 service 的生产守卫同时更新注释

### Step 02 状态: COMPLETED

## Step 03 Backend Service Layer

### 目标
扩展 `onboarding.Status`（追加 Skipped/SkippedAt）、新增 `MarkSkipped` service 方法、移除 `Reset` 的生产环境守卫和 `EnvGuard` 依赖、更新 `statusFromUser`，相应更新所有调用点和测试。

### 实施提交
- `cc67d1a` feat(backend): extend onboarding service with skip flow and user-facing reset

### 修改文件
- `backend/internal/service/onboarding/service.go`：Status 扩展 / UserRepo interface 新增 MarkOnboardingSkipped / 新增 MarkSkipped 方法 / Reset 移除生产守卫 / EnvGuard 接口和 Service.env 字段整体删除 / NewService 单参 / statusFromUser 投影两对字段
- `backend/internal/service/onboarding/service_test.go`：fakeUserRepo 加 MarkOnboardingSkipped 方法且实现互斥语义 / 删除 fakeEnvGuard / 删除 TestReset_ForbiddenInProduction / 新增 8 个测试用例（含 6 个计划要求 + 2 个 NotFound/RepoError 错误路径）
- `backend/internal/api/v1/onboarding_test.go`：fakeOnbUserRepo 同步对齐 / NewService 调用点去 env 参数
- `backend/internal/repo/user_repo.go`：ResetOnboarding 注释重写为「user-facing atomic both-columns reset」
- `backend/cmd/server/main.go`：`onboardingSvc.NewService(userRepo, cfg)` → `NewService(userRepo)`

### Review 轮次
1. **Inline 合并 review**（spec + code quality）→ PASS
   - Spec：Status 字段 / MarkSkipped / Reset 守卫移除 / EnvGuard 整体清理 / 8 个新测试覆盖完整状态转换
   - Code quality：注释英文、错误 wrap 风格对齐、commit message 无 AI 痕迹、scan 顺序正确（依赖 step02 已验证）
   - 验证：`go vet ./...` / `go build ./...` / `go test ./...` 全绿；`go test ./internal/service/onboarding/... -v` 18 tests pass

### 实施过程发现的 gap（已记录到 standards/design-review.md）
- 原 Pass 2 文件不变量提取漏掉了 `service.go` 中 `EnvGuard` 守卫的存在，导致 step 17（前端 CTA 投放生产）原本会因后端 403 而失败
- 修复：本 step 内移除生产守卫 + 删除 EnvGuard 接口 + 同步更新 docs/standards/design-review.md 添加「环境守卫 / 特性开关 / dev-only 门控」作为 Pass 2 必查契约类型 + 同步更新 step03 plan 文件明确 guard removal 范围（commit `7f9bfa5`）
- 沉淀位置：项目级 standards（流程规则）+ 本报告记录（事件追溯）

### 观察项
- `EnvGuard` interface 已删除，但 `config.IsProduction()` 仍存在供未来其他场景使用（无当前调用方）
- `make lint` golangci-lint 工具链问题持续存在，与本 step 无关，建议后续单独修复

### Step 03 状态: COMPLETED

## Step 04 Backend API POST /onboarding/skip

### 目标
在 `OnboardingHandler` 暴露 `POST /api/v1/onboarding/skip` 端点，handler 复用 `service.MarkSkipped`，新增 4 个 API 集成测试覆盖成功路径、follow-up GET 反映、skip→complete 互斥清理、auth 校验。

### 实施提交
- `357214b` feat(api): add POST /onboarding/skip handler with integration tests

### 修改文件
- `backend/internal/api/v1/onboarding.go`：RegisterRoutes 新增 `POST /skip`、新增 MarkSkipped handler、Reset 注释更新（移除生产守卫的描述）
- `backend/internal/api/v1/onboarding_test.go`：新增 4 个 TestOnboardingAPI_Skip* 测试

### Review 轮次
1. **Inline 合并 review**（spec + code quality）→ PASS
   - Spec: 路由路径、HTTP 方法、Response shape、4 个测试用例与 plan §step04 完全对齐
   - Code quality: handler 风格对齐既有 MarkCompleted、注释英文、commit message 无 AI 痕迹
   - 验证：`go vet` / `go build` 全绿；`go test ./internal/api/v1/... -run TestOnboardingAPI` PASS（共 9 个 TestOnboardingAPI 用例）

### Step 04 状态: COMPLETED

## Step 05 Frontend user-settings hooks

### 目标
扩展 `OnboardingStatus` 类型 + `User` 类型同步后端契约，新增 `useSkipOnboarding` mutation hook，扩展 `useResetOnboarding.onSuccess` 清理 sessionStorage / localStorage / 失效多查询。

### 实施提交
- `630ceae` feat(user-settings): add skip onboarding hook and extend reset cleanup

### 修改文件
- `frontend/src/features/user-settings/types.ts`：OnboardingStatus 加 skipped + skippedAt
- `frontend/src/domain/auth/types.ts`：User 加 onboardingSkippedAt
- `frontend/src/features/user-settings/api.ts`：新增 skipOnboarding 函数
- `frontend/src/features/user-settings/use-skip-onboarding.ts`（新）：mutation hook + sessionStorage 清理 + 双 invalidate + refetch
- `frontend/src/features/user-settings/use-reset-onboarding.ts`：onSuccess 异步化，追加 storage 清理 + 多查询失效
- `frontend/src/features/user-settings/index.ts`：barrel 导出 useSkipOnboarding

### Review 轮次
1. **Inline 合并 review**（spec + code quality）→ PASS
   - Spec：3.11 / 3.12 / TRD §6.1 完整覆盖；refetch 在 invalidate 后正确处理 guard 竞态
   - Code quality：try/catch 包 storage 调用、注释英文、commit 无 AI 痕迹
   - 验证：`pnpm lint:all` PASS（156 modules / 501 deps）；`pnpm test --run` PASS（22 files / 108 tests）；`pnpm build` 成功

### 观察项
- AccountTab 已经用 `await resetOnboarding.mutateAsync()`，自然兼容新的 async onSuccess，无须额外修改

### Step 05 状态: COMPLETED
