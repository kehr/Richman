# Step 02 后端 Model 与 Repo 层

## 任务目标

扩展 `User` Go 模型添加 `OnboardingSkippedAt` 字段，同步更新所有共享的 SQL 查询常量和 scan 函数。新增 `MarkOnboardingSkipped` 方法，将 `ClearOnboardingCompleted` 改名为 `ResetOnboarding` 并扩展语义同时清两列。`MarkOnboardingCompleted` 原子清 `skipped_at` 保证互斥。

## 涉及文件

修改：
- `backend/internal/model/user.go`
- `backend/internal/repo/user_repo.go`

调用点同步更新（改名的 `ClearOnboardingCompleted` → `ResetOnboarding`）：
- `backend/internal/service/onboarding/service.go`（仅改方法名，完整 service 重构放在 step03）

## 设计依据

- PRD §2.1 Go 模型
- PRD §2.2 Repo 层
- PRD 附录 B Pass 2 契约打破警报：`userSelectColumns` 与 `scanUser` 列数同步是高严重度契约

## 实施要点

- `User` 新增字段紧贴 `OnboardingCompletedAt`，用 `json:"onboardingSkippedAt,omitempty"`
- `userSelectColumns` 常量追加新列，所有 SELECT / UPDATE RETURNING 自动生效
- `scanUser` 函数同步追加 `&skippedAt` 参数
- `MarkOnboardingCompleted` SQL 追加 `onboarding_skipped_at = NULL` 子句，保持原有 `COALESCE(completed_at, NOW())` 幂等语义
- 新增 `MarkOnboardingSkipped(ctx, userID)` 方法，SQL 对称：`COALESCE(skipped_at, NOW())` + `completed_at = NULL`
- `ClearOnboardingCompleted` 改名为 `ResetOnboarding`，SQL 同时清两列（`completed_at = NULL, skipped_at = NULL`）
- 所有其他 `Get*` / `Patch*` 方法复用更新后的 `userSelectColumns`，无需额外改动

## 验证标准

1. `cd backend && go vet ./...` 通过
2. `go build ./...` 无错误
3. 既有 `user_repo` 相关测试（如有）全部通过
4. 手动 SQL 检查：通过 psql 调用 `MarkOnboardingCompleted` 后，验证 `completed_at` 有值且 `skipped_at` 为 NULL
5. 手动 SQL 检查：调用 `MarkOnboardingSkipped` 后，验证 `skipped_at` 有值且 `completed_at` 为 NULL
6. 手动 SQL 检查：调用 `ResetOnboarding` 后，验证两列都是 NULL
7. 重复调用 `MarkOnboardingSkipped` 两次，第二次的时间戳与第一次相同（幂等）

## 依赖说明

前置：step01（schema 已经有 `onboarding_skipped_at` 列）
