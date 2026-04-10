# Step 1: DB Migration

**依赖：** 无
**设计依据：** TRD §数据库 Schema

## 任务目标

创建两个新迁移文件，新增 `user_schedule_settings` 和 `holding_schedule_overrides` 表。

## 涉及文件

- 创建：`backend/migrations/016_user_schedule_settings.up.sql`
- 创建：`backend/migrations/016_user_schedule_settings.down.sql`
- 创建：`backend/migrations/017_holding_schedule_overrides.up.sql`
- 创建：`backend/migrations/017_holding_schedule_overrides.down.sql`

## 执行步骤

- [ ] 确认当前最高迁移序号：`ls backend/migrations/ | sort | tail -5`，确认 015 是最高序号，本次使用 016/017
- [ ] 创建 `016_user_schedule_settings.up.sql`，按 TRD §user_schedule_settings 建表
- [ ] 创建 `016_user_schedule_settings.down.sql`，`DROP TABLE IF EXISTS user_schedule_settings;`
- [ ] 创建 `017_holding_schedule_overrides.up.sql`，按 TRD §holding_schedule_overrides 建表
- [ ] 创建 `017_holding_schedule_overrides.down.sql`，`DROP TABLE IF EXISTS holding_schedule_overrides;`
- [ ] 在项目根执行 `docker-compose up -d` 确保 PostgreSQL 运行（如已运行跳过）
- [ ] 执行 `cd backend && make migrate-up` 验证迁移成功，无报错
- [ ] `git add backend/migrations/ && git commit -m "feat(db): add schedule settings and holding override tables"`

## 验证标准

- `make migrate-up` 无报错
- `psql` 可查到 `user_schedule_settings` 和 `holding_schedule_overrides` 两张表
- 两张表的字段与 TRD Schema 一致
