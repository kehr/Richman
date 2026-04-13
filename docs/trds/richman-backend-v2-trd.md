# richman Go 后端 v2 TRD

> 版本 1.0 | 关联 PRD: docs/prds/richman-prd-v2.md | 关联 TRD: docs/trds/richson-service-trd.md

## 1. 文档范围

本 TRD 覆盖 v2 版本中 richman（Go 主服务）的全部后端变更，包含：

- richson HTTP 客户端设计（超时、重试、健康检查）
- v2 API handler 层（路由分组、代理/聚合两类 handler）
- v2 service 层新增与变更
- v2 repo 层新增（rs_* 只读访问、rm_user_feedback、rm_users 扩展列）
- 通知与邮件推送系统（HTML 模板、内容组装、推送时机）
- 定时任务（cron）重构（每日分析、简报邮件、周报、事件告警、过期清理）
- 数据库迁移脚本设计（021 rm_ 前缀、022 新表 + 新列）
- 配置变更（新环境变量、启动检查）
- v1 代码废弃计划

不在本 TRD 范围：richson Python 服务内部设计（见 richson-service-trd.md）、前端页面重构（见 frontend-v2-trd.md）。

## 2. 术语表

| 术语 | 含义 |
|------|------|
| 代理端点 | richman 透传请求到 richson 并返回响应，不做业务处理 |
| 聚合端点 | richman 读取多个数据源（rm_*/rs_* 表或 richson API）后组装响应 |
| 平台配额 | 系统内置 LLM API Key，用于标的级预计算等非用户触发的 LLM 调用 |
| percentileLabel | richman 根据历史评分计算的分位标签（"近一年偏高"等） |
| 事件告警（Event Alert） | richson 检测到 Polymarket 概率变动后写入 rs_event_alerts 的数据层记录 |
| 市场快讯（Market Alert） | 推送给用户的即时更新邮件，触发源包括事件告警（Polymarket 概率变动 >20pp）和评分变化（>=10 分） |

## 3. richson HTTP 客户端

### 3.1 客户端结构

新增 `internal/richson/client.go`，封装对 richson 的全部 HTTP 调用。

```go
// Package richson provides an HTTP client for the richson quantitative service.
package richson

// Client wraps HTTP communication with the richson sidecar service.
type Client struct {
    baseURL        string
    apiKey         string
    httpClient     *http.Client // shared, with default timeout
    asyncTimeout   time.Duration
    syncTimeout    time.Duration
    lightTimeout   time.Duration
    logger         *zap.Logger
}

// NewClient creates a richson client from config.
func NewClient(cfg config.RichsonConfig, logger *zap.Logger) *Client
```

### 3.2 超时与重试策略

对齐 richson-service-trd.md SS16.1：

