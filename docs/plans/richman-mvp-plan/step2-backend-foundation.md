# Step 2: Backend Foundation

## 任务目标

搭建 Go 后端骨架：数据库 schema + 迁移、认证系统（邮箱注册/登录 + 邀请码）、配置管理、日志系统（zap）、全局错误处理中间件、CORS 中间件。

## 涉及文件路径

### 创建

- `backend/internal/config/config.go` -- 集中式配置管理
- `backend/internal/logger/logger.go` -- zap Logger 初始化（多 core、轮转、采样）
- `backend/internal/logger/mask.go` -- 敏感数据脱敏
- `backend/internal/model/user.go` -- 用户模型
- `backend/internal/model/plan.go` -- 订阅计划模型
- `backend/internal/model/invite_code.go` -- 邀请码模型
- `backend/internal/api/middleware/auth.go` -- JWT 认证中间件
- `backend/internal/api/middleware/plan_check.go` -- Plan 权限检查中间件
- `backend/internal/api/middleware/request_id.go` -- Request ID 中间件
- `backend/internal/api/middleware/cors.go` -- CORS 中间件
- `backend/internal/api/middleware/error_handler.go` -- 全局错误处理
- `backend/internal/api/middleware/access_log.go` -- HTTP 访问日志
- `backend/internal/api/v1/auth.go` -- 认证路由处理器
- `backend/internal/service/auth/service.go` -- 认证业务逻辑
- `backend/internal/repo/` -- sqlc 生成的数据访问代码
- `backend/db/migration/001_init_schema.up.sql` -- 初始 schema（users, plans, invite_codes）
- `backend/db/migration/001_init_schema.down.sql` -- 回滚
- `backend/db/query/user.sql` -- 用户查询
- `backend/db/query/plan.sql` -- 计划查询
- `backend/db/query/invite_code.sql` -- 邀请码查询
- `backend/db/sqlc.yaml` -- sqlc 配置

### 修改

- `backend/cmd/server/main.go` -- 接入配置、日志、路由、中间件

## PRD/TRD 章节引用

- PRD 4.1 账户系统
- PRD 4.2 权限模型
- PRD 5.3 后端技术栈
- PRD 5.3.1 后端四层架构
- `docs/standards/backend.md` 三层架构
- `docs/standards/database.md` schema 约定、审计字段
- `docs/standards/api.md` 认证端点、错误格式
- `docs/standards/logging.md` 日志系统规范

## 验证标准

- [ ] `sqlc generate` 成功生成 Go 代码
- [ ] 数据库迁移 up/down 均可执行
- [ ] `POST /api/v1/auth/register` 带有效邀请码可注册成功
- [ ] `POST /api/v1/auth/register` 无邀请码返回 400
- [ ] `POST /api/v1/auth/login` 返回 JWT token
- [ ] 带 JWT 的请求能通过 auth 中间件
- [ ] 无 JWT 请求返回 401
- [ ] `GET /api/v1/auth/me` 返回当前用户信息
- [ ] Request ID 在响应头 X-Request-ID 中返回
- [ ] 日志输出包含 requestId 字段
- [ ] 全局错误处理返回统一 JSON 格式
- [ ] `go test ./internal/service/auth/...` 通过
- [ ] `go test ./internal/...` 全部通过

## 依赖说明

- Step 1 完成（Go 项目骨架、Docker PostgreSQL 就绪）
