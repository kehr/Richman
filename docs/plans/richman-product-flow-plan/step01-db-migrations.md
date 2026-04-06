# Step 01 数据库迁移

## 任务目标

新增 3 个数据库迁移文件，落地 TRD §1.3 §2.3 §5.1 §6.1 描述的所有新字段。包含 up / down 双向迁移，保证可回滚。

## 涉及文件

创建：
- `backend/db/migration/006_recommendation_structured.up.sql`
- `backend/db/migration/006_recommendation_structured.down.sql`
- `backend/db/migration/007_user_profile.up.sql`
- `backend/db/migration/007_user_profile.down.sql`
- `backend/db/migration/008_holding_category.up.sql`
- `backend/db/migration/008_holding_category.down.sql`

## 设计依据

- TRD §1.3 数据库迁移清单
- TRD §2.3 decision_cards 表 schema 变更（recommendation_json / action_level / target_position_ratio / badge_state / confidence_delta / prev_card_id / execution_fingerprint）
- TRD §3.3 执行计划指纹列
- TRD §5.1 users 表新增 total_capital_cny / onboarding_completed_at / risk_preference / categories
- PRD §8.2 总资金字段定义

## 实施要点

- 所有 ALTER TABLE 用 IF NOT EXISTS / IF EXISTS 保证幂等
- 索引按 TRD §2.3 列出（idx_dc_badge_state、idx_dc_prev）
- holdings 表 category 字段允许 NULL，不破坏现有数据
- down 迁移精确反向，不删除核心表
- risk_preference 字段加 CHECK 约束限定 enum 值（conservative / neutral / aggressive）

## 验证标准

1. `cd backend && make migrate-up` 成功，无报错
2. `make migrate-down` 回滚至 005，再 `make migrate-up` 重新应用，无残留字段或索引冲突
3. 现有种子数据和现有 API 不因新字段产生 panic（无 NOT NULL 缺省值缺失）
4. `make sqlc` 重新生成代码无报错（如果 sqlc 查询需要新字段，本 step 暂不修改 query 文件，留到后续 step）
5. lint：`make check` 通过

## 依赖说明

无前置依赖。这是整个 Plan 的起点。

## 预估提交

- 1 次 commit：`feat(db): add migrations 006-008 for product flow redesign`