| 调用类型 | HTTP 超时 | 重试次数 | 重试间隔 | 降级行为 |
|----------|-----------|----------|----------|----------|
| 异步触发 POST /jobs/* | 5s | 1 | 2s | 返回 502 |
| 同步分析 POST /analyze/* | 30s | 1 | 2s | 返回降级响应 |
| 轻量查询 GET /market/*, /events/* | 10s | 1 | 2s | 返回缓存或 502 |
| 健康检查 GET /health | 3s | 0 | - | 标记 unhealthy |

重试条件：仅在网络错误或 HTTP 502/503 时重试，其他状态码不重试。重试复用原 context（继承取消信号），但每次请求独立设置 HTTP 超时（通过 http.Client.Timeout 或 per-request context.WithTimeout）。

### 3.3 请求链路追踪

每个调用自动注入 `X-Request-ID` header（richson-service-trd.md SS4.3）。如果调用方 context 中已有 request_id（来自前端请求），复用之；否则生成新 UUID。

```go
func (c *Client) setHeaders(req *http.Request, requestID string) {
    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Request-ID", requestID)
}
```

### 3.4 错误处理

richson 返回的错误格式与 richman 统一（richson-service-trd.md SS4.4）。客户端解析错误响应并映射为 richman 内部错误类型：

```go
// richson 特有错误码映射
var richsonErrorMap = map[string]int{
    "ANALYSIS_IN_PROGRESS":     http.StatusConflict,
    "DATA_SOURCE_UNAVAILABLE":  http.StatusBadGateway,
    "LLM_TIMEOUT":              http.StatusGatewayTimeout,
    "LLM_INVALID_RESPONSE":     http.StatusBadGateway,
    "ASSET_NOT_SUPPORTED":      http.StatusBadRequest,
    "INSUFFICIENT_HISTORY":     http.StatusBadRequest,
}
```

### 3.5 客户端方法清单

| 方法 | richson 端点 | 交互模式 | 返回类型 |
|------|-------------|----------|----------|
| TriggerAssetAnalysis | POST /jobs/analyze-asset | A | JobResponse |
| TriggerBatchAnalysis | POST /jobs/batch-analyze | A | BatchJobResponse |
| GetJobStatus | GET /jobs/{jobId} | C | JobDetailResponse |
| AnalyzeHolding | POST /analyze/holding | B | HoldingAnalysisResponse |
| GetDemoPlan | POST /analyze/demo-plan | C | DemoPlanResponse |
| GetMarketRegime | GET /market/regime | C | MarketRegimeResponse |
| GetOHLCV | GET /market/ohlcv/{code} | C | OHLCVResponse |
| GetScoreHistory | GET /assets/{code}/score-history | C | ScoreHistoryResponse |
| GetEventsRadar | GET /events/radar | C | EventsRadarResponse |
| GenerateWeeklyInsight | POST /content/weekly-insight | B | WeeklyInsightResponse |
| HealthCheck | GET /health | C | HealthResponse |

### 3.6 健康检查集成

richman 启动时异步检查 richson 可用性（非阻塞，仅日志），不阻塞 richman 自身启动。运行时通过 cron 每 30 秒检查 richson 健康状态，状态用 atomic.Bool 存储，v2 端点在 richson 不可用时返回 503。

```go
func (c *Client) IsHealthy() bool
```

## 4. v2 API Handler 层

### 4.1 路由分组

新增 `internal/api/v2/` 目录，与 v1 handler 并行。v2 路由组挂在 `/api/v2` 前缀下：

```
/api/v2/
  market/                # 公开（无 JWT）
    regime               GET  -> MarketHandler
    overview             GET  -> MarketHandler
    :code                GET  -> MarketHandler
    :code/ohlcv          GET  -> MarketHandler
    :code/scores         GET  -> MarketHandler
    :code/demo-plan      GET  -> MarketHandler
    :code/share          GET  -> MarketHandler  # JWT 可选，已登录时附带邀请码（invite-system-trd SS6）
  events/
    radar                GET  -> EventHandler
  analysis/              # 需 JWT
    trigger-asset        POST -> AnalysisHandler
    jobs/:jobId          GET  -> AnalysisHandler
    holding/:holdingId   POST -> AnalysisHandler
  briefing               GET  -> BriefingHandler  # 需 JWT
  feedback               POST -> FeedbackHandler  # 需 JWT
  user/
    risk-preference      PATCH -> UserHandler      # 需 JWT
    email-push           PATCH -> UserHandler      # 需 JWT，body: { "enabled": bool }
  invite/                # 需 JWT，完整设计见 invite-system-trd.md SS5
    my-codes             GET  -> InviteHandler
    my-invites           GET  -> InviteHandler
```

公开端点（market/*、events/*）不经过 JWT 中间件，但经过 IP 限流中间件。认证端点（analysis/*、briefing、feedback、user/*）经过 JWT 中间件。

v1 认证端点（`/api/v1/auth/register`、`/api/v1/auth/login`）也需 IP 限流：每 IP 每分钟 5 次，防止暴力猜测邀请码和密码。在 v1 路由组中增加限流中间件。

注意：`/api/v2/market/:code` 与 `/api/v2/market/regime`、`/api/v2/market/overview` 存在 Gin 路由参数冲突风险。`regime` 和 `overview` 为精确路径，必须在 `:code` 之前注册，Gin 按注册顺序匹配时精确路径优先于参数路径。

### 4.2 Handler 分类

v2 handler 分为两类：

**代理 handler**（透传到 richson）：

| richman 端点 | richson 端点 | 说明 |
|-------------|-------------|------|
| GET /api/v2/market/regime | GET /market/regime | 直接透传 |
| GET /api/v2/market/:code/ohlcv | GET /market/ohlcv/:code | 透传 + 转发 query 参数 |
| GET /api/v2/market/:code/scores | GET /assets/:code/score-history | 透传 + 转发 query 参数 |
| GET /api/v2/events/radar | GET /events/radar | 直接透传 |
| POST /api/v2/analysis/trigger-asset | POST /jobs/analyze-asset | 注入平台 llmConfig 后转发 |
| GET /api/v2/market/:code/demo-plan | -- | 从 rs_asset_analyses.demo_plan 预计算字段读取，不调 richson |

**demo-plan 数据来源**：PRD SS5.2.4 明确"每日随标的级分析一起预计算"，因此 demo plan 在 06:00 批量分析时由 richson 预计算并存入 `rs_asset_analyses.demo_plan`（JSONB）。richman GET 端点直接从 DB 读取，不需要实时调用 richson。独立的 `POST /analyze/demo-plan` richson 端点保留作为 fallback（当 DB 中 demo_plan 为 null 时按需生成）。

代理 handler 不解析 richson 响应体（除错误检测外），直接将 richson 响应 body 写入 gin.Context。

**聚合 handler**（richman 组装数据）：

| richman 端点 | 数据来源 | 组装逻辑 |
|-------------|----------|----------|
| GET /api/v2/market/overview | rm_asset_catalog + rs_asset_analyses | 按类别分组 + 最新分析拼接 |
| GET /api/v2/market/:code | rm_asset_catalog + rs_asset_analyses + rs_asset_analysis_dimensions | 标的元信息 + 完整分析数据 |
| POST /api/v2/analysis/holding/:holdingId | rm_holdings + richson POST /analyze/holding | 取持仓 -> 调 richson -> 持久化决策卡片 |
| GET /api/v2/analysis/jobs/:jobId | rs_analysis_jobs | 直读 DB |
| GET /api/v2/briefing | rm_holdings + rs_asset_analyses + rm_decision_cards | 聚合持仓 + 最新分析 + 最新决策卡片 |
| POST /api/v2/feedback | rm_user_feedback | 直写 DB |
| PATCH /api/v2/user/risk-preference | rm_users | 直更 DB |

### 4.3 公开端点 IP 限流

richman 对 `/api/v2/market/*` 和 `/api/v2/events/*` 实施 IP 级限流（richson-service-trd.md SS20）：

- 限流粒度：单 IP 每分钟 60 次
- 实现方式：Gin 中间件，进程内 map + 滑动窗口计数
- 超限返回 429 Too Many Requests
- 不引入 Redis——MVP 单实例部署，进程内计数足够
- IP 获取策略：优先读取 `X-Forwarded-For` / `X-Real-IP` header（Gin 的 `c.ClientIP()` 已内置此逻辑，需在 Gin Engine 上设置 `SetTrustedProxies`），确保反向代理后获取真实客户端 IP

### 4.4 Handler 注册模式

沿用 v1 的 `RegisterRoutes` 模式：

```go
type MarketHandler struct {
    richsonClient *richson.Client
    marketSvc     *market.Service
    logger        *zap.Logger
}

func (h *MarketHandler) RegisterRoutes(rg *gin.RouterGroup) {
    rg.GET("/regime", h.getMarketRegime)
    rg.GET("/overview", h.getMarketOverview)
    rg.GET("/:code", h.getAssetDetail)
    rg.GET("/:code/ohlcv", h.getOHLCV)
    rg.GET("/:code/scores", h.getScoreHistory)
    rg.GET("/:code/demo-plan", h.getDemoPlan)
    rg.GET("/:code/share", h.getShareData) // invite-system-trd SS6
}
```

## 5. v2 Service 层

### 5.1 新增 Service

| Service | 包路径 | 职责 |
|---------|--------|------|
| MarketService | internal/service/market/ | Market Overview 聚合、标的详情聚合、percentileLabel 计算 |
| BriefingService | internal/service/briefing/ | 投研简报数据聚合（持仓 + 分析 + 决策卡片 + sparkline） |
| FeedbackService | internal/service/feedback/ | 用户反馈 CRUD |
| EmailPushService | internal/service/emailpush/ | 邮件推送内容组装与调度 |

### 5.2 MarketService

```go
type MarketService struct {
    assetRepo     *repo.AssetRepo
    analysisRepo  *repo.AssetAnalysisReadRepo   // 只读 rs_asset_analyses
    dimensionRepo *repo.AnalysisDimensionReadRepo // 只读 rs_asset_analysis_dimensions
    logger        *zap.Logger
}

// GetOverview returns grouped asset cards with latest analysis for each active asset.
func (s *MarketService) GetOverview(ctx context.Context) (*MarketOverviewDTO, error)

// GetAssetDetail returns full analysis data for a single asset code.
func (s *MarketService) GetAssetDetail(ctx context.Context, code string) (*AssetDetailDTO, error)
```

**percentileLabel 计算逻辑**（PRD SS3.4）：

richman 查询目标标的最近 365 天 rs_asset_analyses 记录的 overall_score 分布，计算当前评分在该分布中的百分位：

| 百分位区间 | percentileLabel |
|-----------|-----------------|
| P90+ | 近一年偏高 |
| P75-89 | 近一年中高 |
| P25-74 | 近一年中位 |
| P10-24 | 近一年中低 |
| P10以下 | 近一年偏低 |

此计算在 MarketService 中完成（进程内 TTL 缓存 1 小时，key = asset_code），不依赖 richson。

冷启动处理：历史分析记录不足 30 天时不显示 percentileLabel（返回 null），前端隐藏分位标签。30-364 天时使用可用数据范围的百分位（标注"近 N 月"而非"近一年"）。

### 5.3 BriefingService

```go
type BriefingService struct {
    holdingRepo   *repo.HoldingRepo
    analysisRepo  *repo.AssetAnalysisReadRepo
    cardRepo      *repo.DecisionCardRepo
    logger        *zap.Logger
}

// GetBriefing returns briefing cards for all holdings of a user.
func (s *BriefingService) GetBriefing(ctx context.Context, userID int64) (*BriefingDTO, error)
```

聚合步骤：
1. 查询用户所有活跃持仓（rm_holdings WHERE user_id = ? AND is_deleted = 0）
2. 批量查询持仓标的的最新分析（rs_asset_analyses，按 asset_code IN (...) 最新一条）
3. 批量查询最新决策卡片（rm_decision_cards WHERE holding_id IN (...)）
4. 查询最近 90 天评分用于 sparkline（rs_asset_analyses WHERE asset_code IN (...) 最近 90 条 overall_score）
5. 计算浮盈亏（基于分析记录中的 price_at_analysis 和持仓成本）
6. 计算集中度（同二级分类持仓合计仓位，对照三级阈值）
7. 组装 BriefingCardDTO 列表

步骤 1-4 的 DB 查询可并行执行（errgroup）。

### 5.4 FeedbackService

```go
type FeedbackService struct {
    feedbackRepo *repo.UserFeedbackRepo
    logger       *zap.Logger
}

// Create saves user feedback for an analysis.
func (s *FeedbackService) Create(ctx context.Context, userID int64, input *CreateFeedbackInput) (int64, error)
```

输入校验：
- rating 只接受 "helpful" 或 "not_helpful"
- comment 最大 500 字符
- asset_analysis_id 必须存在于 rs_asset_analyses

### 5.5 持仓级分析流程（AnalysisService 扩展）

持仓级分析的 handler 需要协调多步操作：

1. 根据 holdingId 查询持仓详情（rm_holdings）
2. 查询持仓标的最新分析 ID（rs_asset_analyses WHERE asset_code = ? ORDER BY analyzed_at DESC LIMIT 1）
3. 查询同类标的总敞口（rm_holdings WHERE user_id = ? AND asset_type = ?，汇总 position_ratio）
4. 查询用户风险偏好（rm_users.risk_preference）
5. 查询用户 LLM 配置（rm_llm_configs）并解密 API Key
6. 调用 richson POST /analyze/holding（传入持仓信息、分析 ID、风险偏好、同类敞口、LLM 配置）
7. 将 richson 返回的执行计划持久化到 rm_decision_cards

步骤 1-5 可并行查询。步骤 6 依赖前序查询结果。步骤 7 依赖步骤 6。

**幂等防护**：handler 入口处对 holdingId 加 per-user 内存锁（`sync.Map` key = `userID:holdingID`），TryLock 失败返回 409 "分析进行中"。同一持仓同一时刻只能有一个分析请求在执行，防止双击和 cron/手动并发重复生成决策卡片。

**LLM API Key 传输安全**：richman 调用 richson 时通过 HTTP body 传递解密后的用户 API Key。richson 启动时校验：若 `RICHMAN_BASE_URL` 非 localhost/127.0.0.1 则强制要求 HTTPS，否则拒绝启动。richson 不持久化 key，请求结束后内存释放。日志中对 apiKey 字段做 masking（仅记录后 4 位）。

新增 `internal/service/analysis/v2_holding.go` 承载此逻辑，不修改现有 v1 分析 service。

### 5.6 现有 Service 变更

| Service | 变更 | 说明 |
|---------|------|------|
| UserService | 新增 UpdateRiskPreference 方法 | 更新 rm_users.risk_preference 列 |
| PortfolioService | 无接口变更 | 表名从 holdings 变为 rm_holdings，由 repo 层处理 |
| NotificationService | 新增 SendBroadcast 方法 | 批量发送邮件给所有注册用户（不基于 channel 配置） |

## 6. v2 Repo 层

### 6.1 新增 Repo（只读访问 rs_* 表）

richman 对 rs_* 表只有 SELECT 权限（richson-service-trd.md SS6.4）。

```go
// AssetAnalysisReadRepo provides read-only access to rs_asset_analyses.
type AssetAnalysisReadRepo struct {
    pool *pgxpool.Pool
}

// GetLatestByAssetCode returns the most recent analysis for an asset.
func (r *AssetAnalysisReadRepo) GetLatestByAssetCode(ctx context.Context, code string) (*model.AssetAnalysis, error)

// GetLatestByAssetCodes returns the most recent analysis for multiple assets (batch).
func (r *AssetAnalysisReadRepo) GetLatestByAssetCodes(ctx context.Context, codes []string) (map[string]*model.AssetAnalysis, error)

// GetScoresForPercentile returns overall_score values for the past N days for percentile calculation.
func (r *AssetAnalysisReadRepo) GetScoresForPercentile(ctx context.Context, code string, days int) ([]float64, error)

// GetSparklineScores returns recent N overall_score values for sparkline rendering.
func (r *AssetAnalysisReadRepo) GetSparklineScores(ctx context.Context, code string, limit int) ([]float64, error)
```

```go
// AnalysisDimensionReadRepo provides read-only access to rs_asset_analysis_dimensions.
type AnalysisDimensionReadRepo struct {
    pool *pgxpool.Pool
}

// GetByAnalysisID returns all dimension records for a given analysis.
func (r *AnalysisDimensionReadRepo) GetByAnalysisID(ctx context.Context, analysisID int64) ([]model.AnalysisDimension, error)
```

```go
// AnalysisJobReadRepo provides read-only access to rs_analysis_jobs.
type AnalysisJobReadRepo struct {
    pool *pgxpool.Pool
}

// GetByJobID returns a job by its UUID.
func (r *AnalysisJobReadRepo) GetByJobID(ctx context.Context, jobID string) (*model.AnalysisJob, error)
```

```go
// EventAlertReadRepo provides read-only access to rs_event_alerts.
type EventAlertReadRepo struct {
    pool *pgxpool.Pool
}

// GetUnalerted returns unprocessed event alerts.
func (r *EventAlertReadRepo) GetUnalerted(ctx context.Context) ([]model.EventAlert, error)

// MarkAlerted marks an event alert as alerted (richman writes to rs_event_alerts).
// Note: this is a cross-service write exception -- richman updates only the
// alerted flag to avoid richson needing awareness of notification delivery.
// Production DB user richman_user needs UPDATE permission on rs_event_alerts.alerted column only.
func (r *EventAlertReadRepo) MarkAlerted(ctx context.Context, ids []int64) error
```

rs_event_alerts.alerted 列的跨服务写入是唯一的例外，richman 需要标记已处理的告警以避免重复通知。生产环境可通过列级 GRANT 限制 richman_user 仅更新 alerted 列。

### 6.2 新增 Repo（rm_* 新表）

```go
// UserFeedbackRepo handles CRUD for rm_user_feedback.
type UserFeedbackRepo struct {
    pool *pgxpool.Pool
}

// Create inserts a new feedback record.
func (r *UserFeedbackRepo) Create(ctx context.Context, userID int64, analysisID int64, rating, comment string) (int64, error)
```

### 6.3 现有 Repo 变更

所有现有 repo 的 SQL 查询中表名从无前缀更新为 rm_ 前缀。由 sqlc 重新生成。

| 现有 Repo | 表名变更 | 新增方法 |
|-----------|----------|----------|
| UserRepo | users -> rm_users | UpdateRiskPreference(ctx, userID, preference) |
| HoldingRepo | holdings -> rm_holdings | GetExposureByAssetType(ctx, userID, assetType) float64 |
| AssetRepo | asset_catalog -> rm_asset_catalog | ListActiveWithType(ctx) 返回含 asset_type 的列表 |
| DecisionCardRepo | decision_cards -> rm_decision_cards | GetLatestByHoldings(ctx, holdingIDs) map |
| 其他 repo | 对应 rm_ 前缀 | 无新增方法 |

### 6.4 sqlc 查询文件变更

所有 `backend/db/query/*.sql` 文件中的表引用需更新为 rm_ 前缀。新增查询文件：

注意：当前项目 repo 层使用手写 SQL + pgxpool 直接查询（无 sqlc）。v2 新增的 repo 沿用此模式。新增查询逻辑直接在 repo .go 文件中编写，不依赖 sqlc 代码生成。

v2 新增 repo 涉及的查询：
- rs_asset_analyses 只读查询（AssetAnalysisReadRepo）
- rs_asset_analysis_dimensions 只读查询（AnalysisDimensionReadRepo）
- rs_analysis_jobs 只读查询（AnalysisJobReadRepo）
- rs_event_alerts 读取 + alerted 更新（EventAlertReadRepo）
- rm_user_feedback CRUD（UserFeedbackRepo）

## 7. 通知与邮件推送系统

### 7.1 v2 邮件推送架构

v1 通知系统基于用户配置的 notification_channels 做 per-user 推送。v2 新增**平台级推送**——不依赖用户的 channel 配置，直接用用户注册邮箱发送。两种模式并存：

| 模式 | 触发方 | 收件人 | 使用场景 |
|------|--------|--------|----------|
| v1 Channel 推送 | 用户配置触发 | 用户配置的渠道（邮件/飞书/微信） | 自定义通知 |
| v2 平台推送 | cron 定时 | 所有注册用户的注册邮箱 | 每日简报、每周洞察、市场快讯 |

v2 平台推送新增 `internal/service/emailpush/` 模块，不复用 v1 的 Dispatcher + Channel 流程。

### 7.2 EmailPushService

```go
type EmailPushService struct {
    userRepo       *repo.UserRepo
    analysisRepo   *repo.AssetAnalysisReadRepo
    holdingRepo    *repo.HoldingRepo
    cardRepo       *repo.DecisionCardRepo
    eventAlertRepo *repo.EventAlertReadRepo
    richsonClient  *richson.Client
    emailSender    *email.Sender
    templateEngine *template.Engine
    logger         *zap.Logger
}
```

核心方法：

```go
// SendDailyBriefing sends morning briefing email to all registered users.
// Content: market regime + gold score (vs yesterday) + today's events + holding suggestions.
// PRD SS10.3: even when nothing changed, still send "no action needed" email.
func (s *EmailPushService) SendDailyBriefing(ctx context.Context) error

// SendWeeklyInsight sends Monday weekly insight to all registered users.
// Calls richson POST /content/weekly-insight for LLM-generated content.
func (s *EmailPushService) SendWeeklyInsight(ctx context.Context) error

// SendMarketAlert sends event-driven alert to all registered users.
// Triggered when Polymarket event probability changes > 20pp (PRD SS3.6).
func (s *EmailPushService) SendMarketAlert(ctx context.Context, alert *model.EventAlert) error

// SendHoldingSuggestion sends personalized holding suggestion to a specific user.
// Triggered after daily analysis when execution plan changes.
func (s *EmailPushService) SendHoldingSuggestion(ctx context.Context, userID int64, card *model.DecisionCard) error
```

### 7.3 邮件发送器

扩展现有 `internal/notification/adapter/email/email.go`，新增面向 v2 的 Sender：

```go
// Sender wraps SMTP sending for v2 platform-level emails.
type Sender struct {
    host     string
    port     int
    user     string
    password string
    from     string // display name + address, e.g. "Richman <noreply@richman.app>"
    logger   *zap.Logger
}

// Send delivers an HTML email to a single recipient.
func (s *Sender) Send(ctx context.Context, to, subject, htmlBody string) error

// SendBatch delivers the same email to multiple recipients.
// Uses BCC batching (50 per batch) to avoid exposing recipient list.
func (s *Sender) SendBatch(ctx context.Context, recipients []string, subject, htmlBody string) error
```

SendBatch 按 50 人一组分批 BCC 发送，避免单封邮件收件人过多被 SMTP 服务商拦截。每批之间间隔 1 秒，避免触发速率限制。

### 7.4 HTML 邮件模板引擎

新增 `internal/service/emailpush/template/` 目录，使用 Go `html/template` 渲染邮件。

模板文件：

| 模板 | 文件 | 用途 |
|------|------|------|
| 每日简报 | daily_briefing.html | 市场体制 + 评分 + 事件 + 持仓建议 |
| 每周洞察 | weekly_insight.html | 周回顾 + 周展望 + 教育内容 |
| 市场快讯 | market_alert.html | 事件名 + 概率变动 + 评分影响 |
| 持仓建议 | holding_suggestion.html | 标的 + 评分 + 执行计划摘要 |

模板设计原则：
- 内联 CSS（邮件客户端不支持外部样式表）
- 表格布局（兼容旧版 Outlook 等）
- 响应式：单列布局，最大宽度 600px
- 暗色模式：通过 `@media (prefers-color-scheme: dark)` 提供基础暗色支持
- 每封邮件底部包含"取消订阅"链接（链接到 richman 设置页 /settings）
- 每封邮件底部包含免责声明文字（PRD SS13.1）
- i18n 支持：每个模板按 locale 分为 `{name}_zh.html` 和 `{name}_en.html`，EmailPushService 根据用户语言偏好选择模板。邮件 Subject 同样跟随 locale（如 "Richman 每日简报" / "Richman Daily Briefing"）

### 7.4.1 邮件退订机制

用户在设置页可关闭平台邮件推送。通过 rm_users 表新增 `email_push_enabled BOOLEAN NOT NULL DEFAULT TRUE` 列（纳入 migration 022）控制：

- email_push_enabled = true：接收所有平台推送邮件
- email_push_enabled = false：不接收任何平台推送邮件（v1 channel 推送不受影响）

EmailPushService 在查询用户列表时过滤 `WHERE email_push_enabled = TRUE`。

退订链接跳转到 `/settings` 页面，前端在设置页提供开关。不实现一键邮件退订链接（MVP 简化），后续可增加带 token 的退订 URL。

### 7.5 每日简报内容组装

每日简报（PRD SS10.3）内容全部来自预计算数据，不需要额外 LLM 调用：

```
数据获取（并行）:
1. 查询市场体制 -> richson GET /market/regime（或读缓存）
2. 查询黄金最新分析 -> rs_asset_analyses WHERE asset_code = 'GLD' ORDER BY analyzed_at DESC LIMIT 1
3. 查询昨日分析 -> rs_asset_analyses WHERE asset_code = 'GLD' 倒数第二条（用于计算变化）
4. 查询今日关注事件 -> richson GET /events/radar
5. 查询所有注册用户 -> rm_users WHERE is_deleted = 0

对于有持仓的用户额外查询:
6. 查询用户持仓 -> rm_holdings WHERE user_id = ? AND is_deleted = 0
7. 查询持仓标的最新决策卡片 -> rm_decision_cards

内容组装:
- 公共部分（所有用户相同）: 体制 + 评分 + 评分变化 + 事件列表
- 个性化部分（有持仓用户）: 持仓建议摘要
- 无持仓用户: 公共部分 + "录入持仓获取专属建议" CTA

按用户分组渲染模板，批量发送。
```

**大用户量处理**：步骤 5 查询注册用户时使用游标分页（每页 200 用户），避免一次性加载全部用户到内存。对每页用户批量查询持仓和决策卡片，然后发送该页的邮件后再加载下一页。

### 7.6 每周投研洞察内容组装

每周一 08:30（PRD SS10.3）需调用 richson 生成 LLM 内容：

1. 调用 richson POST /content/weekly-insight（传入平台 LLM 配置 + locale）
2. richson 返回 weeklyReview + weeklyOutlook + educationTopic
3. richman 渲染到 weekly_insight.html 模板
4. 批量发送给所有注册用户

如果 richson 调用失败，跳过本周周报发送，记录 ERROR 日志。不降级为模板生成——周报的价值在于 LLM 生成的深度内容，模板生成无意义。

### 7.7 推送频率控制

PRD SS10.1 规定用户每日最多 3 次推送。richman 在 EmailPushService 中维护每日发送计数：

- 通过 rm_notification_logs 表按 user_id + date 统计当日已发送次数
- 每次发送前检查，超限则跳过并记录 WARN 日志
- 每日简报和每周洞察各算一次，市场快讯和持仓建议各算一次
- 推送优先级：每日简报 > 持仓建议 > 市场快讯 > 每周洞察

## 8. 定时任务（Cron）重构

### 8.1 v2 Cron 任务总览

对齐 richson-service-trd.md SS9.3：

| 任务 | 触发时间（UTC+8） | 动作 | 超时 |
|------|-------------------|------|------|
| 每日标的分析 | 06:00 | 调用 richson POST /jobs/batch-analyze | 5 min |
| 每日持仓分析 | 07:30 | 对所有活跃持仓触发 holding 分析，检测执行计划变化 | 15 min |
| 每日简报邮件 | 08:30 | EmailPushService.SendDailyBriefing | 10 min |
| A股收盘后快讯 | 15:30 | 检查 A 股相关标的评分变化，推送收盘快讯 | 5 min |
| 美股收盘后快讯 | 06:00 | 随每日标的分析完成后检查美股标的评分变化，推送快讯 | 合并在标的分析中 |
| 每周投研洞察 | 周一 08:30 | EmailPushService.SendWeeklyInsight | 5 min |
| 事件告警轮询 | 每小时整点 | 读 rs_event_alerts 未处理记录 -> 发送通知 -> 标记 alerted | 2 min |
| 过期 Job 清理 | 每 10 分钟 | UPDATE rs_analysis_jobs SET status = 'failed' WHERE expired | 30s |
| richson 健康检查 | 每 30 秒 | GET /health | 3s |

**三窗口推送策略**（PRD SS10.1）：

| 窗口 | 时间 | 覆盖标的 | 推送内容 |
|------|------|---------|---------|
| 早盘前 | 08:30 | 全部 | 每日简报（市场体制 + 黄金评分变化 + 今日事件 + 持仓建议摘要） |
| A股收盘后 | 15:30 | A 股相关 | A 股标的收盘快讯（评分变化 >= 5 分的标的摘要 + 持仓建议变化） |
| 美股收盘后 | 06:00 | 美股 + 黄金 | 随每日分析完成后检查评分变化并推送（评分变化 >= 5 分的标的摘要） |

每日每用户最多 3 次推送。推送频率控制见 SS7.7。

### 8.2 Cron 实现

沿用现有的 cron 调度器（`github.com/robfig/cron/v3`），在 `internal/service/schedule/` 下新增 v2 任务注册：

```go
// RegisterV2CronJobs registers all v2 cron tasks.
func RegisterV2CronJobs(
    c *cron.Cron,
    richsonClient *richson.Client,
    emailPushSvc *emailpush.Service,
    eventAlertRepo *repo.EventAlertReadRepo,
    analysisJobRepo *repo.AnalysisJobReadRepo,
    logger *zap.Logger,
) {
    // Daily asset analysis trigger (06:00 UTC+8 = 22:00 UTC)
    c.AddFunc("0 22 * * *", func() { ... })

    // Daily holding analysis (07:30 UTC+8 = 23:30 UTC previous day)
    c.AddFunc("30 23 * * *", func() { ... })

    // Daily briefing email (08:30 UTC+8 = 00:30 UTC)
    c.AddFunc("30 0 * * *", func() { ... })

    // A-share closing alert (15:30 UTC+8 = 07:30 UTC)
    c.AddFunc("30 7 * * 1-5", func() { ... })

    // Weekly insight (Monday 08:30 UTC+8 = Monday 00:30 UTC)
    c.AddFunc("30 0 * * 1", func() { ... })

    // Event alert polling (hourly)
    c.AddFunc("0 * * * *", func() { ... })

    // Expired job cleanup (every 10 minutes)
    c.AddFunc("*/10 * * * *", func() { ... })
}
```

### 8.3 每日标的分析触发

```
流程:
1. 从 rm_asset_catalog 查询所有 is_active = true 的标的
2. 构造 batch-analyze 请求（包含所有激活标的 + 平台 LLM 配置）
3. 调用 richson POST /jobs/batch-analyze
4. 记录响应中的 jobs 和 skipped
5. 成功则记录 INFO 日志，失败记录 ERROR 日志
```

不做轮询等待 job 完成——job 完成后 richson 自动写入 rs_asset_analyses，下游 cron（08:30 简报）直接读取最新数据。

**数据新鲜度保障**：08:30 简报 cron 在组装内容前检查最新分析的 analyzed_at 是否在当日 06:00 之后。如果不是（说明 06:00 的批量分析尚未完成或失败），记录 WARN 日志并使用最近一次可用的分析数据（不阻塞简报发送）。

### 8.3.1 每日持仓分析触发

PRD SS10.3 "持仓建议"通知要求"日常更新后执行计划变化时推送"。这需要每日自动触发所有活跃持仓的 holding 分析：

```
流程:
1. 查询所有有持仓的用户 -> rm_holdings 的 DISTINCT user_id WHERE is_deleted = 0
2. 对每个用户:
   a. 查询所有活跃持仓
   b. 对每个持仓调用 richson POST /analyze/holding（同步，30s 超时）
   c. 将 richson 返回结果持久化到 rm_decision_cards
   d. 比较新旧决策卡片的 action 字段，如果变化则标记需发送通知
