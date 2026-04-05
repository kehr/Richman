# 日志系统规范

## 技术选型

| 组件 | 选型 | 说明 |
|------|------|------|
| 日志库 | Uber zap | 高性能结构化日志，企业级标配 |
| 日志轮转 | lumberjack | 按大小/时间轮转本地日志文件 |
| 远程采集 | 预留接口 | 后续接入 ELK / Loki / CloudWatch |
| 请求链路 | request ID | 每个请求生成唯一 ID，贯穿全链路 |


## 日志架构

```
                              +---> stdout（开发调试 / Docker logs 采集）
                              |
Application ---> zap logger --+---> 本地文件（lumberjack 轮转）
                              |
                              +---> 远程采集（预留 hook，后续接入 ELK/Loki）
```

### 环境差异

| 环境 | 格式 | 输出目标 | 日志级别 | 采样 |
|------|------|---------|---------|------|
| development | Text (Console) | stdout only | Debug | 关闭 |
| staging | JSON | stdout + 文件 | Info | 开启 |
| production | JSON | stdout + 文件 + 远程 | Info | 开启 |


## 日志级别定义

| 级别 | 用途 | 示例 | 频率要求 |
|------|------|------|---------|
| **Debug** | 开发调试细节，生产不输出 | 量化指标计算中间值、LLM prompt 内容 | 不限 |
| **Info** | 关键业务节点，正常运行记录 | 分析任务完成、推送发送成功、用户登录 | 每个请求 2-5 条 |
| **Warn** | 可恢复的异常，降级处理 | LLM 降级到量化底座、数据源重试、缓存未命中 | 偶发 |
| **Error** | 需要人工关注的错误 | 数据源持续不可用、推送发送失败、数据库连接断开 | 尽量少 |
| **Fatal** | 致命错误，进程退出 | 配置加载失败、数据库初始化失败 | 仅启动阶段 |

**原则：** Info 是生产日志的主体。如果一个日志在正常运行时每秒出现超过 10 次，降为 Debug 或开启采样。


## 日志字段规范

### 必须字段

每条日志必须包含以下上下文字段：

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `ts` | string | ISO 8601 时间戳 | `"2026-04-06T08:30:00.123Z"` |
| `level` | string | 日志级别 | `"info"` |
| `msg` | string | 日志消息（人可读） | `"analysis completed"` |
| `service` | string | 服务名 | `"richman-api"` |
| `env` | string | 运行环境 | `"production"` |

### 请求上下文字段

HTTP 请求链路中的日志额外携带：

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `requestId` | string | 请求唯一 ID（UUID v7） | `"019606f8-..."` |
| `userId` | int64 | 当前用户 ID（认证后） | `10001` |
| `method` | string | HTTP 方法 | `"POST"` |
| `path` | string | 请求路径 | `"/api/v1/analysis/trigger"` |
| `status` | int | 响应状态码 | `200` |
| `latency` | string | 请求耗时 | `"1.234s"` |
| `ip` | string | 客户端 IP | `"203.0.113.1"` |

### 业务上下文字段

根据业务场景附加相关字段：

| 场景 | 字段 | 示例 |
|------|------|------|
| 持仓操作 | `holdingId`, `assetCode` | `20001`, `"sh510300"` |
| 分析任务 | `taskId`, `assetCode`, `dimension` | `"abc123"`, `"sh510300"`, `"trend"` |
| 决策卡 | `cardId`, `recommendation`, `confidence` | `50001`, `"hold"`, `75.5` |
| 推送通知 | `channel`, `recipientId`, `messageType` | `"feishu"`, `10001`, `"pm_digest"` |
| LLM 调用 | `provider`, `model`, `tokens`, `latency` | `"claude"`, `"claude-sonnet-4-5-20250514"`, `1250`, `"2.1s"` |
| 数据源 | `source`, `endpoint`, `latency` | `"akshare"`, `"/stock/zh_index"`, `"0.8s"` |


