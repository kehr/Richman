# Step 03 后端 Onboarding Service 层

## 任务目标

扩展 `onboarding.Status` DTO 增加 `Skipped` 和 `SkippedAt` 字段，新增 `MarkSkipped` service 方法，更新 `Reset` 内部调用链指向新命名的 repo 方法。所有 service 级测试对应新增。

## 涉及文件

修改：
- `backend/internal/service/onboarding/service.go`

创建或修改：
- `backend/internal/service/onboarding/service_test.go`（新增多个测试用例）

## 设计依据

- PRD §2.3 Service 层 / §6.1 后端测试要求 / 附录 A 状态空间表（验证互斥字段的组合约束）
- TRD §3.2 Service 层（Status struct 完整定义 + MarkSkipped / GetStatus / Reset 完整方法体）

## 实施要点

- `Status` 结构体追加 `Skipped bool` + `SkippedAt *time.Time`，JSON tag 用 `omitempty`
- `GetStatus` 从 `repo.GetUserByID` 的结果同时读取 completed 和 skipped 两对字段填充
- 新增 `MarkSkipped(ctx, userID) (*Status, error)`，内部调用 `repo.MarkOnboardingSkipped`，错误包装风格与现有 `MarkCompleted` 对齐
- `Reset(ctx, userID)` 内部调用改为 `repo.ResetOnboarding`（step02 已改名），同时清两列
- **移除 `Reset` 的生产环境守卫**：原有的 `if s.env.IsProduction() { return 403 }` 检查整体删除。理由：Reset 从「仅供 dev 误调防护」升级为「生产用户主动重走引导」的正式功能（见 step 17 的 Settings CTA 投放生产）。如果守卫保留，前端 CTA 在生产下会失败，与 nudge dismissed 后的 regret 路径设计冲突
- **相关字段清理**：若 `env` 字段仅用于 Reset 的生产守卫，可以从 `Service` 结构体和 `NewService` 构造函数移除；同时更新 `EnvGuard` interface 的消费者（如果还有其他使用点则保留）
- service 返回的 Status 保持 completed/skipped 互斥，验证依赖 step02 的 SQL 原子性
- 测试覆盖：
  - `TestMarkSkipped_Idempotent`：连续两次调用时间戳相同
  - `TestMarkSkipped_ClearsCompleted`：已完成用户调 skip 后 completed_at 被清
  - `TestMarkCompleted_ClearsSkipped`：已跳过用户调 complete 后 skipped_at 被清（step02 SQL 保证）
  - `TestReset_ClearsBothColumns`：reset 后两列都是 NULL
  - `TestReset_AllowedInProduction`：生产环境调用 Reset 也成功（守卫已移除）
  - `TestGetStatus_ReflectsBothFields`：跳过后 GetStatus 返回 `Skipped: true, Completed: false`
  - **删除**现有的 `TestReset_ForbiddenInProduction` 测试（如果存在），因为守卫已移除

## 验证标准

1. `cd backend && go test ./internal/service/onboarding/...` 全部通过
2. `go vet ./...` 通过
3. 新增的 5 个测试用例覆盖所有状态转换路径
4. `GetStatus` 返回的 Status 对象字段齐全（completed + skipped 两对字段）

## 依赖说明

前置：step02（repo 层的 `MarkOnboardingSkipped` 和 `ResetOnboarding` 方法已存在）