3. 对所有执行计划有变化的持仓，调用 EmailPushService.SendHoldingSuggestion
```

并发控制：对不同用户的持仓可并行分析（goroutine pool，限制并发数 5），但同一用户的持仓串行分析（避免 peerExposure 计算竞态）。

LLM 配额：持仓级分析使用用户自己的 LLM key（优先）或平台配额（兜底）。如果用户无 key 且平台配额用尽，跳过该用户并记录 WARN。

容错：单个持仓分析失败不影响其他持仓。全部完成后记录成功/失败计数。

### 8.3.2 A 股收盘后快讯（15:30）

PRD SS10.1 要求 A 股收盘后推送。仅在交易日（周一至周五）触发：

```
流程:
1. 从 rm_asset_catalog 查询 A 股相关标的（asset_code 为 6 位数字）
2. 读取 rs_asset_analyses 最新分析记录，比较与前一日 overall_score 差异
3. 筛选 |score_delta| >= 5 的标的
4. 如有变化:
   a. 构造 A 股收盘快讯内容（变化标的评分摘要 + 持仓建议变化）
   b. 调用 EmailPushService.SendMarketAlert
5. 无变化则跳过（不发送"无变化"邮件）
```

注意：15:30 快讯不触发新的分析计算。A 股标的的分析数据来自 06:00 全量分析的结果（richson 在全量分析中已包含 A 股 ETF）。此 cron 仅做"评分变化检测 + 推送"。

**数据时效性说明**：06:00 分析使用的 A 股数据为前一日收盘价，15:30 快讯的评分变化不包含当日 A 股交易数据。MVP 阶段仅黄金标的激活（A 股标的置灰），因此此 cron 实际不产生推送。Phase 2 A 股标的激活后，考虑在 15:15 增加 A 股标的的 D4 技术面增量刷新分析（仅价格相关维度），使 15:30 快讯包含当日收盘数据。

**去重逻辑**：15:30 快讯跳过已在 06:00 市场快讯（SS8.3.3）中推送过的标的（通过 rm_notification_logs 查询当日该标的是否已有 market_alert 类型发送记录）。

### 8.3.3 评分变化触发市场快讯

PRD SS10.3 要求"评分变化 >= 10 分"也触发市场快讯（独立于 Polymarket 事件告警）。此检测在每日标的分析完成后执行：

```
流程（嵌入 8.3 每日标的分析触发的步骤 5）:
5. 分析完成后，读取所有激活标的的最新与前一日分析记录
6. 筛选 |overall_score delta| >= 10 的标的
7. 对每个满足条件的标的:
   a. 构造市场快讯内容："黄金评分从 X 升至 Y，主要驱动：[change_summary]"
   b. 调用 EmailPushService.SendMarketAlert（发送给所有注册用户）