## 日志格式

### 生产环境（JSON）

```json
{
  "ts": "2026-04-06T08:30:15.123Z",
  "level": "info",
  "msg": "analysis completed",
  "service": "richman-api",
  "env": "production",
  "requestId": "019606f8-1234-7abc-def0-123456789abc",
  "userId": 10001,
  "taskId": "task-20260406-001",
  "assetCode": "sh510300",
  "trendScore": 0.72,
  "positionScore": 0.45,
  "catalystScore": 0.88,
  "confidence": 68.5,
  "recommendation": "small_add",
  "latency": "3.456s"
}
```

### 开发环境（Text Console）

```
2026-04-06T16:30:15.123+0800  INFO  analysis completed  {"requestId": "019606f8...", "assetCode": "sh510300", "confidence": 68.5, "latency": "3.456s"}
```


## 请求链路追踪

### Request ID 中间件

每个 HTTP 请求入口生成唯一 Request ID（UUID v7，时间有序），写入请求上下文和响应头：

```go
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.Must(uuid.NewV7()).String()
		}
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)

		// Create request-scoped logger with requestId
		logger := zap.L().With(zap.String("requestId", requestID))
		c.Set("logger", logger)

		c.Next()
	}
}
```

### Logger 传递

通过 context 或 Gin context 传递 request-scoped logger，保证同一请求的所有日志携带相同 requestId：

```go
// 在 handler 中获取 logger
func (h *Handler) AnalyzeTrigger(c *gin.Context) {
	log := GetLogger(c) // 从 gin.Context 获取携带 requestId 的 logger
	log.Info("analysis triggered", zap.Int64("userId", userID))
	// ...
}

// 在 service 中使用
func (s *Service) Analyze(ctx context.Context, log *zap.Logger, holdingID int64) {
	log.Info("starting analysis", zap.Int64("holdingId", holdingID))
	// ...
}
```


## 日志分类和文件

### 文件分类

| 日志文件 | 内容 | 轮转策略 |
|---------|------|---------|
| `app.log` | 应用主日志（Info+） | 100MB / 文件，保留 30 天，最多 10 个文件 |
| `error.log` | 错误日志（Error+） | 50MB / 文件，保留 90 天，最多 20 个文件 |
| `access.log` | HTTP 访问日志 | 200MB / 文件，保留 14 天，最多 7 个文件 |

### Lumberjack 配置

```go
appWriter := &lumberjack.Logger{
	Filename:   "/var/log/richman/app.log",
	MaxSize:    100,  // MB
	MaxAge:     30,   // days
	MaxBackups: 10,
	Compress:   true,
}

errorWriter := &lumberjack.Logger{
	Filename:   "/var/log/richman/error.log",
	MaxSize:    50,
	MaxAge:     90,
	MaxBackups: 20,
	Compress:   true,
}
```


## 关键业务日志点

### 必须记录（Info 级别）

| 事件 | 必须字段 | 说明 |
|------|---------|------|
| 用户注册 | userId, email | 注册成功 |
| 用户登录 | userId, ip | 登录成功 |
| 登录失败 | email, ip, reason | 密码错误/账户不存在 |
| 持仓变更 | userId, holdingId, action(create/update/delete) | 增删改持仓 |
| 分析任务启动 | taskId, userId, assets[] | 触发分析 |
| 分析任务完成 | taskId, duration, assetsCount | 分析结束 |
| 单标的分析完成 | taskId, assetCode, confidence, recommendation | 单个决策卡生成 |
| 推送发送 | channel, userId, messageType, success | 通知发出 |
| 定时任务触发 | jobName, triggerTime, usersCount | Cron 调度 |

### 必须记录（Warn 级别）

