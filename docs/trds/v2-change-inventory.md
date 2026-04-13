# v2 变更点清单（Change Inventory）

> 用途：plan 覆盖率校验基准。每个条目必须映射到至少一个 plan step。编码阶段每完成一项勾选。
>
> 来源：richson-service-trd.md / richman-backend-v2-trd.md / frontend-v2-trd.md / invite-system-trd.md

## A. 数据库层 -- richman 迁移（Go migration runner）

### A1. Migration 021: 存量表 rm_ 前缀重命名

| # | 变更 | TRD 引用 |
|---|------|----------|
| A1.1 | ALTER TABLE users RENAME TO rm_users | richson SS6.1 |
| A1.2 | ALTER TABLE plans RENAME TO rm_plans | richson SS6.1 |
| A1.3 | ALTER TABLE invite_codes RENAME TO rm_invite_codes | richson SS6.1 |
| A1.4 | ALTER TABLE asset_catalog RENAME TO rm_asset_catalog | richson SS6.1 |
| A1.5 | ALTER TABLE holdings RENAME TO rm_holdings | richson SS6.1 |
| A1.6 | ALTER TABLE trades RENAME TO rm_trades | richson SS6.1 |
| A1.7 | ALTER TABLE analysis_results RENAME TO rm_analysis_results | richson SS6.1 |
| A1.8 | ALTER TABLE decision_cards RENAME TO rm_decision_cards | richson SS6.1 |
| A1.9 | ALTER TABLE notification_channels RENAME TO rm_notification_channels | richson SS6.1 |
| A1.10 | ALTER TABLE notification_logs RENAME TO rm_notification_logs | richson SS6.1 |
| A1.11 | ALTER TABLE analysis_tasks RENAME TO rm_analysis_tasks | richson SS6.1 |
| A1.12 | ALTER TABLE llm_configs RENAME TO rm_llm_configs | richson SS6.1 |
| A1.13 | ALTER TABLE user_schedule_settings RENAME TO rm_user_schedule_settings | richson SS6.1 |
| A1.14 | ALTER TABLE holding_schedule_overrides RENAME TO rm_holding_schedule_overrides | richson SS6.1 |
| A1.15 | 对应 down 脚本（反向 RENAME） | richson SS6.1 |

### A2. Migration 022: 新表 + 新列

| # | 变更 | TRD 引用 |
|---|------|----------|
| A2.1 | CREATE TABLE rm_user_feedback（含 idx + RESTART WITH） | richman SS9.1.1 |
| A2.2 | rm_users: DROP 旧 CHECK + SET DEFAULT 'moderate' + UPDATE neutral->moderate + ADD 新 CHECK (conservative/moderate/aggressive) | richman SS9.2 |
| A2.3 | rm_users ADD COLUMN email_push_enabled BOOLEAN DEFAULT TRUE | richman SS7.4.1 |
| A2.4 | rm_users ADD COLUMN subscription_tier VARCHAR(16) DEFAULT 'invite'（原 plan，已知问题 22.13 改名） | richman SS9.2 + SS22.13 |
| A2.5 | rm_users ADD COLUMN disclaimer_accepted_at TIMESTAMPTZ | richman SS15.2.1 |
| A2.6 | rm_decision_cards ADD COLUMN action VARCHAR(32) | richman SS9.2 |
| A2.7 | rm_decision_cards ADD COLUMN action_label VARCHAR(128) | richman SS9.2 |
| A2.8 | rm_decision_cards ADD COLUMN scenarios JSONB | richman SS9.2 |
| A2.9 | rm_decision_cards ADD COLUMN stop_loss DECIMAL(20,6) | richman SS9.2 |
| A2.10 | rm_decision_cards ADD COLUMN take_profit DECIMAL(20,6) | richman SS9.2 |
| A2.11 | rm_decision_cards ADD COLUMN valid_days INT | richman SS9.2 |
| A2.12 | rm_decision_cards ADD COLUMN concentration_level VARCHAR(16) | richman SS9.2 |
| A2.13 | rm_decision_cards ADD COLUMN concentration_message TEXT | richman SS9.2 |
| A2.14 | rm_decision_cards ADD COLUMN default_action TEXT | richman SS9.2 |
| A2.15 | rm_decision_cards ADD COLUMN no_trigger_note TEXT | richman SS9.2 |
| A2.16 | rm_decision_cards ADD COLUMN model_version VARCHAR(32) | richman SS9.2 |
| A2.17 | 对应 down 脚本 | richman SS9.2 |

### A3. Migration 023: 邀请系统

| # | 变更 | TRD 引用 |
|---|------|----------|
| A3.1 | CREATE TABLE rm_user_invite_codes（含 idx + uq + RESTART WITH） | invite SS3.1/SS9 |
| A3.2 | CREATE TABLE rm_invite_rewards（含 idx + RESTART WITH） | invite SS3.2/SS9 |
| A3.3 | rm_users ADD COLUMN login_streak INT DEFAULT 0 | invite SS3.3 |
| A3.4 | rm_users ADD COLUMN last_login_date DATE | invite SS3.3 |
| A3.5 | 对应 down 脚本 | invite SS9 |