8. 受每日 3 次推送频率上限约束
```

### 8.4 事件告警轮询

```
流程:
1. 读取 rs_event_alerts WHERE alerted = FALSE AND is_deleted = 0
2. 对每条告警:
   a. 构造市场快讯邮件内容
   b. 调用 EmailPushService.SendMarketAlert
   c. 成功后标记 alerted = TRUE
3. 批量更新 alerted 状态
```

### 8.5 过期 Job 清理

richman 负责清理超时未完成的 job（richson-service-trd.md SS9.3）：

```sql
UPDATE rs_analysis_jobs
SET status = 'failed',
    error_message = 'job expired',
    error_code = 'JOB_EXPIRED',
    updated_at = NOW(),
    modifier = 'richman_cron'
WHERE status IN ('pending', 'running')
  AND expires_at < NOW()
  AND is_deleted = 0;
```

注意：这是 richman 对 rs_* 表的第二个写操作例外（第一个是 event_alerts.alerted）。生产环境需为 richman_user 授予 rs_analysis_jobs 的 UPDATE 权限（仅 status, error_message, error_code, updated_at, modifier 列）。

**跨服务写入例外汇总**（需同步更新 richson-service-trd.md SS6.4）：

| 表 | richman 写入的列 | 原因 |
|----|-----------------|------|
| rs_event_alerts | alerted | 避免 richson 感知通知投递逻辑 |
| rs_analysis_jobs | status, error_message, error_code, updated_at, modifier | 过期 job 清理由 richman cron 统一执行 |

### 8.6 v1 Cron 任务处理

v1 的定时分析触发任务在 Phase 1（richson 上线）后废弃。具体处理：

| v1 Cron 任务 | v2 状态 |
|-------------|---------|
| 早盘分析触发 (AM brief) | Phase 1 废弃，由 v2 每日标的分析替代 |
| 午盘分析触发 (PM digest) | Phase 1 废弃 |
| 美股收盘分析 (US digest) | Phase 1 废弃，由 v2 每日标的分析统一 |

v2 将分析计算统一为每日一次全量分析（美股收盘后 06:00 UTC+8），richson 的分析结果覆盖所有市场。但推送仍保留三窗口差异化（08:30 简报 / 15:30 A 股快讯 / 06:00 美股快讯），满足 PRD SS10.1 要求。

### 8.7 Cron 任务互斥

耗时较长的 cron 任务（每日标的分析、每日持仓分析、每日简报邮件）使用 sync.Mutex 防止下一轮触发时上一轮尚未完成导致并发执行。每个任务独立一把锁，不同任务之间不互斥。TryLock 失败时跳过本轮并记录 WARN 日志。

### 8.8 批量分析失败恢复

如果 06:00 的批量分析因 richson 不可用而完全失败，可通过管理员手动重试：

- 方式一：调用 `POST /api/v2/analysis/trigger-batch`（管理员端点，需 admin JWT）
- 方式二：richman CLI 命令 `make trigger-batch-analysis`

MVP 不实现自动重试——避免在 richson 恢复瞬间产生突发负载。

## 9. 数据库迁移

### 9.1 迁移 021：存量表 rm_ 前缀

完整 SQL 见 richson-service-trd.md SS6.1。此处补充 richman 侧的实施注意事项：

执行顺序：
1. 停止 richman 服务
2. 执行 migration 021（ALTER TABLE ... RENAME）
3. 部署新版 richman（所有 SQL 已更新为 rm_ 表名）
4. 启动 richman

**事务包裹**：14 条 ALTER TABLE RENAME 必须在单个事务中执行。migration runner（runner.go）已自动用 `pool.Begin()`/`tx.Commit()` 包裹每个迁移文件，因此脚本内**不得**再写 `BEGIN;`/`COMMIT;`（否则嵌套事务语义异常）。如果任一 RENAME 失败，runner 自动回滚。

不支持滚动升级——表名变更瞬间，旧版代码无法运行。MVP 阶段单实例部署，停机窗口可接受。

down 迁移：反向 RENAME 回原表名。

### 9.2 迁移 022：新表 + 新列

```sql
-- 022_v2_user_feedback_and_risk_preference.up.sql

