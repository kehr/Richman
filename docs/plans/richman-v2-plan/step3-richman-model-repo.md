# Step 3: richman Model + Repo Layer Updates

> Phase 1 | 并行组 R1 (可与 Step 1, 2 同时执行) | 无前置依赖

## 任务目标

更新 richman 全部现有 repo 中的 SQL 表名为 rm_ 前缀，新增 v2 所需的 Go model structs（映射 rs_* 和 rm_* 新表），新增 v2 repo 层（rs_* 只读 repo + rm_user_feedback repo + invite repos），扩展现有 repo 新增方法。

## 涉及文件

### 创建

- `backend/internal/model/asset_analysis.go` -- AssetAnalysis, AnalysisDimension
- `backend/internal/model/analysis_job.go` -- AnalysisJob
- `backend/internal/model/event_alert.go` -- EventAlert
- `backend/internal/model/user_feedback.go` -- UserFeedback
- `backend/internal/model/invite.go` -- UserInviteCode, InviteReward, InvitedUser
- `backend/internal/repo/asset_analysis_read_repo.go` -- rs_asset_analyses 只读
- `backend/internal/repo/analysis_dimension_read_repo.go` -- rs_asset_analysis_dimensions 只读
- `backend/internal/repo/analysis_job_read_repo.go` -- rs_analysis_jobs 只读
- `backend/internal/repo/event_alert_read_repo.go` -- rs_event_alerts 读 + alerted 更新
- `backend/internal/repo/user_feedback_repo.go` -- rm_user_feedback CRUD
- `backend/internal/repo/user_invite_code_repo.go` -- rm_user_invite_codes CRUD
- `backend/internal/repo/invite_reward_repo.go` -- rm_invite_rewards CRUD

### 修改

- `backend/internal/repo/user_repo.go` -- SQL 表名 users->rm_users + 新增 UpdateRiskPreference, UpdateEmailPush
- `backend/internal/repo/holding_repo.go` -- SQL 表名 holdings->rm_holdings + 新增 GetExposureByAssetType
- `backend/internal/repo/asset_repo.go` -- SQL 表名 asset_catalog->rm_asset_catalog + 新增 ListActiveWithType
- `backend/internal/repo/decision_card_repo.go` -- SQL 表名 decision_cards->rm_decision_cards + 新增 GetLatestByHoldings
- `backend/internal/repo/analysis_result_repo.go` -- SQL 表名 analysis_results->rm_analysis_results
- `backend/internal/repo/trade_repo.go` -- SQL 表名 trades->rm_trades
- `backend/internal/repo/plan_repo.go` -- SQL 表名 plans->rm_plans
- `backend/internal/repo/invite_repo.go` -- SQL 表名 invite_codes->rm_invite_codes
- `backend/internal/repo/notification_channel_repo.go` -- SQL 表名
- `backend/internal/repo/notification_log_repo.go` -- SQL 表名
- `backend/internal/repo/task_repo.go` -- SQL 表名
- `backend/internal/repo/llm_config_repo.go` -- SQL 表名
- `backend/internal/repo/schedule_repo.go` -- SQL 表名
- `backend/internal/model/` 下已有 model 文件 -- 如有表名硬编码需更新

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| rm_ 前缀 SQL 更新 | - | richman SS6.3 |
| AssetAnalysisReadRepo | - | richman SS6.1 |
| AnalysisDimensionReadRepo | - | richman SS6.1 |
| AnalysisJobReadRepo | - | richman SS6.1 |
| EventAlertReadRepo + MarkAlerted | - | richman SS6.1 |
| UserFeedbackRepo | SS6.3 | richman SS6.2 |
| 现有 repo 新增方法 | - | richman SS6.3 |
| Go model structs | - | richman SS11 |
| invite 相关 model/repo | SS14.3 | invite SS10 |
| rs_* 只读权限 | - | richson SS6.4, richman SS6.1 |
| 跨服务写入例外 (event_alerts.alerted) | - | richman SS6.1, SS8.5 |

## 关键约束

- 全部现有 repo SQL 中表名替换必须完整，grep 确认无遗漏
- rs_* repo 为**只读**（SELECT 权限），除两个例外：event_alerts.alerted 更新和 analysis_jobs 过期清理
- 新增 repo 沿用现有模式：struct + NewXxxRepo(pool) + 方法
- Model struct 字段与 TRD 定义的表列严格对齐
- 被邀请人姓名脱敏：InvitedUser struct 需提供脱敏展示方法

## 验证标准

- [ ] `cd backend && make check` 通过（lint + test + build）
- [ ] grep 全部 repo .go 文件，无任何无前缀的旧表名引用
- [ ] 新增的 12 个 repo 文件编译通过
- [ ] 新增的 7 个 model 文件编译通过
- [ ] 现有 repo_test 文件（如 llm_config_repo_test.go）仍然通过

## 变更点清单覆盖

D5.1-D5.11 (11), D6.1-D6.6 (6), D7.1-D7.8 (8) = **25 项**
