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
