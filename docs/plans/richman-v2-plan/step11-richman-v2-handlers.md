# Step 11: richman v2 Handlers + Routing + Error Codes

> Phase 3 | 并行组 R6 (可与 Step 12 同时执行) | 前置: Steps 8, 9, 10

## 任务目标

实现全部 v2 API handler 层（代理 handler + 聚合 handler）、v2 路由注册（/api/v2/ 前缀分组）、IP 限流中间件、v2 错误码定义，以及 v1 认证端点限流和账户注销端点。

## 涉及文件

### 创建

- `backend/internal/api/v2/market.go` -- MarketHandler (7 端点)
- `backend/internal/api/v2/event.go` -- EventHandler (1 端点)
- `backend/internal/api/v2/analysis.go` -- AnalysisHandler (3 端点)
- `backend/internal/api/v2/briefing.go` -- BriefingHandler (1 端点)
- `backend/internal/api/v2/feedback.go` -- FeedbackHandler (1 端点)
- `backend/internal/api/v2/user.go` -- UserHandler (2 端点)
- `backend/internal/api/v2/invite.go` -- InviteHandler (2 端点)
- `backend/internal/api/v2/middleware/ratelimit.go` -- IP 限流中间件

### 修改

- `backend/internal/api/v1/` -- auth 端点增加 IP 限流 + 新增 DELETE /auth/account

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| v2 路由分组 | - | richman SS4.1 |
| 代理 handler (透传 richson) | - | richman SS4.2 |
| 聚合 handler (多数据源组装) | - | richman SS4.2 |
| Gin 路由参数冲突 (:code vs regime/overview) | - | richman SS4.1 |
| IP 限流 (60 次/分钟) | - | richman SS4.3 |
| v1 认证限流 (5 次/分钟) | - | richman SS4.1 |
| handler 注册模式 (RegisterRoutes) | - | richman SS4.4 |
| v2 错误码 (8 个) | - | richman SS14.1 |
| 账户注销 (DELETE /auth/account) | - | richman SS21.1 |
| market/:code/share 端点 (JWT 可选) | SS14.3 | invite SS6.1 |
| demo-plan 数据源 (DB 主 + richson fallback) | SS5.2.4 | richman SS4.2 |
| 用户级限流 (认证端点) | - | richman SS22.15 |

## 关键约束

- Gin 路由注册顺序：`/regime` 和 `/overview` 必须在 `/:code` 之前注册（Gin 参数冲突 SS4.1）
- 代理 handler 不解析 richson 响应体（除错误检测外），直接透传
- analysis/trigger-asset 需注入平台 llmConfig 后转发
- demo-plan 端点从 `rs_asset_analyses.demo_plan` 读取，null 时 fallback 到 richson
- share 端点 JWT 可选（已登录附带邀请码，未登录不附带）
- 限流实现：进程内 map + 滑动窗口计数（MVP 单实例，不引入 Redis）
- IP 获取使用 Gin c.ClientIP()，需配置 SetTrustedProxies
- 错误码定义 8 个 v2 专用错误码（SS14.1）

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] v2 路由注册完整（grep router 注册代码确认 20 个端点）
- [ ] 代理端点能透传 richson 响应（需 mock richson 或集成测试）
- [ ] IP 限流超过阈值返回 429
- [ ] v1 auth 端点限流生效
- [ ] DELETE /auth/account 需密码确认
- [ ] market/:code/share JWT 可选逻辑正确
- [ ] regime/overview 路由不与 :code 冲突（请求 /market/regime 返回体制而非 404）

## 变更点清单覆盖

D2.1-D2.20 (20), D11.1-D11.8 (8), G2.15 (1) = **29 项**
