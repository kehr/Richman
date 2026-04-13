# richson 量化与 LLM 编排服务 TRD

> 版本 1.0 | 关联 PRD: docs/prds/richman-prd-v2.md

## 1. 文档范围

本 TRD 覆盖 v2 版本中 richson（Python 量化 + LLM 编排服务）的完整技术设计，包含：

- richson 服务架构与目录结构
- richman-richson 通信协议（HTTP REST）
- 数据库 schema 演进（存量表 rm_ 前缀迁移 + richson 新表 rs_）
- 量化评分引擎（Layer 1）内部设计
- LLM 智能层（Layer 2 / Layer 3）Google ADK Agent 设计
- 异步任务追踪与前端查询
- richman 侧 v2 API 新增与变更
- 可观测性、降级策略、部署配置
- v1 到 v2 迁移策略

不在本 TRD 范围：前端页面重构（Market Overview、标的详情页、投研简报页）、前端路由变更、通知推送邮件模板，这些由各自独立 TRD 覆盖。

## 2. 术语表

| 术语 | 含义 |
|------|------|
| richman | Go 主服务，负责 API 网关、用户认证、持仓 CRUD、决策卡片持久化、通知推送、定时调度 |
| richson | Python 侧车服务，负责全部金融数据获取、量化计算、LLM 编排、分析结果写入 |
| Layer 1 | 量化底座：确定性数值计算，不涉及 LLM |
| Layer 2 | LLM 智能层：信息检索与研判，输出定性判断，不直接输出数值 |
| Layer 3 | 综合输出层：LLM 生成解读文本、风险因子、执行计划 |
| ADK | Google Agent Development Kit，Python LLM Agent 框架 |
| 标的级分析 | 平台预计算的资产分析，不含用户持仓上下文，所有用户共享 |
| 持仓级分析 | 包含用户持仓信息的个性化分析，生成执行计划 |

## 3. 服务架构

### 3.1 Monorepo 目录结构

```
richson/                          # Python service root
  pyproject.toml                  # uv/pip dependencies, project metadata
  Dockerfile
  .env.example
  alembic.ini                     # DB migration config
  alembic/                        # migration scripts
    versions/
  src/
    richson/
      __init__.py
      main.py                     # FastAPI app entry
      config.py                   # pydantic-settings, env loading
      api/                        # FastAPI routers
        __init__.py
        health.py                 # GET /health
        jobs.py                   # POST /jobs/analyze-asset, POST /jobs/batch-analyze
        analysis.py               # POST /analyze/holding, POST /analyze/demo-plan
        market.py                 # GET /market/regime, GET /market/ohlcv/{code}
        assets.py                 # GET /assets/{code}/score-history
        events.py                 # GET /events/radar
        content.py                # POST /content/weekly-insight
      core/                       # business logic
        __init__.py
        pipeline.py               # L1 -> L2 -> L3 orchestration
        scoring.py                # dimension scoring + percentile
        adjustment.py             # LLM qualitative -> numeric mapping
        confidence.py             # confidence calculation
        indicators/               # per-dimension indicator calculators
          __init__.py
          d1_macro_rates.py
          d2_dollar_liquidity.py
          d3_structural_demand.py
          d4_technical_position.py
        support_resistance.py     # support/resistance level calculation
        regime.py                 # market regime detection
        event_monitor.py          # Polymarket event delta monitoring
      agents/                     # Google ADK agents
        __init__.py
        research_agent.py         # Layer 2: info retrieval + judgment
        interpretation_agent.py   # Layer 3: text generation
        execution_agent.py        # Layer 3: execution plan generation
        prompts/                  # prompt templates
          research.py
          interpretation.py
          execution.py
      datasources/                # external data fetching
        __init__.py
        fred.py                   # FRED API wrapper
        yahoo.py                  # yfinance wrapper
        akshare_client.py         # AKShare wrapper
        polymarket.py             # Polymarket API
        cot.py                    # CFTC COT data
        wgc.py                    # World Gold Council quarterly data (central bank buying + AISC)
        stooq.py                  # stooq fallback for price
        cache.py                  # in-memory TTL cache layer
      db/                         # database access
        __init__.py
        models.py                 # SQLAlchemy models for rs_* tables
        repository.py             # CRUD operations
      schemas/                    # pydantic request/response models
        __init__.py
        jobs.py
        analysis.py
        market.py
        events.py
        common.py
      templates/                  # degraded-mode text templates
        interpretation_zh.py
        interpretation_en.py
```

### 3.2 技术栈

| 层 | 选型 | 版本要求 |
|----|------|----------|
| 语言 | Python | >= 3.12 |
| Web 框架 | FastAPI | >= 0.115 |
| LLM 编排 | google-adk | >= 1.0 |
| ORM / DB | SQLAlchemy 2.0 + asyncpg | |
| 数据处理 | pandas >= 3.0, numpy | pandas 3.0 于 2026-01 发布，需 Python >= 3.12 |
| 技术指标 | pandas-ta-classic | |
| 金融数据 | fredapi, yfinance, akshare | |
| 配置管理 | pydantic-settings | |
| 日志 | structlog (JSON) | |
| 迁移 | Alembic | |
| 包管理 | uv | |

### 3.3 进程模型

richson 以单进程 uvicorn 启动，通过 FastAPI 异步处理请求。耗时的标的级分析通过 asyncio.create_task 在后台执行，状态写入 rs_analysis_jobs 表。同步端点（holding 分析、demo plan）在请求生命周期内完成。

不引入 Celery 等任务队列——MVP 阶段标的数量少（仅黄金），单进程 asyncio 足够。后续标的扩展时可引入 arq 或 Celery。

## 4. 通信协议

### 4.1 交互模式分类

richman 与 richson 之间有三种交互模式，按数据流向和持久化需求区分：

| 模式 | 触发方 | 持久化 | 超时 | 典型场景 |
|------|--------|--------|------|----------|
| A: 异步触发 + DB 写入 | richman cron / 用户请求 | richson 写 rs_* 表 | 无 HTTP 超时（后台任务） | 标的级分析、批量分析 |
| B: 同步调用 + richman 持久化 | richman | richman 持久化返回结果 | 30s | 持仓级分析 |
| C: 同步调用 + 无持久化 | richman | 不持久化（或 richman 侧缓存） | 10s | demo plan、市场体制、K线数据 |

### 4.2 认证

richson 是内部服务，不暴露到公网。认证方式：

- richman 调用 richson 时在 HTTP header 中携带共享密钥：`Authorization: Bearer {INTERNAL_API_KEY}`
- INTERNAL_API_KEY 通过环境变量注入，不存入数据库
- richson 中间件校验 header，拒绝无效请求
- 生产环境 richman 与 richson 之间通过 Docker 内部网络通信（HTTP）；若部署在不同主机，需启用 HTTPS

### 4.3 请求链路追踪

richman 调用 richson 时传递 `X-Request-ID` header（UUID）。richson 在所有日志和 DB 写入中绑定该 ID，实现跨服务关联追踪。

### 4.4 错误响应格式

统一使用 richman 的错误格式：

```json
{
  "error": {
    "code": "ANALYSIS_IN_PROGRESS",
    "message": "An analysis job is already running for this asset",
    "details": []
  }
}
```

richson 特有错误码：

| 错误码 | HTTP 状态 | 含义 |
|--------|-----------|------|
| ANALYSIS_IN_PROGRESS | 409 | 同一标的/持仓已有进行中的分析 |
| DATA_SOURCE_UNAVAILABLE | 502 | 外部数据源不可达 |
| LLM_TIMEOUT | 504 | LLM 调用超时 |
| LLM_INVALID_RESPONSE | 502 | LLM 返回格式不合预期 |
| ASSET_NOT_SUPPORTED | 400 | 标的类型未实现分析模型 |
| INSUFFICIENT_HISTORY | 400 | 历史数据不足以计算百分位 |

## 5. richson HTTP API 契约

### 5.1 模式 A：异步标的级分析

#### POST /jobs/analyze-asset

触发单个标的的完整三层分析。richson 创建 job 记录后立即返回 202，后台异步执行。

