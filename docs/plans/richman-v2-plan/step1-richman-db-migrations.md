# Step 1: richman DB Migrations + Seed Data

> Phase 1 | 并行组 R1 (可与 Step 2, 3 同时执行) | 无前置依赖

## 任务目标

创建 richman 三份数据库迁移脚本（021 表前缀重命名、022 v2 新表/新列、023 邀请系统）和更新 seed 数据。这是全部后续 step 的数据库基础。

## 涉及文件

### 创建

- `backend/db/migration/021_rename_tables_rm_prefix.up.sql`
- `backend/db/migration/021_rename_tables_rm_prefix.down.sql`
- `backend/db/migration/022_v2_user_feedback_and_columns.up.sql`
- `backend/db/migration/022_v2_user_feedback_and_columns.down.sql`
- `backend/db/migration/023_invite_system.up.sql`
- `backend/db/migration/023_invite_system.down.sql`

### 修改

- `backend/db/seed/asset_catalog.sql` -- 追加 PRD SS2.1 全部标的（A 股 / 美股 ETF 等，is_active=false）

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| 表前缀规则 | - | richson SS6.1, docs/standards/database.md |
| 14 张表重命名清单 | - | richson SS6.1 |
| rm_user_feedback 表 | SS6.3 用户反馈 | richman SS9.1.1 |
| risk_preference 枚举变更 | SS7.6 风险偏好 | richman SS9.2 |
| email_push_enabled 列 | SS10.2 退订 | richman SS7.4.1 |
| subscription_tier 列 | SS1.5 商业化 | richman SS9.2 + SS22.13 |
| disclaimer_accepted_at 列 | SS13 免责声明 | richman SS15.2.1 |
| decision_cards v2 新列 | SS8.1 执行计划 | richman SS9.2 |
| 邀请码表 + 奖励表 | SS14.3 邀请裂变 | invite SS3/SS9 |
| login_streak / last_login_date | SS14.3 连续登录 | invite SS3.3 |
| 标的分类体系 | SS2.1 分类框架 | richman SS9.3 |
| migration runner 不嵌套事务 | - | richman SS22.12 |
| subscription_tier 改名 | - | richman SS22.13 |

## 关键约束

- Migration 021: 14 条 ALTER TABLE RENAME 在同一文件中，runner.go 自动包裹事务，脚本内**不写** BEGIN/COMMIT（已知问题 SS22.12）
- Migration 022: `plan` 列改名为 `subscription_tier`（已知问题 SS22.13），risk_preference 需先 DROP 旧 CHECK 再改 DEFAULT 再 ADD 新 CHECK
- Migration 023: 邀请码表的 UNIQUE INDEX 和 RESTART WITH 100000
- Seed 数据只追加新标的记录，不修改已有记录

## 验证标准

- [ ] `make migrate-up` 从干净库（020 状态）成功执行到 023
- [ ] `make migrate-down` 三次回滚到 020 状态，数据库结构恢复
- [ ] 021 执行后 14 张表全部带 rm_ 前缀，\dt 确认
- [ ] 022 执行后 rm_user_feedback 表存在，rm_users 有 5 个新列，rm_decision_cards 有 10 个新列
- [ ] 023 执行后 rm_user_invite_codes 和 rm_invite_rewards 表存在
- [ ] seed 数据包含 PRD SS2.1 所有标的
- [ ] `cd backend && make check` 通过

## 变更点清单覆盖

A1.1-A1.15 (15), A2.1-A2.17 (17), A3.1-A3.5 (5), A4.1 (1) = **38 项**

附带处理已知问题: G2.12 (嵌套事务), G2.13 (plan 改名)
