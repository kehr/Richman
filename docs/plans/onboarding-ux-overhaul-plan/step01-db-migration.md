# Step 01 数据库迁移 onboarding_skipped_at

## 任务目标

为 `users` 表新增 `onboarding_skipped_at` 字段，用于标记用户主动跳过引导的时间戳。字段与现有 `onboarding_completed_at` 语义互斥，互斥约束在后续 step 的 SQL 写入路径中保证。

## 涉及文件

创建：
- `backend/db/migration/010_onboarding_skipped.up.sql`
- `backend/db/migration/010_onboarding_skipped.down.sql`

## 设计依据

- PRD §1.1 schema 迁移
- PRD §1.2 互斥契约（本 step 只添加字段，互斥由后续 SQL 保证）
- 既有 migration 编号到 009，新 migration 用 010

## 实施要点

- up 迁移用 `ADD COLUMN IF NOT EXISTS`，保证幂等
- 字段类型 `TIMESTAMPTZ NULL`，与 `onboarding_completed_at` 对齐
- down 迁移用 `DROP COLUMN IF EXISTS`
- 无数据回填：既有用户该字段默认 NULL，语义正确（未跳过 = NULL）
- 迁移执行通过 `cd backend && make migrate-up` 触发，按文件名排序自动识别

## 验证标准

1. `cd backend && make migrate-up` 执行无错误
2. 手动检查数据库：`\d users` 应显示新列 `onboarding_skipped_at timestamptz nullable`
3. `make migrate-down` 可正确回滚（确认可逆）
4. 执行完后再次 `make migrate-up` 仍然成功（幂等）
5. 既有测试 `go test ./internal/service/onboarding/...` 不因 schema 变化挂掉

## 依赖说明

无前置依赖。这是整个 plan 的第一个 step。