请求：
```json
{
  "assetCode": "GLD",
  "locale": "zh",
  "llmConfig": {
    "provider": "claude",
    "model": "claude-sonnet-4-20250514",
    "apiKey": "sk-..."
  },
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

响应（202）：
```json
{
  "data": {
    "jobId": "a1b2c3d4-...",
    "status": "pending",
    "assetCode": "GLD",
    "createdAt": "2026-04-13T10:00:00Z"
  }
}
```

说明：
- `locale` 控制 Layer 3 文本生成语言。资产级分析的结构化数据（分数/维度/delta）与语言无关，文本按 locale 分别生成并缓存
- `llmConfig` 为平台配额时由 richman 传入系统默认配置。标的级分析统一使用平台配额
- richson 收到请求后检查是否有同资产进行中的 job（partial unique index），有则返回 409

#### POST /jobs/batch-analyze

批量触发多个标的分析。由 richman cron 每日定时调用。

请求：
```json
{
  "assets": [
    { "assetCode": "GLD", "locale": "zh" },
    { "assetCode": "IAU", "locale": "zh" },
    { "assetCode": "518880", "locale": "zh" }
  ],
  "llmConfig": { "provider": "claude", "model": "...", "apiKey": "..." },
  "requestId": "..."
}
```

响应（202）：
```json
{
  "data": {
    "jobs": [
      { "jobId": "...", "assetCode": "GLD", "status": "pending" },
      { "jobId": "...", "assetCode": "IAU", "status": "pending" }
    ],
    "skipped": [
      { "assetCode": "518880", "reason": "ANALYSIS_IN_PROGRESS" }
    ]
  }
}
```

#### GET /jobs/{jobId}

查询 job 状态和进度。richman 代理此接口给前端。

响应：
```json
{
  "data": {
    "jobId": "a1b2c3d4-...",
    "assetCode": "GLD",
    "status": "running",
    "currentStep": "layer2_d3",
    "progress": 0.6,
    "steps": [
      { "name": "data_fetch", "status": "completed", "durationMs": 2300 },
      { "name": "layer1_scoring", "status": "completed", "durationMs": 800 },
      { "name": "layer2_d1", "status": "completed", "durationMs": 5200 },
      { "name": "layer2_d2", "status": "completed", "durationMs": 4800 },
      { "name": "layer2_d3", "status": "running", "durationMs": null },
      { "name": "layer3_interpretation", "status": "pending", "durationMs": null },
      { "name": "persist", "status": "pending", "durationMs": null }
    ],
    "error": null,
    "createdAt": "2026-04-13T10:00:00Z",
    "startedAt": "2026-04-13T10:00:01Z",
    "completedAt": null
  }
}
```

job status 状态机：`pending` -> `running` -> `completed` | `failed`

step status 状态机：`pending` -> `running` -> `completed` | `failed` | `skipped`

### 5.2 模式 B：同步持仓级分析

#### POST /analyze/holding

基于用户持仓生成个性化执行计划。richman 在收到前端请求后同步调用 richson，获取结果后由 richman 持久化。

请求：
```json
{
  "assetCode": "GLD",
  "assetAnalysisId": 10001,
  "holding": {
    "holdingId": 100023,
    "costPrice": 4500.00,
    "positionRatio": 8.0,
    "quantity": 10
  },
  "riskPreference": "moderate",
  "peerExposure": 12.0,
  "language": "zh",
  "llmConfig": {
    "provider": "claude",
    "model": "claude-sonnet-4-20250514",
    "apiKey": "sk-..."
  },
  "requestId": "..."
}
```

字段说明：
- `assetAnalysisId`：引用 rs_asset_analyses 中最新的分析记录 ID，保证持仓分析基于确定的资产分析快照
- `riskPreference`：`conservative` | `moderate` | `aggressive`，映射 PRD SS7.6 风险参数
- `peerExposure`：同一二级分类下所有持仓的合计仓位比例，用于同类标的执行协调（PRD SS8.2）

响应（200，超时 30s）：
```json
{
  "data": {
    "action": "scale_in_on_dip",
    "actionLabel": "逢回调加仓",
    "defaultAction": "维持现有仓位，等待明确信号",
    "currentPosition": 8.0,
    "targetPosition": 12.0,
    "scenarios": [
      {
        "condition": "金价回调至支撑位 $4,600",
        "action": "加仓 2%",
        "lotCount": 2,
        "rationale": "支撑确认 + 评分维持看涨区间",
        "priority": 2
      },
      {
        "condition": "金价跌破关键支撑 $4,400",
        "action": "减仓至 4%",
        "lotCount": -4,
        "rationale": "止损保护",
        "priority": 1
      }
    ],
    "stopLoss": 4400.00,
    "takeProfit": 5100.00,
    "validDays": 7,
    "noTriggerNote": "若 7 天内无场景触发，维持现有仓位，下期自动刷新",
    "concentrationLevel": "blue",
    "concentrationMessage": "黄金配置已达 12%，处于机构建议区间上限"
  }
}
```

### 5.3 模式 C：同步轻量查询

#### POST /analyze/demo-plan

为未登录或无持仓用户生成示范执行计划。使用最新的 rs_asset_analyses 数据 + 固定假设参数，不触发新分析。

请求：
```json
{
  "assetCode": "GLD",
  "language": "zh",
  "llmConfig": { "provider": "claude", "model": "...", "apiKey": "..." },
  "requestId": "..."
}
```

响应（200，超时 10s）：与 holding 分析格式相同，额外标记 `"isDemoPlan": true`。

Demo Plan 的假设参数（固定值，不可配置）：
- 仓位比例：10%
- 成本价：当前价 x 0.95
- 风险偏好：moderate
- 同类敞口：10%

#### GET /market/regime

返回当前宏观体制判断。

响应：
```json
{
  "data": {
    "regime": "risk_on",
    "regimeLabel": "风险偏好",
    "reason": "VIX 持续低于 15，收益率曲线正常化",
    "vix": 14.2,
    "t10y2y": 0.35,
    "creditSpread": 1.05,
    "indices": [
      { "name": "S&P 500", "code": "^GSPC", "price": 5820.50, "changePercent": 0.8 },
      { "name": "Nasdaq", "code": "^IXIC", "price": 18350.20, "changePercent": 1.2 },
      { "name": "Shanghai Composite", "code": "000001.SS", "price": 3180.50, "changePercent": -0.3 },
      { "name": "Gold", "code": "GC=F", "price": 4750.00, "changePercent": 0.5 }
    ],
    "updatedAt": "2026-04-13T06:00:00Z"
  }
}
```

体制判断规则（PRD SS4.2.1）：
- VIX > 25 且持续 >= 5 个交易日 -> `risk_off`
- VIX < 15 且 T10Y2Y > 0 -> `risk_on`
- 其余 -> `neutral`

`t10y2y` 为 FRED `T10Y2Y` 系列最新值，前端不直接使用但体制判断逻辑依赖。信用利差（BAMLC0A0CM）作为确认信号。一句话原因由 LLM 生成或模板生成。

#### GET /market/ohlcv/{assetCode}

返回 OHLCV K 线数据，供前端图表渲染。

查询参数：
- `period`：`1D` | `1W` | `1M` | `3M` | `1Y`，默认 `3M`

响应：
```json
{
  "data": {
    "assetCode": "GLD",
    "currency": "USD",
    "period": "3M",
    "candles": [
      { "date": "2026-04-12", "open": 4720, "high": 4755, "low": 4710, "close": 4750, "volume": 12500000 }
    ],
    "sma200": 4520.30,
    "supportLevels": [4600, 4480],
    "resistanceLevels": [4800, 4850]
  }
}
```

#### GET /assets/{assetCode}/score-history

返回历史评分序列，用于趋势线渲染。

查询参数：
- `days`：`30` | `90` | `180` | `240`，默认 `90`

响应：
```json
{
  "data": {
    "assetCode": "GLD",
    "scores": [
      {
        "date": "2026-04-12",
        "overallScore": 72.5,
        "d1Score": 65.0,
        "d2Score": 78.0,
        "d3Score": 82.0,
        "d4Score": 60.0,
        "modelVersion": "gold_v1.0"
      }
    ],
    "versionChanges": [
      { "date": "2026-03-01", "fromVersion": "gold_v0.9", "toVersion": "gold_v1.0", "note": "D3 央行购金权重调整" }
    ]
  }
}
```

#### GET /events/radar

返回未来 7 天关键宏观事件 + Polymarket 概率。

响应：
```json
{
  "data": {
    "events": [
      {
        "date": "2026-04-16",
        "title": "FOMC Meeting Minutes Release",
        "category": "monetary_policy",
        "impact": "high",
        "goldDirection": "bullish",
        "probability": 0.85,
        "probabilitySource": "polymarket",
        "probabilityChange24h": 0.05
      }
    ],
    "updatedAt": "2026-04-13T06:00:00Z"
  }
}
```

数据来源：Polymarket 事件合约 + 公开经济日历（FOMC 会议、CPI 发布、非农数据等固定日程）。固定日程事件从内置经济日历表获取，不依赖外部 API；Polymarket 提供概率数据。当 Polymarket 不可用时，事件仍展示但无概率列。

### 5.4 健康检查

#### GET /health

响应：
```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "fred": "ok",
    "yahoo": "ok",
    "akshare": "degraded",
    "polymarket": "ok"
  },
  "version": "1.0.0",
  "uptime": 86400
}
```

## 6. 数据库 Schema 演进

### 6.1 存量表 rm_ 前缀迁移

v2 引入应用前缀规则（docs/standards/database.md）。所有 richman 存量表通过迁移统一加 `rm_` 前缀。

需要重命名的表（共 14 张）：

| 原表名 | 新表名 |
|--------|--------|
| users | rm_users |
| plans | rm_plans |
| invite_codes | rm_invite_codes |
| asset_catalog | rm_asset_catalog |
| holdings | rm_holdings |
| trades | rm_trades |
| analysis_results | rm_analysis_results |
| decision_cards | rm_decision_cards |
| notification_channels | rm_notification_channels |
| notification_logs | rm_notification_logs |
| analysis_tasks | rm_analysis_tasks |
| llm_configs | rm_llm_configs |
| user_schedule_settings | rm_user_schedule_settings |
| holding_schedule_overrides | rm_holding_schedule_overrides |

迁移脚本设计（richman 侧 migration 021）：

```sql
-- 021_rename_tables_rm_prefix.up.sql
-- Rename all existing tables to use rm_ prefix.
-- Index and constraint names are NOT auto-renamed by ALTER TABLE RENAME;
-- we leave existing index names as-is (they still function correctly).
-- New indexes created after this migration will use the rm_ prefixed naming.