### A4. Seed 数据

| # | 变更 | TRD 引用 |
|---|------|----------|
| A4.1 | 更新 seed/asset_catalog.sql 追加 PRD SS2.1 全部标的（A 股/美股 ETF 等，is_active=false） | richman SS9.3 |

## B. 数据库层 -- richson Alembic 迁移

| # | 变更 | TRD 引用 |
|---|------|----------|
| B1 | CREATE TABLE rs_asset_analyses（含 2 索引 + RESTART WITH） | richson SS6.2 |
| B2 | CREATE TABLE rs_asset_analysis_dimensions（含索引） | richson SS6.2 |
| B3 | CREATE TABLE rs_analysis_jobs（含 partial unique idx + 2 索引） | richson SS6.2 |
| B4 | CREATE TABLE rs_event_alerts（含 partial unique idx + 索引 + RESTART WITH，已知问题 21.4） | richson SS6.2 |
| B5 | CREATE TABLE rs_dimension_definitions（含 unique idx） | richson SS6.2 |
| B6 | Seed: gold 四维权重数据（4 行 rs_dimension_definitions，model_version=gold_v1.0） | richson SS6.2 |

## C. richson 服务（Python，全新创建）

### C1. 项目脚手架

| # | 变更 | TRD 引用 |
|---|------|----------|
| C1.1 | pyproject.toml（依赖声明 + 元数据） | richson SS3.2 |
| C1.2 | Dockerfile | richson SS12.1 |
| C1.3 | .env.example | richson SS12.2 |
| C1.4 | alembic.ini + alembic/versions/ | richson SS3.1 |
| C1.5 | main.py（FastAPI app 入口） | richson SS3.1 |
| C1.6 | config.py（pydantic-settings） | richson SS3.1 |

### C2. API 端点

| # | 端点 | 交互模式 | TRD 引用 |
|---|------|----------|----------|
| C2.1 | GET /health | C | richson SS5.4 |
| C2.2 | POST /jobs/analyze-asset | A 异步 | richson SS5.1 |
| C2.3 | POST /jobs/batch-analyze | A 异步 | richson SS5.1 |
| C2.4 | GET /jobs/{jobId} | C | richson SS5.1 |
| C2.5 | POST /analyze/holding | B 同步 | richson SS5.2 |
| C2.6 | POST /analyze/demo-plan | C | richson SS5.3 |
| C2.7 | GET /market/regime | C | richson SS5.3 |
| C2.8 | GET /market/ohlcv/{assetCode} | C | richson SS5.3 |
| C2.9 | GET /assets/{assetCode}/score-history | C | richson SS5.3 |
| C2.10 | GET /events/radar | C | richson SS5.3 |
| C2.11 | POST /content/weekly-insight | B 同步 | richson SS9.3 |

### C3. 量化评分引擎（Layer 1）

| # | 模块 | TRD 引用 |
|---|------|----------|
| C3.1 | pipeline.py（L1->L2->L3 编排） | richson SS7.1 |
| C3.2 | scoring.py（维度评分 + 百分位） | richson SS7.2 |
| C3.3 | adjustment.py（LLM 定性->数值映射） | richson SS7.4 |
| C3.4 | confidence.py（综合置信度计算） | richson SS7.5 |
| C3.5 | d1_macro_rates.py（宏观利率维度） | richson SS3.1 |
| C3.6 | d2_dollar_liquidity.py（美元流动性维度） | richson SS3.1 |
| C3.7 | d3_structural_demand.py（结构性需求维度） | richson SS3.1 |
| C3.8 | d4_technical_position.py（技术位置维度） | richson SS3.1 |
| C3.9 | support_resistance.py（支撑/阻力位） | richson SS7.6 |
| C3.10 | regime.py（市场体制检测） | richson SS5.3 |
| C3.11 | event_monitor.py（Polymarket 事件概率监控） | richson SS6.2 |
| C3.12 | compute_drawdown_reference（历史回撤） | richson SS7.7 |
| C3.13 | detect_conflict（冲突检测） | richson SS7.8 |
| C3.14 | D4 ATR 动态权重调制 | richson SS7.3 |

### C4. Google ADK Agent

| # | 模块 | TRD 引用 |
|---|------|----------|
| C4.1 | research_agent.py + prompt 模板（D1/D2/D3 各调一次） | richson SS8.2 |
| C4.2 | interpretation_agent.py + prompt 模板 | richson SS8.3 |
| C4.3 | execution_agent.py + prompt 模板 | richson SS8.4 |
| C4.4 | create_agent 工厂函数（LiteLlm 多 provider） | richson SS8.5 |
| C4.5 | run_agent（InMemoryRunner 驱动） | richson SS8.5 |

### C5. 外部数据源

| # | 模块 | TRD 引用 |
|---|------|----------|
| C5.1 | fred.py（FRED API） | richson SS3.1 |
| C5.2 | yahoo.py（yfinance） | richson SS3.1 |
| C5.3 | akshare_client.py（AKShare） | richson SS3.1 |
| C5.4 | polymarket.py（Polymarket API） | richson SS3.1 |
| C5.5 | cot.py（CFTC COT） | richson SS3.1 |
| C5.6 | wgc.py（World Gold Council 季报） | richson SS10 |
| C5.7 | stooq.py（stooq fallback） | richson SS3.1 |
| C5.8 | cache.py（进程内 TTL 缓存） | richson SS10 |