-- New table: user feedback for analysis quality tracking (PRD SS6.3)
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

-- risk_preference column already exists (migration 007, default 'neutral', CHECK: conservative/neutral/aggressive)
-- v2 changes: default -> 'moderate', CHECK values -> conservative/moderate/aggressive (PRD SS7.6)
ALTER TABLE rm_users DROP CONSTRAINT IF EXISTS chk_users_risk_preference;
ALTER TABLE rm_users ALTER COLUMN risk_preference SET DEFAULT 'moderate';
UPDATE rm_users SET risk_preference = 'moderate' WHERE risk_preference = 'neutral';
ALTER TABLE rm_users ADD CONSTRAINT chk_users_risk_preference
    CHECK (risk_preference IN ('conservative', 'moderate', 'aggressive'));

-- New column: email push opt-out flag
ALTER TABLE rm_users ADD COLUMN email_push_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- New column: subscription plan preset (PRD SS14.2)
ALTER TABLE rm_users ADD COLUMN plan VARCHAR(16) NOT NULL DEFAULT 'invite';

-- New column: disclaimer acceptance timestamp (PRD SS12)
ALTER TABLE rm_users ADD COLUMN disclaimer_accepted_at TIMESTAMPTZ;

-- v2 decision card columns (PRD SS8.1, SS17)
-- v1 columns preserved for historical card read-only access (PRD SS3.9)
ALTER TABLE rm_decision_cards ADD COLUMN action VARCHAR(32);
ALTER TABLE rm_decision_cards ADD COLUMN action_label VARCHAR(128);
ALTER TABLE rm_decision_cards ADD COLUMN scenarios JSONB;
ALTER TABLE rm_decision_cards ADD COLUMN stop_loss DECIMAL(20,6);
ALTER TABLE rm_decision_cards ADD COLUMN take_profit DECIMAL(20,6);
ALTER TABLE rm_decision_cards ADD COLUMN valid_days INT;
ALTER TABLE rm_decision_cards ADD COLUMN concentration_level VARCHAR(16);
ALTER TABLE rm_decision_cards ADD COLUMN concentration_message TEXT;
ALTER TABLE rm_decision_cards ADD COLUMN default_action TEXT;
ALTER TABLE rm_decision_cards ADD COLUMN no_trigger_note TEXT;
ALTER TABLE rm_decision_cards ADD COLUMN model_version VARCHAR(32);
```

down 迁移：

```sql
-- 022_v2_user_feedback_and_risk_preference.down.sql
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS model_version;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS no_trigger_note;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS default_action;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS concentration_message;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS concentration_level;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS valid_days;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS take_profit;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS stop_loss;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS scenarios;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS action_label;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS action;
ALTER TABLE rm_users DROP COLUMN disclaimer_accepted_at;
ALTER TABLE rm_users DROP COLUMN plan;
ALTER TABLE rm_users DROP COLUMN email_push_enabled;
-- Restore v1 risk_preference: default back to 'neutral', CHECK back to conservative/neutral/aggressive
ALTER TABLE rm_users DROP CONSTRAINT IF EXISTS chk_users_risk_preference;
UPDATE rm_users SET risk_preference = 'neutral' WHERE risk_preference = 'moderate';
ALTER TABLE rm_users ALTER COLUMN risk_preference SET DEFAULT 'neutral';
ALTER TABLE rm_users ADD CONSTRAINT chk_users_risk_preference
    CHECK (risk_preference IN ('conservative', 'neutral', 'aggressive'));
DROP TABLE IF EXISTS rm_user_feedback;
```

### 9.3 迁移执行时序

```
Phase 0（richman 先行）:
  migration 021: rm_ prefix rename
  migration 022: new table + new column
  migration 023: invite system (rm_user_invite_codes, rm_invite_rewards, rm_users login_streak/last_login_date)
  -> richman redeploy with updated SQL

Phase 0.5（seed 数据更新）:
  更新 db/seed/asset_catalog.sql：追加 PRD SS2.1 的全部标的
  （A 股宽基/行业 ETF、美股宽基 ETF 等），is_active = false（置灰），
  确保 rm_asset_catalog 在 Market Overview 上展示完整分类卡片墙

Phase 1（richson 上线后）:
  richson Alembic: create rs_* tables
  richson backfill: 90-day historical data
```

richman 的 021/022/023 与 richson 的 Alembic 迁移互不依赖（操作不同的表），但建议先完成 richman 迁移再启动 richson。migration 023 的完整 schema 定义见 invite-system-trd.md SS9。

## 10. 配置变更

### 10.1 新增 Config 结构

```go
// RichsonConfig holds richson sidecar connection settings.
type RichsonConfig struct {
    BaseURL      string        // RICHSON_BASE_URL
    APIKey       string        // RICHSON_API_KEY
    AsyncTimeout time.Duration // RICHSON_ASYNC_TIMEOUT_MS, default 5000
    SyncTimeout  time.Duration // RICHSON_SYNC_TIMEOUT_MS, default 30000
    LightTimeout time.Duration // RICHSON_LIGHT_TIMEOUT_MS, default 10000
}

// PlatformLLMConfig holds platform-level LLM settings for cron-triggered analysis.
type PlatformLLMConfig struct {
    Provider string // PLATFORM_LLM_PROVIDER, default "claude"
    Model    string // PLATFORM_LLM_MODEL
    APIKey   string // PLATFORM_LLM_API_KEY
}
```

在主 Config struct 中新增：

```go
type Config struct {
    // ... existing fields ...
    Richson     RichsonConfig
    PlatformLLM PlatformLLMConfig
}
```

### 10.2 新增环境变量

```env
# richson sidecar
RICHSON_BASE_URL=http://localhost:8001
RICHSON_API_KEY=change-me-in-production

