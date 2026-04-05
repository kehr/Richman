# 后端编码规范

## 技术栈

- Go 1.22+
- Gin（Web 框架）
- sqlc（类型安全 SQL 生成）
- PostgreSQL
- pino 风格结构化日志（Go 对应 slog 或 zerolog）
- Zod 对应 Go 验证库（如 go-playground/validator）


## 三层架构

```
API handlers -> Service -> Repo -> DB (sqlc)
```

### 层间依赖规则

| 层 | 可依赖 | 不可依赖 |
|---|--------|---------|
| API handlers（api/v1/） | service、config、middleware、schemas | repo、db、datasource |
| Service（service/） | 本域 repo、config、外部服务（llm/datasource）、其他域 service（单向） | API handlers、其他域 repo |
| Repo（repo/） | sqlc 生成代码、model | service、API handlers |
| 基础设施（config/llm/datasource/） | config、stdlib、第三方库 | service、repo、API handlers |

### 跨域依赖

- 允许 service 之间单向引用（不可循环）
- 不可反向引用
- 不可跨域直接访问 repo


## 目录结构

```
backend/
  cmd/
    server/main.go          # HTTP 服务入口
  internal/
    api/                     # HTTP 层
      middleware/             # 中间件（auth、plan-check、cors、error-handler）
      v1/                    # v1 路由处理器
        portfolio.go
        analysis.go
        auth.go
        notification.go
        decision_card.go
    service/                 # 业务逻辑层
      portfolio/             # 持仓管理
      analysis/              # 分析编排
      notification/          # 推送编排
      auth/                  # 认证逻辑
    repo/                    # 数据访问层（sqlc 生成 + 手写扩展）
    model/                   # 领域模型
    analysis/                # 三维分析引擎
      trend/                 # 趋势维度（量化）
      position/              # 位置维度（量化）
      catalyst/              # 催化剂维度（量化 + LLM）
      synthesis/             # LLM 综合层
      weight/                # 权重管理
    notification/            # 推送通知中心
      dispatcher.go          # 统一调度器
      adapter/               # 可插拔渠道适配器
        wechat/
        feishu/
        email/
    llm/                     # LLM Provider 抽象
      provider.go            # 统一接口
      claude/
      openai/
    datasource/              # 数据源集成
      akshare/
      yahoo/
      polymarket/
    config/                  # 配置管理
      config.go              # 集中式配置（不直接读 os.Getenv）
  db/
    query/                   # SQL 查询（sqlc 输入）
    migration/               # 数据库迁移
    sqlc.yaml                # sqlc 配置
  go.mod
  go.sum
```


## API Handler 层

**职责：** 参数校验、调用 service、格式化响应。不包含业务逻辑。

```go
// internal/api/v1/portfolio.go
func (h *PortfolioHandler) ListHoldings(c *gin.Context) {
	userID := middleware.GetUserID(c)
	holdings, err := h.portfolioService.ListHoldings(c.Request.Context(), userID)
	if err != nil {
		c.JSON(mapError(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": holdings})
}
```

**规则：**
- 使用 struct binding tag 做参数校验，不手动解析
- 不需要 handler 级 try-catch，全局错误中间件兜底
- 返回明确的 HTTP 状态码（404、409 等），不统一用 500


## Service 层

**职责：** 业务逻辑编排，协调多个 repo 和外部服务。

```go
// internal/service/analysis/service.go
type AnalysisService struct {
	repo       *repo.Queries
	trendCalc  *trend.Calculator
	posCalc    *position.Calculator
	catCalc    *catalyst.Calculator
	synthesizer *synthesis.Synthesizer
	config     *config.Config
}

func (s *AnalysisService) AnalyzeHolding(ctx context.Context, holdingID int64) (*model.DecisionCard, error) {
	holding, err := s.repo.GetHolding(ctx, holdingID)
	if err != nil {
		return nil, fmt.Errorf("get holding: %w", err)
	}

	trendResult := s.trendCalc.Calculate(ctx, holding.AssetCode)
	posResult := s.posCalc.Calculate(ctx, holding.AssetCode)
	catResult := s.catCalc.Calculate(ctx, holding.AssetCode)

	card, err := s.synthesizer.Synthesize(ctx, holding, trendResult, posResult, catResult)
	if err != nil {
		return nil, fmt.Errorf("synthesize: %w", err)
	}
	return card, nil
}
```

**规则：**
- 所有方法接收 context.Context 作为第一个参数
- 返回 (result, error)，不 panic
- 不直接写 SQL，通过 repo 函数
- 不使用 fmt.Println，用结构化日志


## Repo 层

**职责：** 纯数据访问，由 sqlc 生成，必要时手写扩展。

**规则：**
- 默认过滤 `is_deleted = 0`
- 需要查已删除记录时通过参数覆盖
- 不包含业务逻辑
- 批量写入用单条 INSERT 多 VALUES


## 配置管理

**所有配置通过 config.Config 结构体管理，不直接读 os.Getenv。**

```go
// internal/config/config.go
type Config struct {
	DatabaseURL    string
	ServerPort     int
	LLM            LLMConfig
	AKShare        AKShareConfig
	Yahoo          YahooConfig
	Polymarket     PolymarketConfig
	Notification   NotificationConfig
	Auth           AuthConfig
}

func Load() (*Config, error) {
	// Read from env vars, validate, return
}
```


## 错误处理

**全局错误中间件：**

| 错误类型 | HTTP 状态码 | 错误码 |
|---------|-----------|--------|
| 参数校验错误 | 400 | VALIDATION_ERROR |
| 未认证 | 401 | UNAUTHORIZED |
| 权限不足（plan 限制） | 403 | PLAN_LIMIT_EXCEEDED |
| 资源不存在 | 404 | NOT_FOUND |
| 业务逻辑冲突 | 409 | CONFLICT |
| 内部错误 | 500 | INTERNAL_ERROR |

**错误包装：** 使用 `fmt.Errorf("context: %w", err)` 保留错误链。

**自定义业务错误：** 定义 AppError 类型携带 HTTP 状态码和错误码。


## 日志

使用结构化日志（slog 或 zerolog），object-first 模式：

```go
slog.Info("analysis completed",
	"holdingID", holdingID,
	"assetCode", assetCode,
	"duration", elapsed,
)
```

**级别使用：**
- Error: 需要人工介入的错误
- Warn: 可恢复的异常（降级、重试）
- Info: 关键业务节点（分析完成、推送发送）
- Debug: 调试用细节


## 定时任务

使用 Go cron 库（如 robfig/cron）调度分析任务：

- 08:30 CST: A 股标的早盘分析
- 15:30 CST: A 股标的 + 黄金收盘分析
- 06:00 CST (次日): 美股标的收盘分析

每次任务遍历所有用户的相关持仓，逐个触发分析并存储结果。


## 安全

- 密码使用 bcrypt 哈希，不存明文
- JWT token 用于认证，设合理过期时间
- API 限流中间件防止滥用
- LLM API Key 等敏感配置只通过环境变量注入
