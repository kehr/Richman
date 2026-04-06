# Step 09 API DTO 对齐

## 任务目标

把所有现有和新增 API 的响应 DTO 对齐到本次重构的最终形态：
- 决策卡相关接口返回 recommendation 结构、badge_state、confidenceDelta、prevCardId
- 持仓相关接口返回 category 字段
- 凡涉及 *Pct 字段的 DTO 都通过 money.AttachAmounts 附加 *Amount 字段
- user_settings 接口完整暴露
- 推送消息渲染 DTO 不暴露金额字段

## 涉及文件

修改：
- `backend/internal/api/v1/decision_card.go`
- `backend/internal/api/v1/portfolio.go`
- `backend/internal/api/v1/auth.go`（如包含用户信息返回）
- `backend/internal/api/v1/notification.go`（确保推送 adapter 不传金额）
- `backend/internal/api/v1/asset_catalog.go`（如需按 category 过滤）

创建：
- `backend/internal/api/v1/user_settings.go`（GET / PATCH /api/v1/user/settings）
- `backend/internal/api/v1/user_settings_test.go`

修改：
- `backend/internal/notification/render/*.go`（所有 channel adapter 的 render 函数确认只接收 PublicCardSummary，不含金额字段）

## 设计依据

- TRD §2.5 API 响应字段
- TRD §5.3 金额换算在 API 层完成
- TRD §5.2 推送消息渲染层的隔离要求
- PRD §3 §4 §5 §6 §8

## 实施要点

- 每个返回决策卡的 handler 在最后一步调 money.AttachAmounts 注入金额字段
- 决策卡 DTO 反序列化 recommendation_json → Recommendation 结构后透传给前端，不做扁平化
- portfolio 列表 DTO 加 category（来自 holdings.category）
- user_settings handler 严格调 service.PatchUserSettings 做字段验证，不在 handler 里写业务规则
- 推送 adapter 的 render 函数签名审计：确保参数类型不含 totalCapital / amount，编译期就能拦住
- 现有 router 中 `/api/v1/notifications` 路径继续保留，但前端将通过 Settings tab 调用同样的接口（前端在 step18 处理路由变化）
- API 文档（如有 swagger 或 markdown 文档）同步更新

## 验证标准

1. `go test ./internal/api/...` 通过
2. 手动用 curl + 已登录 token 调以下接口确认字段齐全：
   - GET /api/v1/decision-cards/{id}
   - GET /api/v1/decision-cards?latest=true
   - GET /api/v1/holdings
   - GET /api/v1/user/settings
   - PATCH /api/v1/user/settings
3. 设置 / 清空 total_capital_cny 后，决策卡 DTO 的 *Amount 字段相应出现 / 消失
4. `make check` 通过
5. 无任何 *Amount 字段出现在 notification adapter 的输入参数

## 依赖说明

- 前置：step01-08 全部完成
- 是 backend 阶段的收尾

## 预估提交

- commit 1: `feat(api): expose new decision_card structured fields`
- commit 2: `feat(api): add user_settings endpoints with money attach`
- commit 3: `feat(api): add category to portfolio dto`
- commit 4: `chore(notification): audit render functions for capital leakage`
