# Step 08 Onboarding 服务与 API

## 任务目标

新增 onboarding service 跟踪用户的引导完成状态，提供"标记完成"和"重置"两个操作，配套 API 端点。前端的 OnboardingGuard（step10 实现）依赖此 API 判断是否需要把用户跳到引导流程。

## 涉及文件

创建：
- `backend/internal/service/onboarding/service.go`
- `backend/internal/service/onboarding/service_test.go`
- `backend/internal/api/v1/onboarding.go`
- `backend/internal/api/v1/onboarding_test.go`

修改：
- `backend/internal/api/v1/router.go`（注册新路由）
- `backend/cmd/server/main.go`（依赖注入）
- `backend/internal/repo/user_repo.go`（如尚未在 step05 暴露 onboarding_completed_at 的读写，在此补上）

## 设计依据

- TRD §6.1 Onboarding 状态跟踪
- PRD §2.3 Onboarding 4 步流程
- PRD §6.2 开发环境"重置 Onboarding"按钮

## 实施要点

- service 提供 3 个方法：
  - GetStatus(userID) → { completed: bool, completedAt: *time }
  - MarkCompleted(userID) → 写入 users.onboarding_completed_at = NOW()
  - Reset(userID) → 写 NULL，仅 dev/test 环境允许调用
- API 端点：
  - GET /api/v1/onboarding → 返回当前状态
  - POST /api/v1/onboarding/complete → 标记完成
  - DELETE /api/v1/onboarding（仅 dev 环境）→ 重置
- Reset 在生产环境直接 403，由 config.IsProduction() 判断
- 不需要新数据库表，全部基于 step01 已经在 users 表上加好的字段
- 单元测试覆盖：
  - 新用户 GetStatus 返回 completed = false
  - MarkCompleted 后 GetStatus 返回 completed = true
  - Reset 在 dev 模式生效，prod 模式 403
  - 不允许任意未认证调用（middleware 链覆盖）

## 验证标准

1. `go test ./internal/service/onboarding/... ./internal/api/v1/onboarding_test.go` 通过
2. mock 一个 dev 环境跑 Reset → MarkCompleted → GetStatus 完整流程通过
3. `make check` 通过

## 依赖说明

- 前置：step01（users.onboarding_completed_at）
- 可与 step05 / step07 并行

## 预估提交

- commit 1: `feat(onboarding): add status service with mark and reset`
- commit 2: `feat(api): expose onboarding endpoints with prod-mode reset guard`
