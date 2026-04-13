# Step 13: richman Config + Startup + DI + v1 Deprecation + Known Issues

> Phase 3 | 并行组 R7 (单独执行) | 前置: Steps 11, 12

## 任务目标

把前序 steps 创建的所有新组件串联到 richman 启动链路：config 新增 RichsonConfig/PlatformLLMConfig、.env.example 追加 v2 变量、启动检查（必需环境变量 + richson 连通性）、main.go DI 链路更新（新 repo/service/handler 注入）、v2 路由注册、shutdown 增加 cron.Stop()、v1 废弃标记。同时处理全部 richman 后端已知问题。

## 涉及文件

### 修改

- `backend/internal/config/config.go` -- 新增 RichsonConfig, PlatformLLMConfig
- `backend/.env.example` -- 追加 v2 变量模板
- `backend/cmd/server/main.go` -- DI 链路 + 路由注册 + startup check + shutdown
- `backend/internal/analysis/` -- 标记 deprecated 注释
- `backend/internal/llm/` -- 标记 deprecated（crypto.go 保留）
- `backend/internal/datasource/` -- 标记 deprecated
- `backend/internal/api/v1/` -- 废弃端点添加 Deprecation + Sunset 响应头

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| RichsonConfig struct | - | richman SS10.1 |
| PlatformLLMConfig struct | - | richman SS10.1 |
| .env.example 变量 | - | richman SS10.3 |
| 启动检查 (非空校验) | - | richman SS10.4 |
| 异步 richson 连通性检查 | - | richman SS10.4 |
| main.go DI 链路 | - | richman SS13.1 |
| 路由注册 (v1 + v2 并行) | - | richman SS13.2 |
| shutdown cron.Stop() | - | richman SS22.3 |
| v1 废弃标记 | - | richman SS12.1 |
| Deprecation/Sunset header | - | richman SS15.3 |

## 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G2.1 CORS 白名单 | CORS_ALLOWED_ORIGINS 环境变量 | richman SS22.1 |
| G2.2 健康检查端点 | GET /health + docker healthcheck | richman SS22.2 |
| G2.3 Cron 优雅关闭 | cronScheduler.Stop() + 等待 60s | richman SS22.3 |
| G2.4 Job 卡死恢复 | 清理间隔已在 Step 12 设为 10min | richman SS22.4 |
| G2.5 密码策略 | 最低 8 位 + 大小写 + 数字 | richman SS22.5 |
| G2.6 JWT 有效期 | 注释标明 Phase 2 refresh token | richman SS22.6 |
| G2.7 CSP 头部 | 中间件或 nginx 配置 | richman SS22.7 |
| G2.9 端点不一致 | 以 SS4.1 路由树为准 | richman SS22.9 |
| G2.11 备份策略 | 文档注释（不在代码实现） | richman SS22.11 |
| G2.12 嵌套事务 | 已在 Step 1 migration 处理 | richman SS22.12 |
| G2.13 plan 改名 | 已在 Step 1 migration 处理 | richman SS22.13 |
| G2.14 alert 去重 | 已在 Step 12 cron 处理 | richman SS22.14 |
| G2.15 用户级限流 | 已在 Step 11 handler 处理 | richman SS22.15 |

## 关键约束

- 启动时必须校验非空：RICHSON_BASE_URL, RICHSON_API_KEY, PLATFORM_LLM_API_KEY
- richson 连通性检查为异步（不阻塞 richman 启动），仅日志
- DI 链路新增组件顺序：repo -> service -> handler -> route
- v1 + v2 路由并行注册，v1 不移除
- v1 废弃端点使用标准 HTTP Deprecation header
- CORS_ALLOWED_ORIGINS 从环境变量读取，不硬编码
- 密码策略校验在 AuthService.Register 中增加
- shutdown 先 Stop cron，再 Wait 最多 60s，再关闭 DB pool
- config.go 使用现有模式（不直接 os.Getenv）

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] 缺少必需环境变量时启动失败并打印明确错误
- [ ] richson 不可用时 richman 仍能启动（日志 WARN）
- [ ] main.go 编译通过，所有 DI 注入无遗漏
- [ ] v1 废弃端点响应包含 Deprecation header
- [ ] CORS 白名单从环境变量读取
- [ ] GET /health 返回 richman + richson 状态
- [ ] shutdown 流程 cron.Stop() 被调用

## 变更点清单覆盖

D10.1-D10.8 (8), D12.1-D12.4 (4), G2.1-G2.7 (7), G2.9 (1), G2.11 (1) = **21 项**

注：G2.8/G2.10 在 Step 9, G2.12/G2.13 在 Step 1, G2.14/G2.15 在 Steps 11/12