# richson call timeouts (milliseconds)
RICHSON_ASYNC_TIMEOUT_MS=5000
RICHSON_SYNC_TIMEOUT_MS=30000
RICHSON_LIGHT_TIMEOUT_MS=10000

# Platform LLM for cron-triggered analysis (asset-level)
PLATFORM_LLM_PROVIDER=claude
PLATFORM_LLM_MODEL=claude-sonnet-4-20250514
PLATFORM_LLM_API_KEY=sk-...
```

### 10.3 .env.example 更新

在现有 `.env.example` 末尾追加上述变量模板，注释标注 "v2 新增"。

### 10.4 启动检查

richman 启动时新增以下检查：

1. **RICHSON_BASE_URL 非空**：必需，缺失则启动失败
2. **RICHSON_API_KEY 非空**：必需
3. **PLATFORM_LLM_API_KEY 非空**：必需（cron 触发分析需要平台 LLM key）
4. **richson 连通性**：异步检查（goroutine），不阻塞启动，仅 WARN 日志

## 11. Model 层新增

新增 Go struct 对应 v2 数据结构，放在 `internal/model/` 下：

```go
// v2_analysis.go

// AssetAnalysis maps to rs_asset_analyses (read-only from richman).
type AssetAnalysis struct {
    AssetAnalysisID   int64
    AssetCode         string
    Locale            string
    OverallScore      float64
    SignalLevel       string
    Confidence        float64
    ConfidenceBandLow float64
    ConfidenceBandHigh float64
    ModelVersion      string
    MarketInterpretation string
    RiskFactors       json.RawMessage
    RegimeSummary     string
    D1Score           *float64
    D1BaseScore       *float64
    D1LLMAdjustment   *float64
    // ... D2, D3, D4 same pattern
    D1Weight          float64
    D2Weight          float64
    D3Weight          float64
    D4Weight          float64
    LLMSkipped        bool
    DataCoverage      string
    ConflictType      *string
    ConflictMessage   *string
    PrevAnalysisID    *int64
    ScoreDelta        *float64
    ChangeSummary     *string
    MajorChangeRecap  *string
    DataSnapshotAt    time.Time
    UsdExchangeRate   *float64        // CNY/USD snapshot; NULL for USD assets
    PriceAtAnalysis   *float64
    DemoPlan          json.RawMessage
    AnalysisMetadata  json.RawMessage // extensible JSONB (drawdownReference, etc.)
    GeneratedBy       string
    Source            string
    JobID             *string
    AnalyzedAt        time.Time
    CreatedAt         time.Time
    UpdatedAt         time.Time
    IsDeleted         int
}

// AnalysisDimension maps to rs_asset_analysis_dimensions (read-only).
type AnalysisDimension struct {
    ID                 int64
    AssetAnalysisID    int64
    Dimension          string
    SubIndicator       string
    RawValue           *float64
    Percentile1Y       *float64
    Percentile5Y       *float64
    BlendedPercentile  *float64
    NormalizedScore    *float64
    WeightInDimension  *float64
    DataSource         *string
    DataAsOf           *time.Time
}

// AnalysisJob maps to rs_analysis_jobs (read-only).
type AnalysisJob struct {
    JobID             string
    AssetCode         string
    JobType           string
    Status            string
    Progress          float64
    CurrentStep       *string
    Steps             json.RawMessage
    ErrorMessage      *string
    ErrorCode         *string
    AssetAnalysisID   *int64
    ExpiresAt         time.Time
    StartedAt         *time.Time
    CompletedAt       *time.Time
    RequestID         *string
    Locale            *string
    CreatedAt         time.Time
}

// EventAlert maps to rs_event_alerts.
type EventAlert struct {
    ID               int64
    EventSlug        string
    EventTitle       string
    Source           string
    PrevProbability  float64
    CurrProbability  float64
    Delta            float64
    Threshold        float64
    GoldDirection    *string
    Alerted          bool
    DetectedAt       time.Time
}

// UserFeedback maps to rm_user_feedback.
type UserFeedback struct {
    FeedbackID      int64
    UserID          int64
    AssetAnalysisID int64
    Rating          string
    Comment         *string
    CreatedAt       time.Time
}
```

## 12. v1 代码废弃计划

### 12.1 废弃模块清单

| 模块 | 文件数 | 当前职责 | v2 替代 | 废弃阶段 |
|------|--------|----------|---------|----------|
| internal/analysis/ | ~15 | LLM 调用、分析 pipeline、置信度、推荐生成 | richson 全权承担 | Phase 1 冻结，Phase 3 删除 |
| internal/llm/ | ~12 | Claude/OpenAI provider 抽象、加解密、SSRF 防护 | richson ADK | Phase 1 冻结（crypto.go 保留用于解密传递），Phase 3 删除 |
| internal/datasource/ | ~8 | Yahoo/AKShare/Polymarket 数据获取 | richson datasources/ | Phase 1 冻结，Phase 3 删除 |
| internal/service/analysis/ | 已有文件 | v1 分析触发与合成 | v2_holding.go 新增；v1 逻辑不修改 | Phase 1 v1 方法冻结，Phase 3 删除 |

### 12.2 保留模块

以下模块在 v2 中继续使用：

| 模块 | 说明 |
|------|------|
| internal/llm/crypto.go | 用户 LLM API Key 加解密，持仓级分析需解密后传递给 richson |
| internal/api/v1/ | v1 端点保留运行（持仓 CRUD、认证等不变的端点） |
| internal/notification/ | v1 channel 推送保留，v2 平台推送新增 emailpush |
| internal/service/schedule/ | cron 调度器保留，新增 v2 任务 |

### 12.3 废弃标记策略

Phase 1 上线时，废弃模块的公开函数/方法添加 `// Deprecated: v2 uses richson. Will be removed in Phase 3.` 注释。不添加 `go:generate` 或编译期断言——纯注释标记，代码仍可编译运行。

## 13. 依赖注入与启动顺序

### 13.1 新增依赖关系

```
main.go / cmd/server.go
  -> config.Load()
  -> db.Connect()
  -> richson.NewClient(cfg.Richson, logger)           // NEW
  -> repo.NewAssetAnalysisReadRepo(db)                // NEW
  -> repo.NewAnalysisDimensionReadRepo(db)            // NEW
  -> repo.NewAnalysisJobReadRepo(db)                  // NEW
  -> repo.NewEventAlertReadRepo(db)                   // NEW
  -> repo.NewUserFeedbackRepo(db)                     // NEW
  -> service.NewMarketService(assetRepo, analysisRepo, dimensionRepo)   // NEW
  -> service.NewBriefingService(holdingRepo, analysisRepo, cardRepo)    // NEW
  -> service.NewFeedbackService(feedbackRepo)                           // NEW
  -> emailpush.NewService(userRepo, analysisRepo, ...)                  // NEW
  -> v2.NewMarketHandler(richsonClient, marketSvc)                      // NEW
  -> v2.NewAnalysisHandler(richsonClient, analysisSvc)                  // NEW
  -> v2.NewBriefingHandler(briefingSvc)                                 // NEW
  -> v2.NewFeedbackHandler(feedbackSvc)                                 // NEW
  -> v2.NewUserHandler(userSvc)                                         // NEW
  -> v2.NewEventHandler(richsonClient)                                  // NEW
  -> RegisterV2CronJobs(cron, richsonClient, emailPushSvc, ...)         // NEW
  -> router setup (v1 + v2 groups)
  -> server.Start()
```

### 13.2 路由注册

```go
// v1 routes (preserved)
v1Group := router.Group("/api/v1")
// ... existing v1 handler registration ...

// v2 routes (new)
v2Group := router.Group("/api/v2")

// public endpoints with IP rate limit
v2Public := v2Group.Group("", rateLimitMiddleware)
marketGroup := v2Public.Group("/market")
marketHandler.RegisterRoutes(marketGroup)
eventGroup := v2Public.Group("/events")
eventHandler.RegisterRoutes(eventGroup)

// authenticated endpoints
v2Auth := v2Group.Group("", authMiddleware)
analysisGroup := v2Auth.Group("/analysis")
analysisHandler.RegisterRoutes(analysisGroup)
briefingHandler.RegisterRoutes(v2Auth) // GET /briefing
feedbackHandler.RegisterRoutes(v2Auth) // POST /feedback
userGroup := v2Auth.Group("/user")
userHandler.RegisterRoutes(userGroup)
```

## 14. 错误处理

### 14.1 v2 错误码

在现有错误格式基础上新增 v2 特有错误码：

| 错误码 | HTTP 状态 | 触发场景 |
|--------|-----------|----------|
| RICHSON_UNAVAILABLE | 503 | richson 健康检查失败或连接超时 |
| RICHSON_ERROR | 502 | richson 返回非预期错误 |
| ANALYSIS_NOT_FOUND | 404 | 标的无分析数据（尚未运行过分析） |
| JOB_NOT_FOUND | 404 | job ID 不存在 |
| ASSET_NOT_FOUND | 404 | 标的代码不在 rm_asset_catalog 中 |
| INVALID_RISK_PREFERENCE | 400 | risk_preference 不是 conservative/moderate/aggressive |
| FEEDBACK_DUPLICATE | 409 | 同一用户对同一分析重复提交反馈 |
| RATE_LIMIT_EXCEEDED | 429 | IP 限流超限 |

### 14.1.1 v2 响应格式

v2 端点沿用 v1 的统一响应包装：

- 成功：`{ "data": { ... } }`
- 列表：`{ "data": [...], "pagination": { "page": 1, "pageSize": 20, "total": 100 } }`
- 错误：`{ "error": { "code": "ERROR_CODE", "message": "...", "details": [] } }`

