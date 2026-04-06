# Step 4: Data Source Integrations

## 任务目标

实现三个外部数据源的 Go 客户端：AKShare（A 股数据）、Yahoo Finance（美股/黄金数据）、Polymarket API（事件概率）。每个客户端包含 HTTP 调用、响应解析、重试逻辑、错误处理。

## 涉及文件路径

### 创建

- `backend/internal/datasource/types.go` -- 数据源通用类型（行情、估值、事件概率）
- `backend/internal/datasource/akshare/client.go` -- AKShare HTTP 客户端
- `backend/internal/datasource/akshare/parser.go` -- 响应数据解析
- `backend/internal/datasource/akshare/client_test.go` -- 测试
- `backend/internal/datasource/yahoo/client.go` -- Yahoo Finance HTTP 客户端
- `backend/internal/datasource/yahoo/parser.go` -- 响应数据解析
- `backend/internal/datasource/yahoo/client_test.go` -- 测试
- `backend/internal/datasource/polymarket/client.go` -- Polymarket API 客户端
- `backend/internal/datasource/polymarket/client_test.go` -- 测试
- `backend/internal/datasource/fetcher.go` -- 统一数据拉取接口和协调器

## PRD/TRD 章节引用

- PRD 2.1-2.4 各标的类型的量化数据和实时信息
- PRD 3.2.6 数据更新频率（全部日级）
- PRD 3.2.7 降级策略（数据源故障处理）
- PRD 5.4 数据源选型
- `docs/standards/backend.md` HTTP 请求、重试、错误处理

## 验证标准

- [ ] AKShare 客户端能拉取沪深 300 ETF 的行情数据（收盘价序列）
- [ ] AKShare 客户端能拉取 PE/PB 估值数据
- [ ] Yahoo Finance 客户端能拉取纳斯达克 100 ETF 的行情数据
- [ ] Yahoo Finance 客户端能拉取黄金 (GLD) 的行情数据
- [ ] Polymarket 客户端能拉取指定市场的事件概率
- [ ] 所有客户端包含重试逻辑（3 次重试 + 指数退避）
- [ ] 数据源故障时返回明确错误类型（可降级判断）
- [ ] `go test ./internal/datasource/...` 全部通过（使用 mock HTTP）
- [ ] `golangci-lint run ./...` 零错误
- [ ] `go vet ./...` 零警告

## 依赖说明

- Step 2 完成（配置管理、日志系统就绪）
- 不依赖 Step 3（数据源独立于持仓管理）