### C6. DB/Schema 层

| # | 模块 | TRD 引用 |
|---|------|----------|
| C6.1 | models.py（SQLAlchemy rs_* 表映射） | richson SS3.1 |
| C6.2 | repository.py（CRUD 操作） | richson SS3.1 |
| C6.3 | Pydantic request/response schemas（jobs/analysis/market/events/common） | richson SS3.1 |

### C7. 降级与模板

| # | 模块 | TRD 引用 |
|---|------|----------|
| C7.1 | interpretation_zh.py（中文降级文本模板） | richson SS8.6 |
| C7.2 | interpretation_en.py（英文降级文本模板） | richson SS8.6 |
| C7.3 | 降级策略实现（L2 跳过/L3 模板/维度部分失败） | richson SS8.6 + SS17 |

### C8. CLI 命令

| # | 命令 | TRD 引用 |
|---|------|----------|
| C8.1 | backfill --days 90（冷启动回填） | richson SS13.3 |
| C8.2 | backtest --start --end（模型验证） | richson SS18 |
| C8.3 | update-weights（维度权重更新） | richson SS6.2 |

### C9. 可观测性 + 部署

| # | 变更 | TRD 引用 |
|---|------|----------|
| C9.1 | structlog JSON 日志配置 | richson SS11.1 |
| C9.2 | 认证中间件（INTERNAL_API_KEY 校验） | richson SS4.2 |
| C9.3 | X-Request-ID 追踪中间件 | richson SS4.3 |
| C9.4 | 数据生命周期清理任务（rs_asset_analyses 保留策略） | richson SS13.1 |
| C9.5 | asyncio 内部事件调度（Polymarket 每小时拉取） | richson SS9.3 |

## D. richman 后端（Go，修改）

### D1. richson HTTP 客户端（新包）

| # | 变更 | TRD 引用 |
|---|------|----------|
| D1.1 | internal/richson/client.go（Client struct + NewClient） | richman SS3.1 |
| D1.2 | internal/richson/types.go（request/response 类型） | richman SS3.5 |
| D1.3 | TriggerAssetAnalysis 方法 | richman SS3.5 |
| D1.4 | TriggerBatchAnalysis 方法 | richman SS3.5 |
| D1.5 | GetJobStatus 方法 | richman SS3.5 |
| D1.6 | AnalyzeHolding 方法 | richman SS3.5 |
| D1.7 | GetDemoPlan 方法 | richman SS3.5 |
| D1.8 | GetMarketRegime 方法 | richman SS3.5 |
| D1.9 | GetOHLCV 方法 | richman SS3.5 |
| D1.10 | GetScoreHistory 方法 | richman SS3.5 |
| D1.11 | GetEventsRadar 方法 | richman SS3.5 |
| D1.12 | GenerateWeeklyInsight 方法 | richman SS3.5 |
| D1.13 | HealthCheck 方法 + IsHealthy() atomic.Bool | richman SS3.6 |
| D1.14 | 重试逻辑（网络错误/502/503 时重试 1 次） | richman SS3.2 |
| D1.15 | X-Request-ID 注入 | richman SS3.3 |
| D1.16 | 错误码映射（richsonErrorMap） | richman SS3.4 |

### D2. v2 API Handler 层（新包）

| # | 文件 | 端点 | TRD 引用 |
|---|------|------|----------|
| D2.1 | v2/market.go | GET /api/v2/market/regime（代理） | richman SS4.2 |
| D2.2 | v2/market.go | GET /api/v2/market/overview（聚合） | richman SS4.2 |
| D2.3 | v2/market.go | GET /api/v2/market/:code（聚合） | richman SS4.2 |
| D2.4 | v2/market.go | GET /api/v2/market/:code/ohlcv（代理） | richman SS4.2 |
| D2.5 | v2/market.go | GET /api/v2/market/:code/scores（代理） | richman SS4.2 |
| D2.6 | v2/market.go | GET /api/v2/market/:code/demo-plan（DB 读取 + fallback 代理） | richman SS4.2 |
| D2.7 | v2/market.go | GET /api/v2/market/:code/share（JWT 可选） | richman SS4.1 |
| D2.8 | v2/event.go | GET /api/v2/events/radar（代理） | richman SS4.2 |
| D2.9 | v2/analysis.go | POST /api/v2/analysis/trigger-asset（代理 + 注入 llmConfig） | richman SS4.2 |
| D2.10 | v2/analysis.go | GET /api/v2/analysis/jobs/:jobId（DB 直读） | richman SS4.2 |
| D2.11 | v2/analysis.go | POST /api/v2/analysis/holding/:holdingId（聚合 + richson + 持久化） | richman SS4.2 |
| D2.12 | v2/briefing.go | GET /api/v2/briefing（聚合） | richman SS4.2 |
| D2.13 | v2/feedback.go | POST /api/v2/feedback（DB 直写） | richman SS4.2 |
| D2.14 | v2/user.go | PATCH /api/v2/user/risk-preference | richman SS4.1 |
| D2.15 | v2/user.go | PATCH /api/v2/user/email-push | richman SS4.1 |
| D2.16 | v2/invite.go | GET /api/v2/invite/my-codes | invite SS5.1 |
| D2.17 | v2/invite.go | GET /api/v2/invite/my-invites | invite SS5.2 |
| D2.18 | v2/middleware/ratelimit.go（IP 限流中间件） | richman SS4.3 |
| D2.19 | v1 路由组增加 IP 限流中间件（auth 端点 5 次/分钟） | richman SS4.1 |
| D2.20 | DELETE /api/v1/auth/account（账户注销） | richman SS21.1 |