字段命名：camelCase。时间格式：ISO 8601（UTC）。

### 14.2 richson 错误透传

代理端点透传 richson 的错误响应（保持原始错误码和 HTTP 状态码），不做二次包装。聚合端点将 richson 错误映射为 richman 错误码（如 richson 返回 502 -> richman 返回 RICHSON_ERROR + 502）。

## 15. v1/v2 API 共存策略

### 15.1 路由共存

v1（`/api/v1/*`）和 v2（`/api/v2/*`）路由组并行注册，互不影响。前端在切换过程中可同时调用 v1 和 v2 端点。

### 15.2 v1 端点保留清单

| v1 端点 | v2 状态 | 说明 |
|---------|---------|------|
| /api/v1/auth/* | 保留 | 认证逻辑基本不变，注册端点新增 disclaimerAccepted 校验（见下） |
| /api/v1/holdings/* | 保留 | 持仓 CRUD 逻辑不变（表名更新为 rm_） |
| /api/v1/trades/* | 保留 | 交易记录逻辑不变 |
| /api/v1/assets/* | 保留 | 标的目录查询不变 |
| /api/v1/settings/* | 保留 | LLM 配置等设置不变 |
| /api/v1/notifications/* | 保留 | 用户通知渠道管理不变 |
| /api/v1/analysis/trigger | Phase 1 废弃 | 替代: /api/v2/analysis/trigger-asset |
| /api/v1/tasks/:taskId | Phase 1 废弃 | 替代: /api/v2/analysis/jobs/{jobId} |
| /api/v1/decision-cards/* | 冻结只读 | v1 历史卡片保留只读访问 |
| /api/v1/onboarding/* | Phase 2 废弃 | v2 零 onboarding |
| /api/v1/dashboard/* | Phase 2 废弃 | 替代: /api/v2/market/overview + /api/v2/briefing |

### 15.2.1 v1 注册端点免责声明校验

v2 新增免责声明确认要求（PRD SS12，前端 TRD SS14）。v1 注册端点 `POST /api/v1/auth/register` 请求体新增可选字段：

```json
{
  "email": "...",
  "password": "...",
  "inviteCode": "...",
  "disclaimerAccepted": true
}
```

后端校验逻辑：
1. `disclaimerAccepted` 为 `true` 时，写入 `rm_users.disclaimer_accepted_at = NOW()`
2. `disclaimerAccepted` 为 `false` 或缺失时，返回 400 "必须接受免责声明才能注册"
3. 存量用户（v1 注册的）`disclaimer_accepted_at` 为 NULL，不影响登录和使用

### 15.3 废弃端点处理

废弃端点在 Phase 1 上线时添加响应头 `Deprecation: true` 和 `Sunset: <date>`（RFC 8594），帮助前端开发者识别。功能正常运行直到 Phase 3 删除。

## 16. 集中度计算

集中度计算在 richman 侧完成（因为需要用户全部持仓数据），结果用于两个场景：

1. 投研简报卡片的 concentrationLevel/concentrationMessage（BriefingService）
2. 持仓级分析时传入 richson 的 peerExposure 参数

计算逻辑：

```go
// ComputeConcentration calculates concentration level for a given asset type.
// totalExposure = sum of position_ratio for all holdings of the same asset_type.
// riskPreference affects the blue-tier threshold (PRD SS7.6).
func ComputeConcentration(totalExposure float64, riskPreference string) (level string, message string) {
    // Thresholds (PRD SS8.3):
    // >= 35%: red
    // >= 25%: orange
    // >= blue_threshold: blue (10%/15%/20% per risk preference)
    // else: none
}
```

## 17. 决策卡片持久化

v2 持仓级分析结果由 richman 持久化到 rm_decision_cards 表。richson 返回的 HoldingAnalysisResponse 映射为决策卡片记录：

| richson 返回字段 | rm_decision_cards 列 | 说明 |
|-----------------|---------------------|------|
| action | action | 操作方向（hold/add/reduce 等） |
| actionLabel | action_label | 显示文本（"逢回调加仓"） |
| defaultAction | default_action | 无场景触发时的默认建议（PRD SS8.1） |
| scenarios (JSON) | scenarios | JSONB 存储完整场景列表 |
| stopLoss | stop_loss | 止损价 |
| takeProfit | take_profit | 止盈价 |
| validDays | valid_days | 有效期天数 |
| noTriggerNote | no_trigger_note | 到期未触发时的说明文本 |
| -- | concentration_level | 集中度级别（richman 计算后附加） |
| -- | concentration_message | 集中度提示文本（richman 计算后附加） |
| -- | model_version | 生成时的模型版本（从 rs_asset_analyses 读取） |

新增列通过 migration 022 添加（SS9.2）。v1 历史卡片的新列为 NULL，不影响只读展示。

每次持仓级分析生成新卡片记录（INSERT），不更新旧卡片——历史卡片永不重算（PRD SS3.9）。

## 18. richson 不可用时的降级行为

v2 端点在 richson 不可用时的行为因端点类型不同：

| 端点类型 | richson 不可用时行为 | 原因 |
|----------|---------------------|------|
| 代理端点（regime/ohlcv/scores/radar/trigger-asset/demo-plan） | 返回 503 RICHSON_UNAVAILABLE | 数据必须来自 richson，无法降级 |
| 聚合端点（overview/asset detail） | 正常返回 | 数据来自 DB（rs_* 表），不依赖 richson 实时调用。若 rs_asset_analyses 无数据（首次部署未完成分析），返回标的元信息 + `analysis: null`，前端展示"分析数据准备中" |
| 聚合端点（briefing） | 正常返回 | 同上，数据来自 DB |
| 聚合端点（holding analysis） | 返回 503 | 需要实时调用 richson 生成执行计划 |
| 直写端点（feedback/risk-preference） | 正常返回 | 纯 richman DB 操作 |

不对代理端点做响应缓存——缓存过期的市场数据可能误导用户决策，503 是更负责任的响应。

## 19. 可观测性

### 19.1 日志规范

v2 handler 和 service 的日志遵循现有 zap 结构化日志规范（docs/standards/logging.md），新增以下日志字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| api_version | string | "v2" |
| endpoint | string | 如 "market_overview", "holding_analysis" |
| richson_call | string | richson 端点路径（仅涉及 richson 调用时） |
| richson_latency_ms | int | richson 调用耗时 |
| richson_status | int | richson 响应状态码 |

### 19.2 关键监控指标

通过日志聚合驱动监控告警：

| 指标 | 条件 | 级别 |
|------|------|------|
| v2 API 5xx 率 | > 5% 持续 5 分钟 | ERROR |
| richson 调用失败率 | > 50% 持续 5 分钟 | ERROR |
| richson 平均延迟 | > 10s 持续 10 分钟 | WARN |
| 每日简报发送失败 | > 10% 用户发送失败 | ERROR |
| cron 任务超时 | 任务执行时间超过预设超时 | WARN |

### 19.3 richson 调用日志

每次 richson HTTP 调用记录：请求方法、URL、request_id、响应状态码、耗时、是否重试。LLM API key 不记录在日志中（PLATFORM_LLM_API_KEY 和用户 key 均脱敏）。

## 20. 目录结构变更总览

注：email 模板文件按 i18n 约定分 locale 后缀。

```
backend/
  internal/
    richson/                  # NEW: richson HTTP client
      client.go
      types.go                # request/response structs
    api/
      v1/                     # EXISTING: preserved
      v2/                     # NEW: v2 handlers
        market.go
        analysis.go
        briefing.go
        feedback.go
        user.go
        event.go
        middleware/
          ratelimit.go        # IP rate limiting
    service/
      market/                 # NEW
        service.go
      briefing/               # NEW
        service.go
      feedback/               # NEW
        service.go
      emailpush/              # NEW
        service.go
        template/
          engine.go
          daily_briefing_zh.html
          daily_briefing_en.html
          weekly_insight_zh.html
          weekly_insight_en.html
          market_alert_zh.html
          market_alert_en.html
          holding_suggestion_zh.html
          holding_suggestion_en.html
      analysis/
        v2_holding.go         # NEW: v2 holding analysis flow
      schedule/
        v2_cron.go            # NEW: v2 cron task registration
    repo/
      asset_analysis_read_repo.go       # NEW
      analysis_dimension_read_repo.go   # NEW
      analysis_job_read_repo.go         # NEW
      event_alert_repo.go               # NEW
      user_feedback_repo.go             # NEW
    model/
      v2_analysis.go          # NEW: v2 model structs
    config/
      config.go               # MODIFIED: add RichsonConfig, PlatformLLMConfig
  db/
    migration/
      021_rename_tables_rm_prefix.up.sql    # NEW
      021_rename_tables_rm_prefix.down.sql  # NEW
      022_v2_user_feedback_and_risk_preference.up.sql   # NEW
      022_v2_user_feedback_and_risk_preference.down.sql # NEW
    seed/
      asset_catalog.sql       # MODIFIED: 追加 v2 新标的
```

## 21. 账户删除与数据清理

PRD SS13.4 要求"用户删除账户时，持仓数据在 30 天内彻底删除"。

### 21.1 账户删除 API

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| DELETE | /api/v1/auth/account | JWT | 用户自助注销账户 |

请求体：`{ "password": "current-password" }` -- 必须提供当前密码做二次确认，防止 token 泄露导致账户被删。

流程：
1. 验证 JWT，确认操作者身份
2. 验证请求体中的 password 与数据库中的密码 hash 匹配
3. 密码不匹配返回 403
4. 将 rm_users 表对应记录设置 `is_deleted = 1`，`updated_at = NOW()`
5. 记录 `deleted_at`（复用 updated_at 作为删除时间标记）
6. 不立即删除关联数据——延迟清理，给用户后悔窗口
7. 返回 204 No Content
8. 后续登录请求对已删除账户返回 401

### 21.2 数据清理 Cron

新增定时任务（每日 03:00 UTC+8 执行）：

```sql
-- Hard delete user data older than 30 days
-- Step 1: find users soft-deleted > 30 days ago
SELECT user_id FROM rm_users
WHERE is_deleted = 1 AND updated_at < NOW() - INTERVAL '30 days';

-- Step 2: cascade hard delete (per user, in transaction)
DELETE FROM rm_user_feedback WHERE user_id = $1;
DELETE FROM rm_decision_cards WHERE user_id = $1;
DELETE FROM rm_trades WHERE user_id = $1;
DELETE FROM rm_holdings WHERE user_id = $1;
DELETE FROM rm_notification_logs WHERE user_id = $1;
DELETE FROM rm_notification_channels WHERE user_id = $1;
DELETE FROM rm_llm_configs WHERE user_id = $1;
DELETE FROM rm_user_schedule_settings WHERE user_id = $1;
DELETE FROM rm_holding_schedule_overrides WHERE user_id = $1;
DELETE FROM rm_users WHERE user_id = $1;
```

每个用户的数据在单独事务中删除。单个用户删除失败不影响其他用户。BYOK 的 LLM API Key 随 rm_llm_configs 记录一起物理删除。

**邮箱重注册**：rm_users 的 email 唯一约束应为 partial unique index `WHERE is_deleted = 0`，允许已删除账户的邮箱重新注册。

**invite 系统表清理**：hard delete 中需包含 invite-system-trd 定义的表（按外键依赖顺序）：
```sql
DELETE FROM rm_invite_rewards WHERE user_id = $1;
DELETE FROM rm_user_invite_codes WHERE user_id = $1;
```
放在 rm_user_feedback 删除之后。

**索引支持**：需创建 `CREATE INDEX idx_rmu_deleted_at ON rm_users (updated_at) WHERE is_deleted = 1` 支持定期清理查询。

## 22. 已知问题与编码阶段必须处理项

以下问题已在设计审查中识别，必须在编码阶段解决，不可跳过。

### 22.1 CORS 生产环境安全

现有 cors.go 在生产模式下直接回显请求 Origin 头为 `Access-Control-Allow-Origin`，等同于通配符。携带 JWT 的跨域请求可被任意恶意站点发起。

处理方案：改为白名单校验，仅允许 `richman.app` 等合法域名。白名单从环境变量 `CORS_ALLOWED_ORIGINS` 读取。

### 22.2 richman 健康检查端点缺失

TRD SS3.5 设计了 `GET /health`，但当前代码库中没有实现。docker-compose 也没有 healthcheck 配置。

处理方案：实现 `/health` 端点，检查 PostgreSQL 连接（`pool.Ping(ctx)`）。docker-compose 中添加 healthcheck。richson 依赖 richman 先于自身启动，用 `depends_on.condition: service_healthy`。

### 22.3 Cron 优雅关闭

richman 收到 SIGTERM 时已有 `srv.Shutdown` 处理 HTTP 请求，但新增的 cron 任务（每日分析、简报邮件、事件轮询）使用 `sync.Mutex + TryLock`。正在执行的 cron job 在 SIGTERM 时会被强制中断，可能导致 rs_analysis_jobs 永久卡在 running 状态。

处理方案：在 main.go 的 shutdown 流程中调用 `cronScheduler.Stop()` 等待当前 job 完成（`robfig/cron` 的 `Stop()` 返回 `context` 可用于等待），设置最长等待时间（如 60s）。

### 22.4 analysis job 卡死恢复

若 richson 进程崩溃导致 job 停在 running 状态且未超过 expires_at，在下一次 10 分钟清理前该标的被唯一索引锁死，最长阻塞 1 小时。

处理方案：将过期清理间隔从 10 分钟缩短到 3 分钟。richson 侧使用 `UPDATE ... WHERE status = 'running' RETURNING *` 做乐观锁，返回空则放弃写入。

### 22.5 密码策略强化

现有校验仅 `min=6, max=128`（auth.go），无复杂度要求。

处理方案：最低 8 位，至少包含大小写字母和数字。在 Register 和 ChangePassword handler 中统一校验。

### 22.6 JWT Refresh Token 机制

四份 TRD 均未涉及 JWT refresh token 和 token 过期策略。

处理方案：MVP 阶段 JWT 有效期 7 天，无 refresh token。编码时在 JWT 生成处注释标明：Phase 2 需引入 refresh token 机制（短期 access token 15min + 长期 refresh token 30 天）。

### 22.7 CSP 安全头部

四份 TRD 和现有代码均未设置 Content-Security-Policy 头部。

处理方案：在 Gin 中间件或 nginx 反向代理层添加基本 CSP 头部：`default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;`。具体策略在部署阶段根据实际依赖的外部资源（CDN、字体等）调整。

### 22.8 邀请码暴力破解防护

邀请码 8 位大写字母数字（36^8 空间），注册端点已有 IP 限流（每分钟 5 次），但未对邀请码校验失败单独计数。

处理方案：邀请码校验失败累计 N 次（如 10 次/小时）后锁定该码 15 分钟，防止分布式 IP 枚举。锁定状态存内存 map（与现有 rate limiter 一致）。

### 22.9 SS9.1 端点表与路由树不一致

SS4.1 路由树中有 `PATCH /api/v2/user/email-push`、`GET /api/v2/market/{code}/share`、invite 端点，但 SS9.1 端点详情表未全部列出。

处理方案：编码时以 SS4.1 路由树为权威清单。SS9.1 端点表中 richman 自有端点的详细设计在对应 service 章节中已覆盖（email-push 见 SS7.1、share 见 invite-system-trd SS6）。

### 22.10 hard delete 中 invite 表悬空引用

`rm_user_invite_codes.used_by_user_id` 引用被删用户，但无 FK 约束也无清理。`rm_invite_rewards.source_invite_id` 引用 `rm_user_invite_codes.invite_code_id`，删除邀请码后 reward 记录悬空。

处理方案：SS21.2 的 hard delete 顺序已包含 invite 表，但需额外处理：`UPDATE rm_user_invite_codes SET used_by_user_id = NULL WHERE used_by_user_id = $1`（保留邀请码记录但清除被邀请人关联），然后再 DELETE 该用户自己的邀请码。

### 22.11 数据库备份策略

四份 TRD 均未涉及备份。richson backfill 90 天需可观计算成本，数据库损坏后无恢复路径。

处理方案：部署时配置 pg_dump 每日备份 + WAL 归档。最低要求：每日全量备份，保留 7 天。配置文档写入 `docs/standards/` 或运维手册。

### 22.12 migration 020 嵌套事务

现有 `020_sequence_start_100000.up.sql` 文件内部包含 `BEGIN;` 和 `COMMIT;` 语句，而 runner.go 已自动用 `pool.Begin()`/`tx.Commit()` 包裹。PostgreSQL 在已有事务中执行 `BEGIN` 会发出 WARNING，内部 `COMMIT` 会提交外层事务，导致 runner 的后续 `tx.Commit()` 作用在已提交的连接上。

处理方案：v2 编码阶段修复 020 的 up/down 文件，移除文件内部的 `BEGIN;`/`COMMIT;`。020 已经执行过的环境不受影响（序列值已设置），修复仅防止 down->up 重跑时的语义异常。

### 22.13 plan 列与 plan_id 列语义混淆

migration 022 新增 `rm_users.plan VARCHAR(16)` 存储订阅等级（如 'invite'/'free'），但 rm_users 表已有 `plan_id BIGINT` 外键指向 rm_plans 表。两列在名称上易混淆。

处理方案：编码阶段将新列改名为 `subscription_tier VARCHAR(16)`（或 `subscription_plan`），避免与现有 `plan_id` 列名冲突。migration 022 脚本同步修改。

### 22.14 score alert 与 market alert 去重边界

SS8.3.3 每日 06:00 市场快讯推送标的价格变动，SS8.4 每日 15:30 得分变化推送也包含价格信息。两条通知路径对同一标的可能产生重复感知（用户在 06:00 已收到价格变动快讯，15:30 又收到得分变化通知）。当前去重逻辑仅在 SS8.4 中按 `rm_notification_logs` 查询当日 `market_alert` 类型记录跳过已推送标的，但未考虑反向去重（SS8.3.3 是否应跳过当日已有 `score_change` 通知的标的）。

处理方案：编码阶段确认单向去重是否足够（15:30 跳过 06:00 已推送的标的），还是需要双向去重。若单向足够，在代码注释中明确说明设计决策和理由。

### 22.15 认证端点用户级限流

当前限流仅在 IP 级（公开端点 60 次/分钟、auth 端点 5 次/分钟）。认证后的端点（`POST /analysis/trigger-asset`、`POST /analysis/trigger-holding`、`POST /feedback`）缺少用户级限流。恶意用户可通过高频调用 trigger 端点大量消耗 richson 计算资源和 LLM token。

处理方案：对 trigger-asset 和 trigger-holding 端点增加用户级限流（如每用户每小时 10 次手动触发）。feedback 端点限流较宽松（每用户每分钟 5 次）。限流状态存内存 map，与现有 IP rate limiter 共用基础设施。
