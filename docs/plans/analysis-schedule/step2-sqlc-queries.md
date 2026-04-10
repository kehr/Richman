# Step 2: sqlc Queries

**依赖：** Step 1（表已存在）
**设计依据：** TRD §后端层级结构

## 任务目标

为两张新表编写 sqlc SQL 查询文件并生成 Go 代码。

## 涉及文件

- 创建：`backend/internal/repo/schedule_queries.sql`
- 生成（执行 make sqlc）：`backend/internal/repo/schedule_queries.sql.go`（自动生成，不手写）

## 执行步骤

- [ ] 查看现有 repo 层的 SQL 文件格式参考：`ls backend/internal/repo/*.sql | head -3`，参照命名和注释风格
- [ ] 创建 `backend/internal/repo/schedule_queries.sql`，包含以下查询：
  - `GetUserScheduleSettings` — 按 user_id 查单条（含 is_deleted=false）
  - `UpsertUserScheduleSettings` — INSERT ... ON CONFLICT (user_id) DO UPDATE SET ... 含 updated_at=now()
  - `GetHoldingScheduleOverride` — 按 user_id + holding_id 查单条
  - `UpsertHoldingScheduleOverride` — INSERT ... ON CONFLICT (user_id, holding_id) DO UPDATE SET ...
  - `ListActiveUserScheduleSettings` — 查所有 is_deleted=false 的记录（调度器启动时用）
- [ ] 执行 `cd backend && make sqlc` 验证代码生成无报错
- [ ] 检查生成的 `.go` 文件确保函数签名与查询名对应
- [ ] `git add backend/internal/repo/ && git commit -m "feat(repo): add sqlc queries for schedule settings"`

## 验证标准

- `make sqlc` 无报错
- 生成的 Go 文件包含上述 5 个函数
- 每个函数参数与 SQL 查询参数一致