### D3. v2 Service 层

| # | Service | 方法 | TRD 引用 |
|---|---------|------|----------|
| D3.1 | MarketService | GetOverview（rm_asset_catalog + rs_asset_analyses 聚合） | richman SS5.2 |
| D3.2 | MarketService | GetAssetDetail（标的详情 + 维度 + percentileLabel） | richman SS5.2 |
| D3.3 | MarketService | percentileLabel 计算逻辑（365 天百分位 + TTL 缓存） | richman SS5.2 |
| D3.4 | BriefingService | GetBriefing（持仓 + 分析 + 决策卡片 + sparkline + 浮盈亏 + 集中度） | richman SS5.3 |
| D3.5 | FeedbackService | Create（校验 + 写入 rm_user_feedback） | richman SS5.4 |
| D3.6 | EmailPushService | SendDailyBriefing（每日简报邮件，含内容组装 + 游标分页） | richman SS7.2/SS7.5 |
| D3.7 | EmailPushService | SendWeeklyInsight（调 richson + 模板渲染） | richman SS7.6 |
| D3.8 | EmailPushService | SendMarketAlert（事件/评分驱动快讯） | richman SS7.2 |
| D3.9 | EmailPushService | SendHoldingSuggestion（个人持仓建议邮件） | richman SS7.2 |
| D3.10 | v2_holding.go | 持仓级分析完整流程（查持仓->查分析->查敞口->查风险偏好->查 LLM key->调 richson->持久化） | richman SS5.5 |
| D3.11 | v2_holding.go | 幂等防护（sync.Map per-user:holdingID TryLock） | richman SS5.5 |
| D3.12 | InviteService | GenerateCodesForUser | invite SS8.1 |
| D3.13 | InviteService | UseInviteCode（原子消费 + 双向奖励） | invite SS8.1 |
| D3.14 | InviteService | GetMyCodes | invite SS8.1 |
| D3.15 | InviteService | GetMyInvites | invite SS8.1 |
| D3.16 | InviteService | GetFirstAvailableCode（分享链接用） | invite SS8.1 |
| D3.17 | InviteService | UpdateLoginStreak（原子 SQL + 7 天解锁） | invite SS8.1 |
| D3.18 | ComputeConcentration 函数（集中度计算） | richman SS16 |

### D4. 现有 Service 变更

| # | Service | 变更 | TRD 引用 |
|---|---------|------|----------|
| D4.1 | UserService | 新增 UpdateRiskPreference 方法 | richman SS5.6 |
| D4.2 | UserService | 新增 UpdateEmailPush 方法 | richman SS7.4.1 |
| D4.3 | NotificationService | 新增 SendBroadcast 方法 | richman SS5.6 |
| D4.4 | AuthService | Register 增加 disclaimerAccepted 校验 | richman SS15.2.1 |
| D4.5 | AuthService | Register 集成 InviteService（消费专属码 + 生成 3 码） | invite SS8.2 |
| D4.6 | AuthService | Login 集成 InviteService.UpdateLoginStreak | invite SS8.2 |

### D5. v2 Repo 层

| # | Repo | 方法 | TRD 引用 |
|---|------|------|----------|
| D5.1 | AssetAnalysisReadRepo | GetLatestByAssetCode | richman SS6.1 |
| D5.2 | AssetAnalysisReadRepo | GetLatestByAssetCodes（批量） | richman SS6.1 |
| D5.3 | AssetAnalysisReadRepo | GetScoresForPercentile | richman SS6.1 |
| D5.4 | AssetAnalysisReadRepo | GetSparklineScores | richman SS6.1 |
| D5.5 | AnalysisDimensionReadRepo | GetByAnalysisID | richman SS6.1 |
| D5.6 | AnalysisJobReadRepo | GetByJobID | richman SS6.1 |
| D5.7 | EventAlertReadRepo | GetUnalerted | richman SS6.1 |
| D5.8 | EventAlertReadRepo | MarkAlerted（跨服务写入例外） | richman SS6.1 |
| D5.9 | UserFeedbackRepo | Create | richman SS6.2 |
| D5.10 | UserInviteCodeRepo（全套 CRUD） | invite SS10 |
| D5.11 | InviteRewardRepo（全套 CRUD） | invite SS10 |

### D6. 现有 Repo 变更

