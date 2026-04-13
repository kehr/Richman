# Step 7: richman richson HTTP Client

> Phase 3 | 并行组 R4 (单独执行) | 前置: Steps 1, 3, 6

## 任务目标

实现 richman 中调用 richson 的完整 HTTP 客户端封装：Client struct、全部 11 个方法、超时/重试策略、X-Request-ID 注入、错误码映射、健康检查集成（atomic.Bool）。这是 richman 全部 v2 业务逻辑的基础通信层。

## 涉及文件

### 创建

- `backend/internal/richson/client.go` -- Client struct + NewClient + 11 个方法
- `backend/internal/richson/types.go` -- 全部 request/response Go 类型定义

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| Client 结构 | - | richman SS3.1 |
| 超时/重试策略 | - | richman SS3.2 |
| X-Request-ID 注入 | - | richman SS3.3 |
| 错误码映射 (richsonErrorMap) | - | richman SS3.4 |
| 方法清单 (11 个) | - | richman SS3.5 |
| 健康检查 + IsHealthy() | - | richman SS3.6 |
| richson API 契约 | SS3.1 | richson SS5.1-SS5.4 |

## 关键约束

- 三级超时：异步触发 5s, 同步分析 30s, 轻量查询 10s, 健康检查 3s
- 重试仅在网络错误或 502/503 时执行 1 次，间隔 2s
- 重试复用原 context（继承取消信号）
- IsHealthy() 使用 `atomic.Bool`，cron 每 30s 更新
- types.go 中 response 类型须与 richson API 响应体严格对齐（richson SS5.1-SS5.4）
- richsonErrorMap 映射 richson 错误码到 HTTP 状态码
- 每个请求自动注入 Authorization header 和 X-Request-ID

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] Client 可实例化（NewClient 传入 mock config）
- [ ] 全部 11 个方法存在且签名正确
- [ ] types.go 中类型可正常 JSON 序列化/反序列化
- [ ] IsHealthy() 初始返回 false，HealthCheck 成功后返回 true

## 变更点清单覆盖

D1.1-D1.16 = **16 项**
