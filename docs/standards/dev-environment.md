## 目标

本规范覆盖开发环境的启动顺序、pull 后的必做动作、以及运行期失效模式的排查清单。所有约束都有机械层面的防御支持，文档只是约束的记录和排查时的参考。

## 适用范围

- 本地 dev 环境：每次 clone、pull、切分支后启动后端
- 追查运行期异常：尤其是 500 / 内部错误 / 认证失败等症状
- 新成员入职：理解项目的启动顺序和失效模式

生产环境由 CI/CD 管线统一处理 migration，不经过本规范。

## 启动顺序（强制）

### 首次 clone

1. `cp frontend/.env.example frontend/.env` 并按需填写
2. `cp backend/.env.example backend/.env` 并按需填写
3. `docker-compose up -d` 启动本地 PostgreSQL（5433 端口）
4. `cd backend && make migrate-up` 初始化数据库 schema
5. `cd backend && make dev` 启动后端（make dev 已经依赖 migrate-up，这一步会重复跑一次但幂等）
6. `cd frontend && pnpm install` 安装前端依赖
7. `cd frontend && pnpm dev` 启动前端

### 每次 pull / 切分支

```
git pull            # 或 git checkout <branch>
cd backend && make dev
cd frontend && pnpm dev
```

**关键约定**：`make dev` 的 target 已经声明为 `dev: migrate-up`，所以每次启动后端会自动先跑 migration 再起服务。手动绕过（例如直接 `go run ./cmd/server/main.go`）不推荐，但也会被 server 启动时的 schema drift 检查拦下。

### 运行 migration 的触发条件

满足以下任一条件必须运行 `make migrate-up`：

- pull 了包含 `backend/db/migration/*.sql` 新文件的代码
- 切到一个 schema 比当前数据库新的分支
- 清空或重建了本地 postgres 容器
- 启动后端时看到 `schema drift detected at startup: pending versions: [...]` 致命日志

幂等保证：`make migrate-up` 通过 `schema_migrations` 表跳过已 applied 的版本，多次运行无副作用。

## Schema drift 防御机制

项目对"代码期望新 schema 但数据库还在老 schema"这类漂移有三层防御：

| 层级 | 位置 | 触发时机 | 行为 |
|---|---|---|---|
| 1. Makefile 依赖 | `backend/Makefile` `dev: migrate-up` | 每次 `make dev` 启动时 | 先跑 migration 再起服务，自动修复 |
| 2. 启动检查 | `backend/internal/migration/verify.go` VerifyCurrent | main.go 连 DB 后、init service 前 | 扫描 `db/migration/` 目录和 `schema_migrations` 表，差异时 Fatal 退出 |
| 3. 文档与 memory | 本文件 + 项目 memory feedback | 人工触发 | 排查时提供定位思路 |

**设计原则**：
- dev 走 Makefile 依赖（自动修复）
- prod 走启动检查（快速失败）
- 不依赖开发者记忆或文档

## 常见失效模式与排查清单

### 症状 A：login 返回 500 `internal server error`，access log 只有 INFO 级别

**最常见根因**：**schema drift**。代码里的 SELECT 查询引用了数据库没有的列。

排查顺序：

1. 看后端启动日志。如果有 `schema drift detected at startup: pending versions: [N]`，直接跑 `make migrate-up`
2. 如果启动时没有 drift 日志但运行期报 500，看后端 ERROR 日志里是否有 `unhandled service error`，error 字段会显示完整的 wrapped 错误链（例如 `find user: column "xxx" does not exist`）
3. 如果代码是刚拉下来的新代码但没跑过 migration，直接 `cd backend && make migrate-up`
4. 验证 `docker exec -i richman-postgres psql -U richman -d richman -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1"` 的最大版本是否与 `ls backend/db/migration/*.up.sql | tail -1` 一致

### 症状 B：`make dev` 报 `connect database` 失败

**根因**：postgres 容器没起。

排查：

1. `docker ps --filter "name=postgres"` 确认容器在 Running
2. 如果没在，跑 `docker-compose up -d`
3. 确认 `backend/.env` 里的 `DATABASE_URL` 端口和 docker-compose.yml 一致（默认 5433）

### 症状 C：frontend 调用 `/api/v1/auth/me` 一直 401

**可能根因**：JWT secret 在 `.env` 里被改过、或者 cookie domain 不匹配。这不是 schema drift，属于 auth 配置问题，单独排查。

## 日志可观测性约定

**硬约束**：任何 500 响应必须在后端日志里留下 ERROR 级别的调用链。具体机制：

- 所有 handler 的错误出口是 `backend/internal/api/v1/auth.go` 的 `handleServiceError`
- `AppError`（预期业务错误）走原路径，不打 ERROR 日志
- 非 `AppError`（意外错误）在 `handleServiceError` 内部调用 `logger.Error("unhandled service error", ...)`，字段包含 `requestId`、`path`、`method`、`error`（完整 wrapped 链）
- 绝对不允许 handler 自己处理 `err` 并直接 `c.JSON(500, ...)` 而不经过 handleServiceError——会绕过日志

如果看到 500 但 ERROR 日志里没对应记录，说明有 handler 绕开了约定，属于 bug，必须修复。

## 生产 vs 开发的差异

| 维度 | dev | prod |
|---|---|---|
| migration 触发 | `make dev` 自动，或手动 `make migrate-up` | CI/CD 部署前统一执行 |
| Schema drift 行为 | Fatal 退出，控制台显示 remediation | Fatal 退出，监控立刻告警 |
| ERROR 日志输出 | Console 彩色文本 | JSON 结构化到 stdout + 远程采集 |
| 连接池配置 | 小（2-5） | 大（按 `cfg.DB.MaxConns`） |

## 不允许的捷径

- 不允许为了绕过 schema drift 直接改 `schema_migrations` 表塞一行假数据
- 不允许在 handler 层直接写 `c.JSON(500, ...)` 绕开 `handleServiceError`
- 不允许 `make dev` 时加 `-k` 或其他标志忽略 migrate-up 失败

这些动作都会把失败从"启动时快速失败"变成"运行时神秘 500"，显著增加排查成本。