| # | Repo | 变更 | TRD 引用 |
|---|------|------|----------|
| D6.1 | 全部现有 repo SQL 表名更新为 rm_ 前缀 | richman SS6.3 |
| D6.2 | UserRepo | 新增 UpdateRiskPreference | richman SS6.3 |
| D6.3 | UserRepo | 新增 UpdateEmailPush | richman SS6.3 |
| D6.4 | HoldingRepo | 新增 GetExposureByAssetType | richman SS6.3 |
| D6.5 | AssetRepo | 新增 ListActiveWithType | richman SS6.3 |
| D6.6 | DecisionCardRepo | 新增 GetLatestByHoldings | richman SS6.3 |

### D7. Model 层新增

| # | Struct | TRD 引用 |
|---|--------|----------|
| D7.1 | AssetAnalysis（映射 rs_asset_analyses） | richman SS11 |
| D7.2 | AnalysisDimension（映射 rs_asset_analysis_dimensions） | richman SS11 |
| D7.3 | AnalysisJob（映射 rs_analysis_jobs） | richman SS11 |
| D7.4 | EventAlert（映射 rs_event_alerts） | richman SS11 |
| D7.5 | UserFeedback（映射 rm_user_feedback） | richman SS11 |
| D7.6 | UserInviteCode | invite SS10 |
| D7.7 | InviteReward | invite SS10 |
| D7.8 | InvitedUser（脱敏展示用） | invite SS10 |

### D8. Cron 任务

| # | 任务 | 触发时间 | TRD 引用 |
|---|------|----------|----------|
| D8.1 | 每日标的分析触发 | 06:00 UTC+8 | richman SS8.3 |
| D8.2 | 每日持仓分析触发 | 07:30 UTC+8 | richman SS8.3.1 |
| D8.3 | 每日简报邮件 | 08:30 UTC+8 | richman SS8.1 |
| D8.4 | A 股收盘后快讯 | 15:30 UTC+8 工作日 | richman SS8.3.2 |
| D8.5 | 评分变化触发市场快讯 | 06:00 分析完成后 | richman SS8.3.3 |
| D8.6 | 每周投研洞察 | 周一 08:30 UTC+8 | richman SS8.1 |
| D8.7 | 事件告警轮询 | 每小时整点 | richman SS8.4 |
| D8.8 | 过期 Job 清理 | 每 3 分钟（已知问题 22.4） | richman SS8.5 |
| D8.9 | richson 健康检查 | 每 30 秒 | richman SS3.6 |
| D8.10 | 账户数据清理 | 每日 03:00 UTC+8 | richman SS21.2 |
| D8.11 | 推送频率控制逻辑（每日 3 次上限） | richman SS7.7 |
| D8.12 | Cron 互斥锁（sync.Mutex + TryLock） | richman SS8.7 |

### D9. 邮件模板

| # | 文件 | TRD 引用 |
|---|------|----------|
| D9.1 | daily_briefing_zh.html | richman SS7.4 |
| D9.2 | daily_briefing_en.html | richman SS7.4 |
| D9.3 | weekly_insight_zh.html | richman SS7.4 |
| D9.4 | weekly_insight_en.html | richman SS7.4 |
| D9.5 | market_alert_zh.html | richman SS7.4 |
| D9.6 | market_alert_en.html | richman SS7.4 |
| D9.7 | holding_suggestion_zh.html | richman SS7.4 |
| D9.8 | holding_suggestion_en.html | richman SS7.4 |
| D9.9 | template/engine.go（模板渲染引擎） | richman SS7.4 |
| D9.10 | email/Sender + SendBatch（BCC 50 人分批） | richman SS7.3 |

### D10. 配置与启动

| # | 变更 | TRD 引用 |
|---|------|----------|
| D10.1 | config.go 新增 RichsonConfig struct | richman SS10.1 |
| D10.2 | config.go 新增 PlatformLLMConfig struct | richman SS10.1 |
| D10.3 | .env.example 追加 v2 变量模板 | richman SS10.3 |
| D10.4 | 启动检查：RICHSON_BASE_URL/API_KEY/PLATFORM_LLM_API_KEY 非空 | richman SS10.4 |
| D10.5 | 启动时异步检查 richson 连通性 | richman SS10.4 |
| D10.6 | main.go DI 链路更新（新 repo/service/handler 注入） | richman SS13.1 |
| D10.7 | 路由注册更新（v1 + v2 并行） | richman SS13.2 |
| D10.8 | shutdown 流程增加 cronScheduler.Stop() | richman SS22.3 |

### D11. 错误处理

| # | 错误码 | TRD 引用 |
|---|--------|----------|
| D11.1 | RICHSON_UNAVAILABLE (503) | richman SS14.1 |
| D11.2 | RICHSON_ERROR (502) | richman SS14.1 |
| D11.3 | ANALYSIS_NOT_FOUND (404) | richman SS14.1 |
| D11.4 | JOB_NOT_FOUND (404) | richman SS14.1 |
| D11.5 | ASSET_NOT_FOUND (404) | richman SS14.1 |
| D11.6 | INVALID_RISK_PREFERENCE (400) | richman SS14.1 |
| D11.7 | FEEDBACK_DUPLICATE (409) | richman SS14.1 |
| D11.8 | RATE_LIMIT_EXCEEDED (429) | richman SS14.1 |

### D12. v1 废弃处理