| 事件 | 必须字段 | 说明 |
|------|---------|------|
| LLM 降级 | provider, fallbackReason | LLM 不可用，使用量化底座 |
| 数据源重试 | source, attempt, error | 数据拉取失败重试 |
| 数据陈旧 | source, lastUpdated, staleDuration | 数据超过阈值未更新 |
| Plan 限额接近 | userId, resource, used, limit | 用量达 80% |

### 必须记录（Error 级别）

| 事件 | 必须字段 | 说明 |
|------|---------|------|
| 数据源持续失败 | source, attempts, lastError | 所有重试均失败 |
| 推送失败 | channel, userId, error | 通知发送失败 |
| LLM 调用失败 | provider, model, error, latency | API 错误 |
| 数据库错误 | operation, table, error | 查询/写入异常 |
| 分析任务失败 | taskId, assetCode, error | 分析过程异常 |


## 敏感数据脱敏

日志中禁止输出以下敏感信息：

| 数据 | 处理方式 |
|------|---------|
| 密码 | 永不记录 |
| JWT Token | 仅记录前 8 位 + `...` |
| 邮箱 | 脱敏为 `k***@gmail.com` |
| LLM API Key | 永不记录 |
| 用户投资金额 | 仅记录仓位比例，不记录绝对金额 |

```go
func MaskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***"
	}
	return string(parts[0][0]) + "***@" + parts[1]
}
```


## 性能保护

### 采样策略

生产环境开启 zap 采样，防止高频日志拖垮性能：

```go
samplingConfig := zap.SamplingConfig{
	Initial:    100, // 每秒前 100 条全量记录
	Thereafter: 10,  // 之后每 10 条记录 1 条
}
```

### 异步写入

文件和远程日志使用 zap 的 `BufferedWriteSyncer`，避免 IO 阻塞请求：

```go
bufferedWriter := &zapcore.BufferedWriteSyncer{
	WS:            fileWriter,
	Size:          256 * 1024, // 256KB buffer
	FlushInterval: 5 * time.Second,
}
```

### 日志大小控制

- 单条日志消息不超过 1KB
- 不在日志中输出完整的 LLM prompt/response（仅记录 token 数和延迟）
- 不在日志中输出完整的数据源响应体（仅记录状态码和延迟）


## 监控告警（预留）

基于日志的告警规则（后续接入监控平台时启用）：

| 规则 | 条件 | 级别 |
|------|------|------|
| 错误率激增 | 5 分钟内 Error 数 > 50 | P1 |
| 数据源全部不可用 | 任一 source 连续失败 > 10 次 | P1 |
| 推送批量失败 | 单次推送任务失败率 > 20% | P2 |
| LLM 延迟异常 | 单次调用 > 30 秒 | P2 |
| 分析任务超时 | 单用户分析 > 5 分钟 | P3 |


## Logger 初始化示例

```go
func NewLogger(cfg *config.Config) (*zap.Logger, error) {
	var cores []zapcore.Core

	// Encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Stdout core
	if cfg.Env == "development" {
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel))
	} else {
		jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
		cores = append(cores, zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), zap.InfoLevel))
	}

	// File core (staging + production)
	if cfg.Env != "development" {
		jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

		// App log
		appWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename: cfg.LogDir + "/app.log", MaxSize: 100, MaxAge: 30, MaxBackups: 10, Compress: true,
		})
		cores = append(cores, zapcore.NewCore(jsonEncoder, appWriter, zap.InfoLevel))

		// Error log
		errorWriter := zapcore.AddSync(&lumberjack.Logger{
			Filename: cfg.LogDir + "/error.log", MaxSize: 50, MaxAge: 90, MaxBackups: 20, Compress: true,
		})
		cores = append(cores, zapcore.NewCore(jsonEncoder, errorWriter, zap.ErrorLevel))
	}

	// Build logger
	core := zapcore.NewTee(cores...)
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
		zap.Fields(
			zap.String("service", "richman-api"),
			zap.String("env", cfg.Env),
		),
	)

	return logger, nil
}
```