ALTER TABLE users RENAME TO rm_users;
ALTER TABLE plans RENAME TO rm_plans;
ALTER TABLE invite_codes RENAME TO rm_invite_codes;
ALTER TABLE asset_catalog RENAME TO rm_asset_catalog;
ALTER TABLE holdings RENAME TO rm_holdings;
ALTER TABLE trades RENAME TO rm_trades;
ALTER TABLE analysis_results RENAME TO rm_analysis_results;
ALTER TABLE decision_cards RENAME TO rm_decision_cards;
ALTER TABLE notification_channels RENAME TO rm_notification_channels;
ALTER TABLE notification_logs RENAME TO rm_notification_logs;
ALTER TABLE analysis_tasks RENAME TO rm_analysis_tasks;
ALTER TABLE llm_configs RENAME TO rm_llm_configs;
ALTER TABLE user_schedule_settings RENAME TO rm_user_schedule_settings;
ALTER TABLE holding_schedule_overrides RENAME TO rm_holding_schedule_overrides;
```

对应的 down 脚本反向重命名。

影响范围：
- 所有 sqlc query 文件中的表名需更新
- Go repo 层所有 SQL 查询需重新生成
- 外键引用自动跟随 RENAME（PostgreSQL 自动更新 pg_constraint 中的表 OID）
- 序列名不自动变更（如 `users_user_id_seq` 仍保留旧名，不影响功能）

### 6.2 richson 新表（rs_ 前缀）

richson 拥有写权限的三张核心表：

#### rs_asset_analyses

存储标的级分析结果。每次完整分析生成一条记录。

```sql
CREATE TABLE rs_asset_analyses (
    asset_analysis_id   BIGSERIAL PRIMARY KEY,
    asset_code          VARCHAR(32) NOT NULL,
    locale              VARCHAR(8)  NOT NULL DEFAULT 'zh',

    -- overall
    overall_score       DECIMAL(5,2) NOT NULL,
    signal_level        VARCHAR(32) NOT NULL,  -- strong_bullish|moderate_bullish|neutral|moderate_bearish|strong_bearish
    confidence          DECIMAL(5,2) NOT NULL,
    confidence_band_low DECIMAL(5,2) NOT NULL,
    confidence_band_high DECIMAL(5,2) NOT NULL,
    model_version       VARCHAR(32) NOT NULL,

    -- market interpretation (Layer 3 output)
    market_interpretation TEXT NOT NULL DEFAULT '',
    risk_factors        JSONB NOT NULL DEFAULT '[]',
    regime_summary      TEXT NOT NULL DEFAULT '',

    -- dimension scores (denormalized for query performance)
    d1_score            DECIMAL(5,2),
    d1_base_score       DECIMAL(5,2),
    d1_llm_adjustment   DECIMAL(5,2) DEFAULT 0,
    d2_score            DECIMAL(5,2),
    d2_base_score       DECIMAL(5,2),
    d2_llm_adjustment   DECIMAL(5,2) DEFAULT 0,
    d3_score            DECIMAL(5,2),
    d3_base_score       DECIMAL(5,2),
    d3_llm_adjustment   DECIMAL(5,2) DEFAULT 0,
    d4_score            DECIMAL(5,2),
    d4_base_score       DECIMAL(5,2),

    -- weights (snapshot at analysis time)
    d1_weight           DECIMAL(4,2) NOT NULL DEFAULT 0.30,
    d2_weight           DECIMAL(4,2) NOT NULL DEFAULT 0.25,
    d3_weight           DECIMAL(4,2) NOT NULL DEFAULT 0.25,
    d4_weight           DECIMAL(4,2) NOT NULL DEFAULT 0.20,

    -- degradation markers
    llm_skipped         BOOLEAN NOT NULL DEFAULT FALSE,
    data_coverage       VARCHAR(16) NOT NULL DEFAULT 'full',  -- full|partial|degraded

    -- conflict detection
    conflict_type       VARCHAR(16),
    conflict_message    TEXT,

    -- change tracking (vs previous analysis)
    prev_analysis_id    BIGINT,
    score_delta         DECIMAL(5,2),
    change_summary      TEXT,
    major_change_recap  TEXT,

    -- currency conversion (for CNY assets, snapshot at analysis time)
    usd_exchange_rate   DECIMAL(12,6),  -- e.g. 0.137 for CNY/USD; NULL for USD assets

    -- data freshness
    data_snapshot_at    TIMESTAMPTZ NOT NULL,
    price_at_analysis   DECIMAL(20,6),

    -- demo plan (pre-computed, PRD SS5.2.4)
    demo_plan           JSONB,

    -- extensible metadata (drawdown reference, etc.)
    analysis_metadata   JSONB NOT NULL DEFAULT '{}',

    -- generation mode: 'full' (L1+L2+L3), 'l1_only' (LLM skipped, template text), 'backfill' (historical)
    generated_by        VARCHAR(16) NOT NULL DEFAULT 'full',

    -- metadata
    source              VARCHAR(16) NOT NULL DEFAULT 'scheduled',
    job_id              UUID,
    analyzed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator             VARCHAR(64) NOT NULL DEFAULT 'richson',
    modifier            VARCHAR(64) NOT NULL DEFAULT 'richson',
    is_deleted          SMALLINT NOT NULL DEFAULT 0
);

-- query: latest analysis per asset
CREATE INDEX idx_rsa_asset_latest
    ON rs_asset_analyses (asset_code, analyzed_at DESC)
    WHERE is_deleted = 0;

-- query: score history for trend line
CREATE INDEX idx_rsa_asset_date
    ON rs_asset_analyses (asset_code, analyzed_at)
    WHERE is_deleted = 0;

-- sequence start
ALTER SEQUENCE rs_asset_analyses_asset_analysis_id_seq RESTART WITH 100000;
```

#### rs_asset_analysis_dimensions

存储每个维度的子指标明细。一条 rs_asset_analyses 记录对应 N 条子指标记录。

```sql
CREATE TABLE rs_asset_analysis_dimensions (
    id                  BIGSERIAL PRIMARY KEY,
    asset_analysis_id   BIGINT NOT NULL REFERENCES rs_asset_analyses(asset_analysis_id),
    dimension           VARCHAR(8) NOT NULL,
    sub_indicator       VARCHAR(64) NOT NULL,
    raw_value           DECIMAL(20,6),
    percentile_1y       DECIMAL(5,2),
    percentile_5y       DECIMAL(5,2),
    blended_percentile  DECIMAL(5,2),
    normalized_score    DECIMAL(5,2),
    weight_in_dimension DECIMAL(4,2),
    data_source         VARCHAR(32),
    data_as_of          DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator             VARCHAR(64) NOT NULL DEFAULT 'richson',
    modifier            VARCHAR(64) NOT NULL DEFAULT 'richson',
    is_deleted          SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rsad_analysis
    ON rs_asset_analysis_dimensions (asset_analysis_id)
    WHERE is_deleted = 0;
```

#### rs_analysis_jobs

异步任务追踪表。

```sql
CREATE TABLE rs_analysis_jobs (
    job_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_code          VARCHAR(32) NOT NULL,
    job_type            VARCHAR(32) NOT NULL DEFAULT 'asset_analysis',
    status              VARCHAR(16) NOT NULL DEFAULT 'pending',
    progress            DECIMAL(4,2) NOT NULL DEFAULT 0,
    current_step        VARCHAR(64),
    steps               JSONB NOT NULL DEFAULT '[]',
    error_message       TEXT,
    error_code          VARCHAR(64),

    -- result reference
    asset_analysis_id   BIGINT,

    -- timing
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '1 hour'),
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,

    -- metadata
    request_id          UUID,
    locale              VARCHAR(8),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator             VARCHAR(64) NOT NULL DEFAULT 'richson',
    modifier            VARCHAR(64) NOT NULL DEFAULT 'richson',
    is_deleted          SMALLINT NOT NULL DEFAULT 0
);

-- prevent duplicate concurrent jobs per asset
CREATE UNIQUE INDEX uq_rsj_asset_active
    ON rs_analysis_jobs (asset_code)
    WHERE status IN ('pending', 'running') AND is_deleted = 0;

-- query: job status lookup
CREATE INDEX idx_rsj_status
    ON rs_analysis_jobs (status)
    WHERE is_deleted = 0;

-- cleanup: find expired jobs
CREATE INDEX idx_rsj_expires
    ON rs_analysis_jobs (expires_at)
    WHERE status IN ('pending', 'running') AND is_deleted = 0;