| # | 变更 | TRD 引用 |
|---|------|----------|
| D12.1 | internal/analysis/ 标记 deprecated | richman SS12.1 |
| D12.2 | internal/llm/ 标记 deprecated（crypto.go 保留） | richman SS12.1 |
| D12.3 | internal/datasource/ 标记 deprecated | richman SS12.1 |
| D12.4 | v1 废弃端点添加 Deprecation + Sunset 响应头 | richman SS15.3 |

## E. 前端（React/TypeScript）

### E1. 新增 Feature 模块

| # | 模块 | 包含文件 | TRD 引用 |
|---|------|----------|----------|
| E1.1 | features/market-overview/ | api + types + use-market-regime + use-market-overview + index | frontend SS3.1 |
| E1.2 | features/asset-detail/ | api + types + 6 hooks (use-asset-detail/ohlcv/score-history/demo-plan/trigger-holding/analysis-job) + index | frontend SS3.2 |
| E1.3 | features/event-radar/ | api + types + use-event-radar + index | frontend SS3.3 |
| E1.4 | features/research-briefing/ | api + types + use-briefing + index | frontend SS3.4 |
| E1.5 | features/user-feedback/ | api + types + use-submit-feedback + index | frontend SS3.5 |
| E1.6 | features/invite/ | api + types + use-my-codes + use-my-invites + index | frontend SS3.6 / invite SS10 |

### E2. 现有 Feature 模块变更

| # | 模块 | 变更 | TRD 引用 |
|---|------|------|----------|
| E2.1 | portfolio | 新增 useHoldingByAssetCode hook | frontend SS3.6 |
| E2.2 | portfolio | Holding 类型新增 entryMode 字段 | frontend SS3.6 |
| E2.3 | user-settings | 新增 riskPreference 字段 + usePatchRiskPreference hook | frontend SS3.6 |
| E2.4 | auth | 注册表单增加 disclaimerAccepted checkbox | frontend SS3.6 |
| E2.5 | auth | 注册表单增加 ref 参数自动填充 | frontend SS3.6 / invite SS6.2 |

### E3. 新增页面

| # | 页面 | TRD 引用 |
|---|------|----------|
| E3.1 | MarketOverviewPage（/market） | frontend SS4 |
| E3.2 | AssetDetailPage（/market/:code） | frontend SS5 |
| E3.3 | ResearchBriefingPage（/briefing，从 DashboardPage 重构） | frontend SS6 |
| E3.4 | RiskPreferenceSubPage（/settings/risk-preference） | frontend SS2.1 |

### E4. 新增组件（Market Overview 页）

| # | 组件 | TRD 引用 |
|---|------|----------|
| E4.1 | MarketRegimeBar（体制判断条） | frontend SS4.2 |
| E4.2 | AssetCardWall（卡片墙容器） | frontend SS4.3 |
| E4.3 | AssetGroupSection（分类组） | frontend SS4.3 |
| E4.4 | AssetCard（单个标的卡片，含激活/置灰两态） | frontend SS4.3 |
| E4.5 | EventRadarSection（事件雷达列表） | frontend SS4.6 |
| E4.6 | RegisterCTA（底部注册引导条） | frontend SS4.5 |

### E5. 新增组件（标的详情页）

| # | 组件 | TRD 引用 |
|---|------|----------|
| E5.1 | StickyHeader 容器 | frontend SS5.2 |
| E5.2 | AssetIdentity（名称 + 价格 + 涨跌） | frontend SS5.2 |
| E5.3 | ScoreSummary（评分 + 方向 + 分位） | frontend SS5.2 |
| E5.4 | ChangeSummary（变化摘要，条件展示 delta>=5） | frontend SS5.2 |
| E5.5 | MajorChangeRecap（重大变化复盘，条件展示 |delta|>20） | frontend SS5.2 |
| E5.6 | ConflictWarning（冲突警告） | frontend SS5.2 |
| E5.7 | FreshnessIndicator（数据新鲜度三级警告） | frontend SS5.2 |
| E5.8 | OhlcvChart（K 线图 + SMA200 + 支撑/阻力位，lightweight-charts v4） | frontend SS5.3 |
| E5.9 | InterpretationCard（市场解读文本） | frontend SS5.3 |
| E5.10 | DimensionPanelList（四维折叠面板 + 子指标表） | frontend SS5.3 |
| E5.11 | ScoreTrendChart（评分趋势线 + 版本变更竖线，echarts） | frontend SS5.3 |
| E5.12 | RiskFactorList | frontend SS5.4 |
| E5.13 | KeyPriceLevels（支撑/阻力位表格 + CNY 附注 USD 等价） | frontend SS5.4 |
| E5.14 | DrawdownReference（回撤对比） | frontend SS5.4 |
| E5.15 | EventCalendar（标的相关事件） | frontend SS5.4 |
| E5.16 | DemoPlanWithRegisterCTA | frontend SS5.5 |
| E5.17 | DemoPlanWithAddHoldingCTA | frontend SS5.5 |
| E5.18 | FullExecutionPlan（条件分支执行计划完整展示） | frontend SS5.5 |
| E5.19 | ScoreBar（置信区间色带） | frontend SS11.3 |

