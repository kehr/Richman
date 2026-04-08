# Step 04 后端 API 层 POST /onboarding/skip

## 任务目标

在 `OnboardingHandler` 新增 `MarkSkipped` HTTP handler，对应 `POST /api/v1/onboarding/skip` 路由，返回更新后的 `onboarding.Status`。同步新增 API 集成测试覆盖完整请求-响应路径。

## 涉及文件

修改：
- `backend/internal/api/v1/onboarding.go`
- `backend/internal/api/v1/onboarding_test.go`

## 设计依据

- PRD §2.4 API 层 / §6.1 后端测试要求
- TRD §3.3 API 层（完整 handler 实现 + 路由注册位置）
- 既有 `POST /complete` handler 作为实现参考

## 实施要点

- 在 `RegisterRoutes` 注册 `group.POST("/skip", h.MarkSkipped)`
- `MarkSkipped` handler 结构对齐既有 `MarkCompleted`：取 `userID` → 调 service → 错误走 `handleServiceError` → 成功返回 `gin.H{"data": status}`
- Response body 遵循现有 `{"data": {...}}` 包裹风格
- HTTP 状态码用 `http.StatusOK`（不是 201，因为不是创建资源）
- 测试覆盖：
  - `TestOnboardingAPI_SkipEndpoint_Success`：POST /skip 返回 200 + `data.skipped=true`
  - `TestOnboardingAPI_SkipThenGet`：skip 后调 GET /onboarding 返回的 status 反映跳过状态
  - `TestOnboardingAPI_SkipThenCompleteClearSkipped`：skip 后 complete，验证 skipped_at 被清且 completed_at 有值
  - `TestOnboardingAPI_SkipRequiresAuth`：未带 auth 的请求返回 401

## 验证标准

1. `cd backend && go test ./internal/api/v1/... -run TestOnboardingAPI` 全部通过
2. `go build ./...` 无错误
3. `curl -X POST http://localhost:8080/api/v1/onboarding/skip -H "Authorization: Bearer <token>"` 在 dev 环境返回 200 + 包含 `data.skipped: true`
4. 后续再调 `GET /api/v1/onboarding` 返回的 status 同样反映跳过状态

## 依赖说明

前置：step03（`onboarding.Service.MarkSkipped` 已实现）