```

#### rs_event_alerts

事件概率变动监控表（PRD SS3.6）。

```sql
CREATE TABLE rs_event_alerts (
    id                  BIGSERIAL PRIMARY KEY,
    event_slug          VARCHAR(128) NOT NULL,
    event_title         TEXT NOT NULL,
    source              VARCHAR(32) NOT NULL DEFAULT 'polymarket',
    prev_probability    DECIMAL(5,4) NOT NULL,
    curr_probability    DECIMAL(5,4) NOT NULL,
    delta               DECIMAL(5,4) NOT NULL,
    threshold           DECIMAL(5,4) NOT NULL DEFAULT 0.20,
    gold_direction       VARCHAR(16),
    alerted             BOOLEAN NOT NULL DEFAULT FALSE,
    detected_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator             VARCHAR(64) NOT NULL DEFAULT 'richson',
    modifier            VARCHAR(64) NOT NULL DEFAULT 'richson',
    is_deleted          SMALLINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX uq_rsea_slug_active
    ON rs_event_alerts (event_slug)
    WHERE is_deleted = 0 AND alerted = FALSE;

CREATE INDEX idx_rsea_unalerted
    ON rs_event_alerts (alerted)
    WHERE alerted = FALSE AND is_deleted = 0;
```

#### rs_dimension_definitions

维度权重配置表（PRD SS3.3.1）。权重可后台调整，不需要发版。每个标的类型的维度集合和权重独立管理。

```sql
CREATE TABLE rs_dimension_definitions (
    id                  BIGSERIAL PRIMARY KEY,
    asset_type          VARCHAR(32) NOT NULL,
    dimension           VARCHAR(8) NOT NULL,
    name_zh             VARCHAR(32) NOT NULL,
    name_en             VARCHAR(32) NOT NULL,
    weight              DECIMAL(4,2) NOT NULL,
    description_zh      TEXT,
    description_en      TEXT,
    display_order       SMALLINT NOT NULL DEFAULT 0,
    model_version       VARCHAR(32) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator             VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier            VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted          SMALLINT NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX uq_rsdd_type_dim_version
    ON rs_dimension_definitions (asset_type, dimension, model_version)
    WHERE is_deleted = 0;
```

种子数据（黄金四维，model_version = 'gold_v1.0'）：

| asset_type | dimension | name_zh | weight | display_order |
|-----------|-----------|---------|--------|---------------|
| gold | D1 | 宏观利率 | 0.30 | 1 |
| gold | D2 | 美元流动性 | 0.25 | 2 |
| gold | D3 | 结构性需求 | 0.25 | 3 |
| gold | D4 | 技术位置 | 0.20 | 4 |

微调约束：单维度权重变更幅度不超过 +/-10%，四维之和始终 100%。权重变更时递增 `model_version`。

种子数据通过 richson 的 Alembic migration 初始化。后续权重调整通过 richson CLI 命令 `python -m richson.cli update-weights --asset-type gold --d1 0.30 --d2 0.25 --d3 0.25 --d4 0.20 --version gold_v1.1` 执行，自动校验权重之和为 100% 且变更幅度合规。

### 6.3 跨服务表依赖

| richson 读取的 rm_* 表 | 用途 |
|------------------------|------|
| rm_asset_catalog | 获取标的元信息（code, name, asset_type, exchange） |
| rm_holdings | 批量分析时获取活跃标的列表（用于 cron 触发） |

| richman 读取的 rs_* 表 | 用途 |
|------------------------|------|
| rs_asset_analyses | 标的详情页展示、投研简报数据、评分趋势线 |
| rs_asset_analysis_dimensions | 标的详情页维度展开面板 |
| rs_analysis_jobs | 前端查询分析进度 |
| rs_event_alerts | 触发事件通知 |

### 6.4 数据库用户权限隔离（生产环境）

| DB 用户 | rm_* 表权限 | rs_* 表权限 |
|---------|-------------|-------------|
| richman_user | SELECT, INSERT, UPDATE, DELETE | SELECT + 下列例外 |
| richson_user | SELECT | SELECT, INSERT, UPDATE, DELETE |

**richman 对 rs_* 表的写入例外**（详见 richman-backend-v2-trd.md SS6.1 和 SS8.5）：

| 表 | richman 写入的列 | 原因 |
|----|-----------------|------|
| rs_event_alerts | alerted | richman 标记已发送通知的事件，避免 richson 感知通知投递逻辑 |
| rs_analysis_jobs | status, error_message, error_code, updated_at, modifier | 过期 job 清理由 richman cron 统一执行 |

生产环境通过列级 GRANT 限制 richman_user 仅更新上述列。开发环境使用同一用户简化配置。生产环境通过环境变量注入不同的 DATABASE_URL。

## 7. 量化评分引擎（Layer 1）

### 7.1 评分流水线

```
数据获取 -> 频率对齐 -> 百分位排名 -> 维度评分 -> ATR 权重调制 -> 综合评分
```

每个标的类型注册一个 AssetAnalyzer 实现。MVP 仅实现 `GoldAnalyzer`。

```python
class AssetAnalyzer(Protocol):
    """Interface for per-asset-type scoring."""
    asset_type: str
    dimensions: list[DimensionDef]

    async def fetch_data(self, asset_code: str) -> DataSnapshot: ...
    async def compute_scores(self, snapshot: DataSnapshot) -> DimensionScores: ...
```

### 7.2 双窗口混合百分位算法

所有子指标的归一化使用以下确定性算法（PRD SS3.3.2）：

```python
def blended_percentile(
    current_value: float,
    history_1y: pd.Series,
    history_5y: pd.Series,
    invert: bool = False,
) -> float:
    """
    Compute dual-window blended percentile.

    Args:
        current_value: the latest observation
        history_1y: daily series for past 1 year
        history_5y: daily series for past 5 years
        invert: if True, lower values are more bullish (e.g., TIPS, DXY)

    Returns:
        Blended percentile 0-100, where 100 = most bullish for gold.
    """
    pct_1y = percentile_rank(current_value, history_1y)
    pct_5y = percentile_rank(current_value, history_5y)
    blended = 0.70 * pct_1y + 0.30 * pct_5y
    return (100.0 - blended) if invert else blended
```

冷启动处理：历史数据不足 1 年时使用可用数据范围的百分位（标注 `data_coverage: partial`），不足 90 天时该子指标标记为 `unavailable`。

### 7.3 D4 ATR 动态权重调制

D4 技术位置维度内的子指标权重会根据 ATR(14) 的历史百分位动态调整（PRD SS3.3.6）：

| ATR 百分位 | 体制 | 权重调整 |
|-----------|------|----------|
| > P75（5年） | 趋势市场 | RSI +5%, 均线交叉 +5%, 唐奇安 +5%, 比价 -10%, ATR -5% |
| < P25（5年） | 震荡市场 | 比价 +5%, RSI +5%, 均线交叉 -5%, 唐奇安 -5% |
| P25-P75 | 正常 | 使用默认权重 |

调整后各子指标权重之和仍为 100%。

### 7.4 LLM 定性判断到数值映射

Layer 2 LLM 输出结构化定性判断，由确定性规则转换为数值调整（PRD SS3.4）：

```python
ADJUSTMENT_MAP: dict[tuple[str, str], tuple[float, float]] = {
    # (magnitude, confidence): (min_adjustment, max_adjustment)
    ("major", "high"):    (12.0, 15.0),
    ("major", "medium"):  (8.0, 11.0),
    ("moderate", "high"): (6.0, 8.0),
    ("moderate", "medium"): (3.0, 5.0),
    ("minor", "high"):    (1.0, 2.0),
    ("minor", "medium"):  (1.0, 2.0),
    ("minor", "low"):     (1.0, 2.0),
}
# direction: "bullish" -> positive, "bearish" -> negative, "neutral" -> 0
# magnitude "major" + confidence "low" -> capped at "moderate"/"medium" range
# single-source major events -> capped at "moderate" range (PRD SS3.1)
```

映射取区间中点值。绝对值上限 15 分。多事件叠加时取所有事件调整值之和，仍受 15 分上限约束。

### 7.5 综合置信度计算

```python
def compute_confidence(
    d_scores: list[float],
    data_completeness: dict[str, bool],
    llm_available: bool,
) -> float:
    """
    Compute overall confidence (PRD SS3.5).

    Base confidence from dimension direction agreement:
    - All 4 agree (all >= 50 or all < 50): 80-100%
    - 3 agree, 1 diverge: 60-80%
    - 2v2 split: 40-60%
    - 3+ diverge: 20-40%

    Deductions:
    - FRED delay > 3 days: -15%
    - Polymarket unavailable: -5%
    - LLM failed: -10%
    """
    bullish_count = sum(1 for s in d_scores if s >= 50)
    bearish_count = 4 - bullish_count

    if bullish_count == 4 or bearish_count == 4:
        base = 90.0
    elif bullish_count == 3 or bearish_count == 3:
        base = 70.0
    elif bullish_count == 2:
        base = 50.0
    else:
        base = 30.0

    if not data_completeness.get("fred_fresh", True):
        base -= 15.0
    if not data_completeness.get("polymarket", True):
        base -= 5.0
    if not llm_available:
        base -= 10.0

    return max(0.0, min(100.0, base))
```

### 7.6 支撑/阻力位计算

```python
def compute_support_resistance(
    ohlcv: pd.DataFrame,
    sma200: float,
) -> tuple[list[float], list[float]]:
    """
    Support levels (PRD SS5.2.3):
    - 20-day Donchian channel lower band
    - Most recent significant low in past 60 days (rebound > 3%)
    - 200-day SMA
    Pick the one closest to current price.

    Resistance levels:
    - 20-day Donchian channel upper band
    - Most recent significant high in past 60 days
    - All-time high
    Pick the one closest to current price.
    """
```

### 7.7 历史回撤计算

```python
def compute_drawdown_reference(
    ohlcv: pd.DataFrame,
) -> dict:
    """
    PRD SS5.2.3: current bull-run max drawdown and historical comparison.

    Algorithm:
    1. Identify current bull-run start: most recent 20% drawdown from any
       prior peak, or series start if no such drawdown exists.
    2. Compute max drawdown within current bull-run:
       peak = rolling max of close; drawdown = (close - peak) / peak.
    3. Compute historical average max drawdown across all completed bull-runs
       in the full OHLCV history.

    Returns: {
        "currentBullRunStart": "2024-10-15",
        "maxDrawdown": -0.085,         # -8.5%
        "maxDrawdownDate": "2026-02-15",
        "historicalAvgDrawdown": -0.12  # -12%
    }
    """
```

回撤数据写入 rs_asset_analyses 的 `analysis_metadata` JSONB 字段（key: `drawdown_reference`），通过 GET /api/v2/market/{code} 返回给前端。

### 7.8 冲突检测

```python
def detect_conflict(d_scores: list[float]) -> tuple[str | None, str | None]:
    """
    PRD SS3.4 conflict detection:
    - Strong conflict: any two dimensions where one >= 70 and another <= 30
    - Weak conflict: max - min > 40, but no strong conflict
    Returns (conflict_type, conflict_message).
    """
```

## 8. Google ADK Agent 设计

### 8.1 Agent 架构

richson 使用 Google ADK 创建三个专用 Agent，对应 PRD SS3.10 的三次 LLM 调用：

```
research_agent      -> Layer 2: 信息检索 + 研判（D1/D2/D3 各调用一次）
interpretation_agent -> Layer 3: 综合解读文本生成
execution_agent     -> Layer 3: 执行计划生成
```

每个 Agent 独立定义 name、instruction、tools、output_schema（ADK 参数名为 `instruction` 单数形式）。Agent 之间不共享状态——通过 pipeline.py 按顺序编排，前一个 Agent 的输出作为后一个的输入。Agent 执行通过 `InMemoryRunner` 驱动（非 Agent 直接调用）。

### 8.2 research_agent

功能：接收维度名称、检索策略、当前量化分数，通过联网搜索获取实时信息，输出结构化定性判断。

Tools（from `google.adk.tools`）:
- `google_search`：ADK 内置 Google Search tool
- `load_web_page`：ADK 内置网页内容读取 tool（跟进具体 URL）

Output Schema（per dimension）：
```python
class ResearchResult(BaseModel):
    dimension: str  # "D1" | "D2" | "D3"
    events: list[ResearchEvent]
    judgment: QualitativeJudgment

class ResearchEvent(BaseModel):
    source_url: str
    source_name: str
    date: str
    summary: str

class QualitativeJudgment(BaseModel):
    direction: Literal["bullish", "bearish", "neutral"]
    magnitude: Literal["major", "moderate", "minor"]
    confidence: Literal["high", "medium", "low"]
    rationale: str
```

验证规则（PRD SS3.1）：
- `events` 列表中无 `source_url` 的事件自动过滤
- `magnitude` 为 `major` 时检查独立信源数 >= 2，不满足则降为 `moderate`
- 异常检测（PRD SS3.1）：当 LLM 调整后某维度 `abs(llm_adjustment)` > 10 时，richson 在该维度的 API 响应中附加 `llmAnomalyFlag: true`（运行时计算，不入库）。前端据此展示"此调整基于最新信息，尚待多源验证"提示
- D4 不调用 research_agent

### 8.3 interpretation_agent

功能：接收四维最终得分和 LLM 研判摘要，生成综合解读文本。

Output Schema：
```python
class InterpretationResult(BaseModel):
    market_interpretation: str  # 100-200 chars
    risk_factors: list[str]     # 2-3 items, each 30-50 chars
    regime_summary: str         # one sentence
    major_change_recap: str | None = None  # only when |score_delta| > 20 (PRD SS3.4)
```

内容风格通过 system prompt 控制（PRD SS13.3）：专业、克制、有主见，先结论后原因。

### 8.4 execution_agent

功能：基于标的评分、用户持仓、风险偏好生成条件分支执行计划。

输入除评分数据外，还包括：
- 用户持仓信息（costPrice、positionRatio，来自 richman 请求）
- 用户风险偏好对应的参数约束（PRD SS7.6 映射表）
- 同类标的总敞口
- 支撑/阻力位和 ATR 值（richson 计算）

设计决策（PRD SS13.4 vs SS3.10 矛盾）：PRD SS13.4 写"持仓级 LLM 调用仅发送标的代码和评分，不发送用户成本和仓位"，但 SS3.10 调用三明确要求输入"用户持仓信息（标的代码、仓位比例、成本价）"用于生成精准的条件分支执行计划（止损位、加仓量依赖成本和仓位）。本 TRD 按 SS3.10 设计，持仓数据仅在当次 LLM 请求中使用，richson 不持久化用户持仓信息。PRD SS13.4 该条款待产品侧修正。

Instructions 约束（写入 Agent system prompt，PRD SS8.1 + SS8.2）：

1. **分数 Kelly 准则**：单笔建仓/加仓不超过半 Kelly，零售投资者单笔不超过风险偏好对应上限（保守 2% / 稳健 5% / 进取 8%）
2. **金字塔建仓**：加仓分 2-3 批次，每批独立触发条件，后续批次仓位 <= 前一批
3. **止损优先**：止损/减仓场景的 priority 永远为 1，优先级高于所有加仓场景
4. **加仓后止损上移**：任一加仓场景执行后，止损位自动调整为新加仓的成本价（保本止损），scenarios 中需注明此规则
5. **同向场景互斥**：同一方向的场景不可叠加执行（如不能在支撑位加仓后又在阻力位加仓，除非执行计划已刷新）；同向加仓场景标注 `exclusion_group: "long_add"`，同向减仓标注 `exclusion_group: "long_reduce"`
6. **评分门槛**：综合评分 < 40 时不建议加仓。评分 >= 60 且无强冲突时，浮亏持仓仍可在支撑位加仓但附加风险提示："当前持仓浮亏 X%，加仓将摊低成本但增加敞口"
7. **ATR 止损**：止损位基于风险偏好对应的 ATR 倍数（保守 1.5x / 稳健 2x / 进取 3x）
8. **同类协调**：当 peerExposure + 建议加仓量 > 集中度蓝色阈值时，削减建议加仓量或标注集中度警告
9. **no_trigger_note 格式**：必须包含有效期天数和"维持现有仓位"的明确默认建议

Output Schema：
```python
class ExecutionPlan(BaseModel):
    action: str
    action_label: str
    default_action: str
    current_position: float
    target_position: float
    scenarios: list[Scenario]
    stop_loss: float
    take_profit: float
    valid_days: int = 7
    no_trigger_note: str

class Scenario(BaseModel):
    condition: str
    action: str
    lot_count: int
    rationale: str
    priority: int  # 1 = highest (stop-loss always priority 1)
    exclusion_group: str | None = None  # same group value = mutually exclusive (e.g. "long_add")
```

### 8.5 LLM Provider 初始化

richson 通过 ADK 的 LiteLlm 适配器支持多 provider：

```python
from google.adk.agents import Agent
from google.adk.models.lite_llm import LiteLlm
from google.adk.runners import InMemoryRunner

def create_agent(
    name: str,
    llm_config: LLMConfig,
    instruction: str,
    tools: list,
    output_schema: type[BaseModel] | None = None,
) -> Agent:
    """
    Create ADK agent with the given LLM configuration.

    llm_config.provider determines the LiteLlm model string:
    - "claude" -> LiteLlm(model="anthropic/claude-sonnet-4-20250514")
    - "openai" -> LiteLlm(model="openai/gpt-4o")
    - "openai_compatible" -> LiteLlm(model="openai/<model>", api_base=..., api_key=...)
    - "gemini" -> plain string "gemini-2.5-flash" (no LiteLlm wrapper needed)
    """
    model = _resolve_model(llm_config)  # returns str for Gemini, LiteLlm for others
    return Agent(
        name=name,
        model=model,
        instruction=instruction,
        tools=tools,
        output_schema=output_schema,
    )

async def run_agent(agent: Agent, user_input: str) -> dict:
    """Execute agent via InMemoryRunner and return structured output."""
    runner = InMemoryRunner(agent=agent, app_name="richson")
    # runner.run_async yields events; collect final output
    ...
```

LLM API Key 传递：richman 在调用 richson API 时明文传入 `llmConfig.apiKey`（richman 从 rm_llm_configs 解密后传递）。richson 不存储 key，仅在内存中用于当次请求。

平台配额 key 通过 richson 环境变量 `PLATFORM_LLM_API_KEY` 注入，用于标的级分析。

### 8.6 降级策略

| 失败场景 | 降级行为 |
|----------|----------|
| research_agent 超时/失败 | Layer 2 跳过，LLM 调整值为 0，标注 `llm_skipped: true` |
| interpretation_agent 失败 | 使用内置模板生成解读文本（`templates/interpretation_{locale}.py`） |
| execution_agent 失败 | 返回错误，richman 展示"执行计划暂时不可用" |
| 单维度数据源失败 | 该维度标记 `unavailable`，总分基于剩余维度重新加权 |
| Polymarket 不可用 | D1 降息概率子指标权重归零（其余 D1 子指标重新归一化），D3 地缘事件子指标权重归零（同上），事件雷达降级为公开经济日历。置信度扣 5%。替代数据源：D1 可由 CME FedWatch 替代（PRD SS12.3），D3 地缘概率由 LLM 联网搜索补充，事件雷达可用 Kalshi 或公开经济日历 API 替代 |
| 全部数据源失败 | job 标记 `failed`，不生成分析记录 |

模板生成的文本附加标记 `generated_by = 'l1_only'`（DB）/ `"generatedBy": "l1_only"`（API），前端据此展示"模板生成"标签。

## 9. richman 侧 v2 API 新增与变更

richman 作为前端唯一的 API 网关，前端不直连 richson。以下端点均在 richman 侧实现，部分代理到 richson。

### 9.1 新增端点

| 方法 | 路径 | 认证 | 说明 | 数据来源 |
|------|------|------|------|----------|
| GET | /api/v2/market/regime | 无 | 市场体制判断 | 代理 richson GET /market/regime |
| GET | /api/v2/market/overview | 无 | 标的卡片墙数据 | richman 聚合 rm_asset_catalog + rs_asset_analyses |
| GET | /api/v2/market/{code} | 无 | 标的详情核心数据 | richman 读 rs_asset_analyses + rs_asset_analysis_dimensions |
| GET | /api/v2/market/{code}/ohlcv | 无 | K 线数据 | 代理 richson GET /market/ohlcv/{code} |
| GET | /api/v2/market/{code}/scores | 无 | 评分趋势线 | 代理 richson GET /assets/{code}/score-history |
| GET | /api/v2/events/radar | 无 | 事件雷达 | 代理 richson GET /events/radar |
| POST | /api/v2/analysis/trigger-asset | JWT | 手动触发标的分析 | 代理 richson POST /jobs/analyze-asset |
| GET | /api/v2/analysis/jobs/{jobId} | JWT | 查询 job 进度 | richman 直读 rs_analysis_jobs |
| POST | /api/v2/analysis/holding/{holdingId} | JWT | 持仓级分析 | richman 调 richson POST /analyze/holding，持久化结果 |
| GET | /api/v2/market/{code}/demo-plan | 无 | 示范执行计划 | richman 从 rs_asset_analyses.demo_plan 预计算字段读取（见 richman-backend-v2-trd SS4.2）；fallback: 代理 richson POST /analyze/demo-plan |
| GET | /api/v2/briefing | JWT | 投研简报 | richman 聚合 rm_holdings + rs_asset_analyses |
| POST | /api/v2/feedback | JWT | 用户反馈（PRD SS6.3） | richman 写 rm_user_feedback |
| PATCH | /api/v2/user/risk-preference | JWT | 设置风险偏好 | richman 更新 rm_users.risk_preference |

### 9.1.1 核心端点响应结构

#### GET /api/v2/market/{code} 响应

```json
{
  "data": {
    "assetCode": "GLD",
    "name": "SPDR Gold Shares",
    "nameZh": "SPDR 黄金 ETF",
    "assetType": "gold",
    "currency": "USD",
    "currentPrice": 4750.00,
    "changePercent": 1.2,
    "analysis": {
      "assetAnalysisId": 100042,
      "overallScore": 72.5,
      "signalLevel": "moderate_bullish",
      "confidence": 75.0,
      "confidenceBandLow": 65.0,
      "confidenceBandHigh": 79.0,
      "percentileLabel": "近一年中高",
      "modelVersion": "gold_v1.0",
      "marketInterpretation": "...",
      "riskFactors": ["...", "..."],
      "regimeSummary": "...",
      "conflictType": "strong",
      "conflictMessage": "宏观利率(30)与结构性需求(85)方向冲突",
      "scoreDelta": 7.0,
      "changeSummary": "D3+5(央行购金报道) D2+3(DXY走弱)",
      "majorChangeRecap": null,
      "usdExchangeRate": null,
      "priceAtAnalysis": 4680.00,
      "analyzedAt": "2026-04-13T06:00:00Z",
      "generatedBy": "full",
      "llmSkipped": false,
      "drawdownReference": {
        "currentBullRunStart": "2024-10-15",
        "maxDrawdown": -0.085,
        "maxDrawdownDate": "2026-02-15",
        "historicalAvgDrawdown": -0.12
      },
      "demoPlan": { "..." : "..." },
      "dimensions": [
        {
          "dimension": "D1",
          "nameZh": "宏观利率",
          "nameEn": "Macro Rates",
          "score": 65.0,
          "baseScore": 60.0,
          "llmAdjustment": 5.0,
          "weight": 0.30,
          "subIndicators": [
            {
              "name": "10Y TIPS Yield",
              "rawValue": 1.85,
              "percentile1y": 45.0,
              "percentile5y": 55.0,
              "blendedPercentile": 48.0,
              "normalizedScore": 52.0,
              "weightInDimension": 0.35,
              "dataSource": "FRED",
              "dataAsOf": "2026-04-12"
            }
          ]
        }
      ]
    }
  }
}
```

#### GET /api/v2/market/overview 响应

```json
{
  "data": {
    "groups": [
      {
        "category": "commodity",
        "categoryLabel": "商品",
        "assets": [
          {
            "assetCode": "GLD",
            "name": "SPDR Gold Shares",
            "nameZh": "SPDR 黄金 ETF",
            "assetType": "gold",
            "currency": "USD",
            "isActive": true,
            "currentPrice": 4750.00,
            "changePercent": 1.2,
            "overallScore": 72.5,
            "signalLevel": "moderate_bullish",
            "percentileLabel": "近一年中高"
          }
        ]
      }
    ]
  }
}
```

richman 从 rm_asset_catalog（标的元信息）+ rs_asset_analyses（最新分析记录，仅 isActive 标的）聚合。置灰标的返回 `isActive: false`，无 score/signal 字段。

`percentileLabel` 由 richman 根据综合评分在最近一年 rs_asset_analyses 记录中的百分位计算：P90+ -> "近一年偏高"，P75-89 -> "近一年中高"，P25-74 -> "近一年中位"，P10-24 -> "近一年中低"，P10以下 -> "近一年偏低"（PRD SS3.4）。

#### GET /api/v2/briefing 响应

```json
{
  "data": {
    "cards": [
      {
        "holdingId": 100023,
        "assetCode": "GLD",
        "assetName": "SPDR Gold Shares",
        "costPrice": 4500.00,
        "positionRatio": 8.0,
        "profitLoss": 250.00,
        "profitLossPercent": 5.56,
        "overallScore": 72.5,
        "signalLevel": "moderate_bullish",
        "scoreDelta": 7.0,
        "changeSummary": "D3+5(央行购金报道) D2+3(DXY走弱)",
        "conflictType": null,
        "conflictMessage": null,
        "actionSummary": "逢回调加仓",
        "concentrationLevel": "blue",
        "concentrationMessage": "黄金配置已达 12%",
        "sparklineScores": [68.0, 69.5, 72.5],
        "analyzedAt": "2026-04-13T06:00:00Z"
      }
    ]
  }
}
```

#### POST /api/v2/feedback 请求/响应

请求：
```json
{
  "assetAnalysisId": 100042,
  "rating": "helpful",
  "comment": "金价没跌到支撑位就反弹了"
}
```

`rating`：`helpful` | `not_helpful`。`comment` 可选，最大 500 字符。

响应（201）：
```json
{
  "data": { "feedbackId": 1001 }
}
```

richman 将反馈写入 rm_user_feedback 表（richman migration 022 新建）：

```sql
CREATE TABLE rm_user_feedback (
    feedback_id         BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL,
    asset_analysis_id   BIGINT NOT NULL,
    rating              VARCHAR(16) NOT NULL,
    comment             TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator             VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier            VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted          SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmuf_user ON rm_user_feedback (user_id) WHERE is_deleted = 0;
ALTER SEQUENCE rm_user_feedback_feedback_id_seq RESTART WITH 100000;
```

同一 migration 中将 rm_users 已有的 `risk_preference` 列默认值从 `'neutral'` 改为 `'moderate'`，CHECK 约束更新为 `conservative/moderate/aggressive`（PRD SS7.6）。该列由 migration 007 创建，022 仅变更枚举值。

MVP 阶段反馈仅存储，不用于模型优化。

### 9.2 v1 端点保留与废弃

| v1 端点 | v2 状态 | 说明 |
|---------|---------|------|
| /api/v1/analysis/trigger | 废弃 | 替代: /api/v2/analysis/trigger-asset |
| /api/v1/tasks/:taskId | 废弃 | 替代: /api/v2/analysis/jobs/{jobId} |
| /api/v1/decision-cards/* | 保留但冻结 | v1 历史卡片只读，新卡片走 v2 |
| /api/v1/holdings/* | 保留 | 持仓 CRUD 逻辑不变，表名更新为 rm_holdings |
| /api/v1/auth/* | 保留 | 认证逻辑不变 |
| /api/v1/assets/* | 保留 | 标的目录查询不变 |

### 9.3 richman 定时任务

| 任务 | 触发时间 | 动作 |
|------|----------|------|
| 每日标的分析 | 美股收盘后（UTC+8 06:00） | 调用 richson POST /jobs/batch-analyze，传入所有激活标的 |
| 每日持仓分析 | UTC+8 07:30 | 对所有活跃持仓调用 richson POST /analyze/holding，持久化决策卡片，检测执行计划变化并推送通知（详见 richman-backend-v2-trd.md SS8.3.1） |
| 每日简报邮件 | UTC+8 08:30 | 从 rs_asset_analyses 提取最新数据，生成邮件内容，推送给所有注册用户（PRD SS10.3） |
| A 股收盘后快讯 | UTC+8 15:30（工作日） | 检查 A 股相关标的评分变化，推送收盘快讯（详见 richman-backend-v2-trd.md SS8.3.2） |
| 每周投研洞察 | 每周一 UTC+8 08:30 | 调用 richson POST /content/weekly-insight 生成周报文本，推送邮件（PRD SS10.3） |
| 事件告警轮询 | 每小时 | 读取 rs_event_alerts WHERE alerted = FALSE，触发通知后标记 alerted = TRUE |
| 过期 job 清理 | 每 10 分钟 | UPDATE rs_analysis_jobs SET status = 'failed' WHERE expires_at < NOW() AND status IN ('pending', 'running') |

注意：richman 负责 cron 调度和通知推送，richson 不含定时任务逻辑（除事件概率监控外）。事件监控的定时拉取由 richson 内部 `asyncio` scheduler 完成（每小时拉取 Polymarket 数据，比较概率变动，超阈值写入 rs_event_alerts）。

每日简报邮件内容由 richman 从预计算数据中提取，不需要额外 LLM 调用。邮件包含：市场体制 + 黄金评分（较昨日变化）+ 今日关注事件 + 持仓建议摘要（如有持仓）。即使无变化也发送——"今天不需要操作"本身就是有价值的信息（PRD SS10.3）。

每周投研洞察需要一次 LLM 调用生成 300-500 字的周报内容。richson 新增端点：

#### POST /content/weekly-insight

请求：
```json
{
  "locale": "zh",
  "llmConfig": { "provider": "claude", "model": "...", "apiKey": "..." },
  "requestId": "..."
}
```

响应（200，超时 30s）：
```json
{
  "data": {
    "weeklyReview": "上周黄金...",
    "weeklyOutlook": "本周关注...",
    "educationTopic": "什么是金银比？...",
    "locale": "zh"
  }
}
```

## 10. 数据源缓存策略

richson 内部维护 per-source 的内存缓存（基于 `cachetools.TTLCache`），避免重复调用外部 API：

| 数据源 | 缓存 TTL | 说明 |
|--------|----------|------|
| FRED | 1 小时 | 宏观数据日级更新，小时级缓存足够 |
| Yahoo Finance（价格） | 5 分钟 | 盘中需要较新价格 |
| Yahoo Finance（ETF 持仓） | 1 小时 | 日级更新 |
| AKShare | 5 分钟 | A 股盘中价格 |
| Polymarket | 15 分钟 | 事件概率变动频繁但不需要秒级 |
| CFTC COT | 24 小时 | 周度更新 |
| WGC（央行购金 + AISC） | 30 天 | 季度更新，详见下方说明 |

**D3 央行购金 + AISC 数据源设计（PRD SS3.3.5）**：

世界黄金协会（WGC）无公开 API，D3 的两个 WGC 子指标（央行净购买量 40%、AISC 利润率 10%）采用以下方案：

1. **初始数据**：通过 `wgc.py` 从 WGC 公开报告页面解析最新季报数据（央行年化购买量吨数、行业平均 AISC 值）。解析失败时回退到手动维护的 `config/wgc_quarterly.json` 种子文件
2. **更新频率**：季度更新。两次季报之间，子指标得分保持不变（PRD SS3.3.5："央行购金数据为季度更新，两次季报之间保持上期值"）
3. **LLM 补偿**：Layer 2 research_agent 在 D3 维度检索时主动搜索"央行购金"新闻，发现月度/周度购金动态后通过 LLM 调整弥补季报空窗期的时效性
4. **AISC 利润率计算**：`(当前金价 - AISC) / AISC`，AISC 取最新季报值，金价取 Yahoo Finance 实时价

缓存仅为进程内 dict，richson 重启后自动失效。不引入 Redis -- MVP 阶段数据量小，进程内缓存足够。

## 11. 可观测性

### 11.1 日志规范

richson 使用 structlog 输出 JSON 格式日志，字段对齐 richman 的 zap 格式：

```json
{
  "ts": "2026-04-13T10:00:00.000Z",
  "level": "info",
  "msg": "analysis job started",
  "request_id": "550e8400-...",
  "job_id": "a1b2c3d4-...",
  "asset_code": "GLD",
  "step": "layer1_scoring"
}
```

关键日志点：
- job 创建 / 开始 / 每 step 开始与结束 / 完成 / 失败
- 外部数据源调用（请求 URL + 耗时 + 状态码）
- LLM 调用（provider + model + token count + 耗时 + 成功/失败）
- 降级触发（哪个数据源/LLM 不可用，采用的降级路径）

### 11.2 监控告警

通过日志级别驱动外部监控采集：

| 条件 | 日志级别 | 含义 |
|------|----------|------|
| job pending > 5 min | WARN | 可能卡住，需排查 |
| job running > 30 min | WARN | 超时风险 |
| LLM 调用连续失败 >= 3 次 | ERROR | LLM 服务可能不可用 |
| FRED 数据延迟 > 3 天 | WARN | 影响评分精度 |
| 数据源 HTTP 5xx | WARN | 外部服务异常 |

### 11.3 docker-compose healthcheck

```yaml
richson:
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8001/health"]
    interval: 30s
    timeout: 10s
    retries: 3
    start_period: 15s
```

## 12. 部署配置

### 12.1 docker-compose 扩展

在现有 docker-compose.yml 中新增 richson 服务：

```yaml
services:
  richson:
    build:
      context: ./richson
      dockerfile: Dockerfile
    ports:
      - "8001:8001"
    environment:
      - DATABASE_URL=postgresql+asyncpg://richson_user:${DB_PASSWORD}@postgres:5432/richman
      - INTERNAL_API_KEY=${INTERNAL_API_KEY}
      - PLATFORM_LLM_API_KEY=${PLATFORM_LLM_API_KEY}
      - FRED_API_KEY=${FRED_API_KEY}
      - LOG_LEVEL=info
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped
```

### 12.2 环境变量

richson `.env.example`：

```env
# Database
DATABASE_URL=postgresql+asyncpg://richson_user:password@localhost:5432/richman

# Internal service auth
INTERNAL_API_KEY=change-me-in-production

# Platform LLM key (for asset-level analysis)
PLATFORM_LLM_API_KEY=sk-...

# FRED API
FRED_API_KEY=...

# Server
HOST=0.0.0.0
PORT=8001
LOG_LEVEL=info
WORKERS=1
```

### 12.3 richman 侧新增配置

richman `.env` 新增：

```env
# richson service URL
RICHSON_BASE_URL=http://localhost:8001
RICHSON_API_KEY=change-me-in-production

# richson call timeouts
RICHSON_ASYNC_TIMEOUT_MS=5000
RICHSON_SYNC_TIMEOUT_MS=30000
RICHSON_LIGHT_TIMEOUT_MS=10000
```

## 13. 数据生命周期

### 13.1 rs_asset_analyses 保留策略

- 最近 365 天：保留全部记录
- 365 天 - 5 年：每周保留一条（周一的记录），其余软删除
- 5 年以上：全部软删除

由 richson 定时任务（每周日 UTC+8 03:00）执行清理。

### 13.2 rs_analysis_jobs 保留策略

- completed 状态：90 天后软删除
- failed 状态：30 天后软删除
- pending/running 且超过 expires_at：由 richman 过期 job 清理任务标记为 failed

### 13.3 冷启动 bootstrapping

首次部署时历史数据不足以计算百分位。bootstrapping 策略：

1. richson 提供 CLI 命令 `python -m richson.cli backfill --days 90`
2. 使用历史行情数据回算近 90 天的每日指标快照
3. 写入 rs_asset_analyses，标记 `source = 'backfill'`
4. 90 天后系统积累足够真实数据，backfill 记录自然进入清理窗口

## 14. v1 -> v2 迁移策略

### 14.1 迁移阶段

```
Phase 0: 准备
  - richman migration 021: 所有存量表加 rm_ 前缀
  - richman migration 022: 新增 rm_user_feedback 表 + rm_users.risk_preference 枚举值变更 + 新列
  - richman sqlc 重新生成所有 query
  - richman 所有 SQL 引用更新为 rm_ 表名
  - 此阶段不影响功能，只是表名变更 + 新表

Phase 1: richson 上线（Phase 0 完成后执行）
  - richson 部署并创建 rs_* 表（Alembic migration）——与 Phase 0 的 rm_* 迁移无依赖（独立表），但建议在 Phase 0 之后执行以避免混淆
  - richson bootstrapping（backfill 90 天历史数据）
  - richman cron 开始调用 richson batch-analyze
  - richman v2 API 端点上线

Phase 2: 前端切换
  - 前端路由从 v1 切换到 v2
  - Market Overview 替代 onboarding 作为首页
  - 标的详情页、投研简报页上线

Phase 3: 清理
  - v1 analysis handler 废弃
  - richman 侧 LLM provider 代码标记 deprecated
  - rm_analysis_results 表冻结（只读，不再写入新数据）
  - v1 decision_cards 保留只读访问
```

### 14.2 richman 现有代码废弃清单

以下 richman 模块在 richson 上线后标记为 deprecated：

| 模块 | 当前职责 | v2 替代 |
|------|----------|---------|
| internal/analysis/ | LLM 调用、分析 pipeline | richson 全权承担 |
| internal/llm/ | 多 provider 抽象层 | richson ADK |
| internal/datasource/ | 金融数据获取 | richson datasources/ |

这些模块在 Phase 1 结束后不再接收新功能，Phase 3 时删除代码。

### 14.3 数据兼容性

- rm_analysis_results（v1）和 rs_asset_analyses（v2）是独立表，不做数据迁移
- rm_decision_cards（v1）保留只读访问，v2 决策卡片由 richman 基于 richson 的 holding 分析结果写入
- v1 analysis_tasks 和 v2 rs_analysis_jobs 是独立表，v2 上线后新任务全部走 rs_analysis_jobs

## 15. 风险偏好参数映射

richson 的 execution_agent 使用以下风险参数约束（PRD SS7.6）：

```python
RISK_PARAMS: dict[str, dict[str, float]] = {
    "conservative": {
        "max_single_add": 2.0,       # percent
        "stop_loss_atr_multi": 1.5,
        "concentration_blue": 10.0,
        "score_threshold_add": 70.0,
    },
    "moderate": {
        "max_single_add": 5.0,
        "stop_loss_atr_multi": 2.0,
        "concentration_blue": 15.0,
        "score_threshold_add": 60.0,
    },
    "aggressive": {
        "max_single_add": 8.0,
        "stop_loss_atr_multi": 3.0,
        "concentration_blue": 20.0,
        "score_threshold_add": 55.0,
    },
}

# Concentration thresholds are shared across all risk profiles (PRD SS8.3)
CONCENTRATION_THRESHOLDS: list[tuple[float, str, str]] = [
    (35.0, "red",    "黄金配置严重集中，强烈建议控制敞口"),
    (25.0, "orange", "黄金配置集中度较高，请注意分散风险"),
    (15.0, "blue",   "黄金配置已达 {pct}%，处于机构建议区间上限"),
]
# Evaluated top-down; first matching threshold wins.
# concentration_blue in RISK_PARAMS adjusts the blue-tier threshold per risk preference.
```

## 16. 重试与超时策略

### 16.1 richman 调用 richson

| 调用类型 | HTTP 超时 | 重试次数 | 重试间隔 | 降级 |
|----------|-----------|----------|----------|------|
| 异步触发 (POST /jobs/*) | 5s | 1 | 2s | 返回 502 给前端 |
| 同步分析 (POST /analyze/*) | 30s | 1 | 2s | 返回降级响应 |
| 轻量查询 (GET /market/*, /events/*) | 10s | 1 | 2s | 返回缓存或 502 |
| 健康检查 (GET /health) | 3s | 0 | - | 标记 richson unhealthy |

### 16.2 richson 调用外部数据源

| 数据源 | HTTP 超时 | 重试 | 降级 |
|--------|-----------|------|------|
| FRED | 10s | 2 | 使用缓存值 |
| Yahoo Finance | 10s | 2 | 使用 stooq fallback |
| AKShare | 10s | 2 | 使用缓存值 |
| Polymarket | 10s | 1 | 权重归零 |
| LLM (ADK) | 60s | 0 | Layer 2 跳过 / 模板生成 |

### 16.3 richson 内部 LLM 调用

LLM 调用不做自动重试——LLM 超时通常意味着 provider 过载，重试只会加重负载。失败时直接走降级路径。

## 17. 维度部分失败处理

当 4 个维度中部分失败时（PRD SS3.7）：

1. 失败维度标记 `status: "unavailable"`，该维度不参与总分计算
2. 剩余维度权重按原始比例重新归一化（如 D2 失败，D1/D3/D4 权重从 30/25/20 归一化为 40/33/27）
3. 总分仍映射为 0-100
4. 置信度额外扣减 15%（每个失败维度）
5. 前端展示"部分维度数据暂不可用"提示

最少需要 2 个维度可用才生成分析结果。不足 2 个维度时 job 标记为 `failed`。

## 18. 模型验证要求

MVP 上线前，黄金四维模型必须通过历史回算验证（PRD SS3.8）：

- 验证范围：2020-2025 年（覆盖 COVID 冲击、加息周期、降息周期、央行购金潮四个体制）
- 验证指标：综合评分与后续 1-3 个月金价走势的方向一致率（评分 > 60 时金价上涨，评分 < 40 时金价下跌）
- 最低标准：方向一致率 > 60%
- 执行方式：richson CLI 命令 `python -m richson.cli backtest --start 2020-01-01 --end 2025-12-31`
- 输出：逐月评分 + 实际金价变动 + 方向一致率统计，写入 `docs/validation/` 目录
- 验证结果用于内部模型校准，不面向用户展示
- 验证不通过则阻塞上线，需调整权重或子指标逻辑

回算与 §13.3 的 backfill 区别：backfill 是为生产环境冷启动填充 90 天近期数据；backtest 是用 5 年历史数据验证模型有效性，结果不写入生产表。

## 19. 模型版本管理

分析模型的权重和参数通过 rs_dimension_definitions 表管理，每次调整产生新的 `model_version`（PRD SS3.9）。

版本号格式：`{asset_type}_v{major}.{minor}`，如 `gold_v1.0`。

- 权重微调（单维度 +/-10% 以内）：minor 版本递增（如 `gold_v1.0` -> `gold_v1.1`）
- 维度增删或子指标变更：major 版本递增（如 `gold_v1.1` -> `gold_v2.0`）

版本变更流程：
1. 更新 rs_dimension_definitions 中对应 asset_type 的记录，设置新 model_version
2. 旧版本记录保留（is_deleted = 0），用于历史查询
3. 新分析自动使用最新 model_version
4. rs_asset_analyses 每条记录的 `model_version` 字段记录生成时的版本，保证历史可追溯
5. 评分趋势线 API（GET /assets/{code}/score-history）返回 `versionChanges` 列表，前端据此标注版本变更竖线

历史决策卡片永不重算——每张卡片保留生成时的 model_version 和评分，确保历史一致性。

变化摘要生成（change_summary）：Layer 3 interpretation_agent 在生成市场解读后，比较本次与上次各维度得分差异，对 `abs(delta) >= 3` 的维度按 delta 绝对值降序拼接，格式为 `D{n}{+/-delta}(原因短语)`，如 `D3+5(央行购金报道) D2+3(DXY走弱)`。原因短语由 LLM 根据 Layer 2 研究摘要生成，限 10 字以内。首次分析（无 prev_analysis_id）时 change_summary 为 NULL。

重大变化复盘（PRD SS3.4）：当 `score_delta` 绝对值 > 20（跨越一个信号级别）时，interpretation_agent 额外生成 `major_change_recap` 文本，包含上期判断核心依据、哪些假设被证伪、当前应如何调整。

## 20. 公开 API 防护

Market Overview 和标的详情页的公开端点（无 JWT）需要基本防护以防止滥用：

- richman 对公开端点实施 IP 级限流：单 IP 每分钟最多 60 次请求（Gin 中间件）
- richson 内部端点不需要额外限流（由 richman 网关控制）
- MVP 不实施用户级限流（登录端点已有 JWT 保护）
- 后续可通过 CDN 或 Cloudflare 增强防护

## 21. 已知问题与编码阶段必须处理项

以下问题已在设计审查中识别，必须在编码阶段解决，不可跳过。

### 21.1 richson 端口暴露

docker-compose 中 richson 使用 `ports: "8001:8001"` 将端口映射到宿主机。若 VPS 防火墙未阻断 8001，内部服务直接暴露公网。

处理方案：改为 `expose: [8001]`（仅 docker 内部网络可达）或绑定 `127.0.0.1:8001:8001`。`INTERNAL_API_KEY` 必须在部署时替换为强随机值，禁止使用默认值。

### 21.2 healthcheck 命令依赖

TRD SS11.3 的 healthcheck 使用 `curl -f`，但 Python alpine 镜像通常不含 curl。

处理方案：Dockerfile 中安装 curl，或改用 `python -c "import urllib.request; urllib.request.urlopen('http://localhost:8001/health')"`。

### 21.3 asyncio 事件循环阻塞风险

单进程 asyncio 中，pandas 的 CPU 密集计算（Layer 1）会阻塞事件循环，导致 07:30 的持仓分析同步端点超时。

处理方案：pandas 计算段使用 `asyncio.to_thread()` 或 `run_in_executor(ProcessPoolExecutor)` 卸载到子进程/线程。

### 21.4 rs_event_alerts 缺少 RESTART WITH

rs_asset_analysis_dimensions 和 rs_event_alerts 的序列未设置 `RESTART WITH 100000`，与其他表不一致。

处理方案：在 Alembic 迁移中补齐 `ALTER SEQUENCE ... RESTART WITH 100000`。

### 21.5 LLM 成本上限缺失

批量分析 + 周报 + demo plan 均消耗 LLM token，但无每日/每月预算上限。

处理方案：在 richson 配置中新增 `DAILY_LLM_BUDGET_USD` 环境变量，pipeline 中累计当日 token 消耗，超预算时跳过 Layer 2/3 降级到 l1_only 模式。日志中记录告警。

### 21.6 集中度警告文本硬编码中文

richson SS15 的 `CONCENTRATION_THRESHOLDS` 中阈值提示文本硬编码中文，未走 i18n。

处理方案：提示文本作为模板参数，由 richman 传入的 `language` 参数选择对应模板。或在 richson 的 `config/` 下按 locale 分文件存储。

### 21.7 backfill 数据标记

90 天 backfill 记录标记 `source='backfill'`，但 percentileLabel 计算和 sparkline 查询未排除 backfill 记录。

处理方案：评估 backfill 数据是否应参与百分位计算。若不应参与，查询中加 `WHERE source != 'backfill'` 过滤。若应参与（回测一致性），则无需处理但需在代码注释中明确说明设计决策。

### 21.8 validDays 下界约束缺失

SS6.2 分析请求中 `validDays` 参数默认值 7，但 richson 入参校验未定义下界。`validDays: 0` 或负数会导致 `expires_at` 早于 `created_at`，唯一索引立即释放但分析结果无法被前端缓存命中。

处理方案：richson FastAPI 端点使用 Pydantic `Field(ge=1, le=90)` 约束 `validDays`。richman 代理调用时也做前置校验（1-90），不合法值返回 400。

### 21.9 LLM apiKey 日志脱敏

richson 请求体中携带明文 `llmConfig.apiKey`（由 richman 解密后传入）。FastAPI 默认日志和异常 traceback 可能打印请求体，导致 API Key 泄露到日志文件。

处理方案：在 richson 的日志中间件中对请求体做脱敏处理，`apiKey` 字段替换为 `"sk-***"` 再写入日志。Pydantic model 的 `apiKey` 字段使用 `repr=False` 避免 repr 输出泄露。异常处理中禁止将原始请求体写入错误日志。