### E6. 新增组件（投研简报页）

| # | 组件 | TRD 引用 |
|---|------|----------|
| E6.1 | BriefingHeader（标题 + 简洁/详细切换） | frontend SS6.1 |
| E6.2 | BriefingCardList | frontend SS6.1 |
| E6.3 | BriefingCard（持仓决策卡片，含 sparkline + 反馈按钮） | frontend SS6.2 |
| E6.4 | EmptyBriefingState（无持仓空状态） | frontend SS6.1 |

### E7. 新增组件（持仓管理）

| # | 组件 | TRD 引用 |
|---|------|----------|
| E7.1 | ModeSelector（标记/快速/明细模式切换） | frontend SS7.1 |
| E7.2 | TagModeForm（标记模式表单 + PositionTierRadio） | frontend SS7.1 |
| E7.3 | RiskPreferenceModal（三型风险偏好选择弹窗） | frontend SS7.2 |
| E7.4 | 持仓列表升级提示标签 | frontend SS7.3 |
| E7.5 | 持仓集中度 Alert（红/橙/蓝三级） | frontend SS7.4 |
| E7.6 | LLM 配置引导（首次持仓触发） | frontend SS7.5 |

### E8. 新增组件（设置页扩展）

| # | 组件 | TRD 引用 |
|---|------|----------|
| E8.1 | InviteSection（邀请码列表 + 解锁进度 + 已邀请列表） | invite SS7.1 / frontend SS3.6 |
| E8.2 | EmailPushToggle（平台邮件推送开关） | frontend SS3.6 |
| E8.3 | AccountDeletionSection（账户注销，需密码确认） | frontend SS3.6 |

### E9. 路由变更

