# Step 05 user_settings 服务与隐私守卫

## 任务目标

新增 user_settings 服务，承载总资金、风险偏好、关注类型、onboarding 状态等用户级配置的读写。同时实现 TRD §5.2 §5.3 要求的 3 项编译期约束 + 1 项运行时守卫，以及金额换算工具函数。

## 涉及文件

创建：
- `backend/internal/service/user_settings/service.go`
- `backend/internal/service/user_settings/service_test.go`
- `backend/internal/service/user_settings/money.go`（金额附加工具）
- `backend/internal/service/user_settings/money_test.go`
- `backend/internal/service/user_settings/privacy_guard.go`（运行时守卫）
- `backend/internal/service/user_settings/privacy_guard_test.go`

修改：
- `backend/internal/repo/user_repo.go`（新增字段读写）
- `backend/db/query/user.sql`（如使用 sqlc）
- `backend/internal/model/user.go`（如有 model 层）

## 设计依据

- TRD §5.1 字段定义
- TRD §5.2 访问路径隔离的 3 编译期约束 + 运行时守卫
- TRD §5.3 金额换算在 API 层完成
- PRD §6.2 Tab 1 账户字段
- PRD §8 总资金功能完整规格

## 实施要点

- service 提供 GetUserSettings(userID) 和 PatchUserSettings(userID, patch) 两个核心方法
- patch 是稀疏对象，未传字段不更新
- 字段验证：
  - total_capital_cny：>= 0 或 null（清空）
  - risk_preference：枚举 conservative / neutral / aggressive
  - categories：JSON 数组，每项必须是 PRD §1.5 定义的 4 类之一
- money.AttachAmounts 是通用工具：接受任意 DTO + total_capital，反射检查所有 `*Pct` 字段并附加同名 `*Amount` 字段
- privacy_guard.AssertNoCapitalLeakage 通过反射检查输入的 struct 不含 totalCapital / positionAmount / targetPositionAmount / unrealizedAmount / realizedAmount 等具体敏感字段（精确列表避免 paymentAmount 等良性字段误报）
- 守卫在所有构建中启用（反射成本可忽略，不使用 build tag）。handler 层在构造分析请求 / LLM 上下文 / 推送渲染 DTO 前显式调用
- 单元测试覆盖：
  - GetUserSettings 默认值
  - PatchUserSettings 各字段单独/组合更新
  - money.AttachAmounts 在总资金为 nil 时不修改 DTO
  - money.AttachAmounts 在总资金已设时正确附加 amount 字段
  - privacy_guard 检测含敏感字段的 struct 报错

## 验证标准

1. `go test ./internal/service/user_settings/...` 通过
2. 测试覆盖率 >= 90%
3. `go build ./...` 通过（不需要额外 build tag）
4. `make check` 通过

## 依赖说明

- 前置：step01（users 表新字段）

## 预估提交

- commit 1: `feat(user_settings): add service for capital, risk preference, categories`
- commit 2: `feat(user_settings): add money attach utility for api dto enrichment`
- commit 3: `feat(user_settings): add runtime privacy guard for total_capital`