| # | 变更 | TRD 引用 |
|---|------|----------|
| E9.1 | / 重定向至 /market | frontend SS2.1 |
| E9.2 | 新增 /market 路由 | frontend SS2.1 |
| E9.3 | 新增 /market/:code 路由 | frontend SS2.1 |
| E9.4 | 新增 /settings/risk-preference 路由 | frontend SS2.1 |
| E9.5 | 移除 /onboarding/* 全部路由 | frontend SS2.2 |
| E9.6 | 移除 /decision-cards/:id 路由（合并到标的详情页） | frontend SS2.2 |
| E9.7 | 移除 OnboardingGuard | frontend SS2.4 |
| E9.8 | AuthGuard 简化（移除 onboarding 检查） | frontend SS2.4 |
| E9.9 | 导航栏更新（行情/持仓/投研简报三项） | frontend SS2.3 |

### E10. API 客户端

| # | 变更 | TRD 引用 |
|---|------|----------|
| E10.1 | domain/http/client.ts 拆分为 requestV1/requestV2/requestPublic | frontend SS8.1 |
| E10.2 | 现有 feature 调用点从 request() 迁移到 requestV1() | frontend SS16.1 |
| E10.3 | useAnalysisJob 轮询 hook（3s 间隔，最大 60 次） | frontend SS8.3 |

### E11. i18n

| # | 变更 | TRD 引用 |
|---|------|----------|
| E11.1 | 新增 market.* 翻译 key（zh + en） | frontend SS9.2 |
| E11.2 | 新增 assetDetail.* 翻译 key（zh + en） | frontend SS9.2 |
| E11.3 | 新增 briefing.* / eventRadar.* 翻译 key（zh + en） | frontend SS9.2 |
| E11.4 | 新增 settings.account.riskPreference.* 翻译 key | frontend SS9.1 |
| E11.5 | 新增 settings.account.emailPush.* 翻译 key | frontend SS9.1 |
| E11.6 | 新增 settings.account.deleteAccount.* 翻译 key | frontend SS9.1 |
| E11.7 | 新增 settings.invite.* 翻译 key | frontend SS9.1 |
| E11.8 | 新增 common.signal.* / common.regime.* 翻译 key | frontend SS9.1 |
| E11.9 | 新增 common.disclaimer.* 翻译 key | frontend SS14 |

### E12. SEO + 依赖

| # | 变更 | TRD 引用 |
|---|------|----------|
| E12.1 | pnpm add @dr.pogodin/react-helmet | frontend SS12.1 |
| E12.2 | MarketOverviewPage meta 标签（Helmet） | frontend SS12.1 |
| E12.3 | AssetDetailPage meta 标签（Helmet） | frontend SS12.1 |
| E12.4 | 涨跌颜色函数 getPriceChangeColor | frontend SS4.4 |
| E12.5 | 免责声明展示（4 个位置） | frontend SS14 |
| E12.6 | 简洁/详细模式切换（localStorage richman_briefing_view_mode） | frontend SS6.3 |

### E13. 废弃代码清理

| # | 变更 | TRD 引用 |
|---|------|----------|
| E13.1 | 删除 pages/onboarding/* 全部文件 | frontend SS13.1 |
| E13.2 | 删除 domain/auth/onboarding-guard.tsx | frontend SS13.1 |
| E13.3 | dashboard-summary feature 改名为 research-briefing + API 切换 v2 | frontend SS13.2 |

## F. 部署与配置

| # | 变更 | TRD 引用 |
|---|------|----------|
| F1 | docker-compose.yml 新增 richson 服务定义 | richson SS12.1 |
| F2 | richson .env.example | richson SS12.2 |
| F3 | richman .env.example 追加 v2 变量 | richman SS10.3 |
| F4 | docker-compose richson 端口改为 expose（已知问题 21.1） | richson SS21.1 |
| F5 | docker-compose 两服务 healthcheck 配置 | richman SS22.2 / richson SS11.3 |

## G. 已知问题（编码阶段必须处理）

### G1. richson 已知问题

| # | 问题 | TRD 引用 |
|---|------|----------|
| G1.1 | richson 端口暴露 -> expose/127.0.0.1 绑定 | richson SS21.1 |
| G1.2 | healthcheck curl 依赖 -> Dockerfile 安装或 python 替代 | richson SS21.2 |
| G1.3 | asyncio 事件循环阻塞 -> to_thread/ProcessPoolExecutor | richson SS21.3 |
| G1.4 | rs_event_alerts + rs_asset_analysis_dimensions 序列 RESTART WITH | richson SS21.4 |
| G1.5 | LLM 成本上限 -> DAILY_LLM_BUDGET_USD 环境变量 | richson SS21.5 |
| G1.6 | 集中度警告文本硬编码中文 -> locale 参数化 | richson SS21.6 |
| G1.7 | backfill 数据标记 -> 评估 percentileLabel 是否排除 | richson SS21.7 |
| G1.8 | validDays 下界约束 -> Pydantic Field(ge=1, le=90) | richson SS21.8 |
| G1.9 | LLM apiKey 日志脱敏 -> repr=False + 中间件脱敏 | richson SS21.9 |

### G2. richman 已知问题

| # | 问题 | TRD 引用 |
|---|------|----------|
| G2.1 | CORS 白名单 -> CORS_ALLOWED_ORIGINS 环境变量 | richman SS22.1 |
| G2.2 | richman 健康检查端点 -> 实现 GET /health + docker healthcheck | richman SS22.2 |
| G2.3 | Cron 优雅关闭 -> cronScheduler.Stop() + 等待 60s | richman SS22.3 |
| G2.4 | analysis job 卡死恢复 -> 清理间隔 10min->3min | richman SS22.4 |
| G2.5 | 密码策略强化 -> 最低 8 位 + 大小写 + 数字 | richman SS22.5 |
| G2.6 | JWT 7 天有效期注释标明 Phase 2 refresh token | richman SS22.6 |
| G2.7 | CSP 安全头部 -> 中间件或 nginx | richman SS22.7 |
| G2.8 | 邀请码暴力破解防护 -> 失败计数锁定 | richman SS22.8 |
| G2.9 | SS9.1 端点表与路由树不一致 -> 以 SS4.1 为准 | richman SS22.9 |
| G2.10 | hard delete invite 表悬空引用处理 | richman SS22.10 |
| G2.11 | 数据库备份策略 -> pg_dump 每日 + WAL | richman SS22.11 |
| G2.12 | migration 020 嵌套事务 -> 移除内部 BEGIN/COMMIT | richman SS22.12 |
| G2.13 | plan 列改名 subscription_tier | richman SS22.13 |
| G2.14 | score alert 与 market alert 去重边界 | richman SS22.14 |
| G2.15 | 认证端点用户级限流 | richman SS22.15 |

### G3. 前端已知问题

| # | 问题 | TRD 引用 |
|---|------|----------|
| G3.1 | HTTP client 迁移（request -> requestV1） | frontend SS16.1 |
| G3.2 | react-helmet barrel 豁免 | frontend SS16.2 |
| G3.3 | v1/v2 决策卡片展示区分 | frontend SS16.3 |
| G3.4 | i18n namespace 拆分评估（500 key 阈值） | frontend SS16.4 |
| G3.5 | 涨跌颜色逻辑改用 assetType 判定 | frontend SS16.5 |
| G3.6 | richson 503 asset detail page 降级 UI | frontend SS16.6 |
| G3.7 | localStorage key 前缀统一迁移 | frontend SS16.7 |
| G3.8 | dashboard-llm-status feature 处置 | frontend SS16.8 |
| G3.9 | Market Overview 页 richson 503 降级 | frontend SS16.9 |
| G3.10 | 邮件模板 CTA 链接目标定义 | frontend SS16.10 |

### G4. 邀请系统已知问题

| # | 问题 | TRD 引用 |
|---|------|----------|
| G4.1 | login_streak 增长无上限 -> 邀请码总数上限 20 | invite SS11.1 |
| G4.2 | login_streak 时区边界 -> Asia/Shanghai | invite SS11.2 |
| G4.3 | used_by_user_id 悬空引用 -> hard delete 流程处理 | invite SS11.3 |

## 统计

| 层 | 条目数 |
|----|--------|
| A. DB - richman 迁移 | 37 |
| B. DB - richson Alembic | 6 |
| C. richson 服务（新建） | 49 |
| D. richman 后端（修改） | 96 |
| E. 前端（修改） | 73 |
| F. 部署配置 | 5 |
| G. 已知问题 | 37 |
| **合计** | **303** |
