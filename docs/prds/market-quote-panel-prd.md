# 决策卡实盘行情面板 PRD

## 1 背景与动机

Richman 的决策卡详情页当前展示 AI 分析建议、执行计划和三维推理，但缺少实时市场行情信息。用户在阅读持有/减仓/加仓建议后，需要切换到外部应用（如同花顺、雪球）才能查看当前价格走势，打断了决策流。

本功能在决策卡详情页 Hero 区块下方新增一个「市场行情上下文面板」，将当前价、涨跌、30 日走势与执行计划的关键价位（成本/止损/触发）叠加展示，让用户在同一页面完成"看行情 → 验证决策 → 执行"的闭环。

## 2 目标用户

个人投资者（Richman 目标用户），持有 A 股 ETF、美股、黄金 ETF 等资产，每天或每周查看决策卡并据此做交易决策。

## 3 功能范围

### 3.1 包含

- 决策卡详情页新增市场行情面板
- 后端新增行情查询 API 端点（复用现有 datasource 层）
- 后端服务层 QuoteProvider 抽象 + 120s 内存缓存
- 前端新增 features/market-quote 通用模块
- 前端引入 lightweight-charts（金融图表）+ echarts（统计图表，本次安装但不使用）
- 面板降级态处理（A 股数据源未接入时的折叠态）
- i18n 双语支持

### 3.2 不包含

- 分钟级/tick 级实时行情
- WebSocket 推送
- K 线周期切换（日/周/月）
- MACD/RSI 等技术指标叠加
- 投资组合分析图表（ECharts 仅安装，不实现）
- 用户自定义价位线

## 4 面板形态

### 4.1 正常态（数据源可用）

面板高度约 200px，位于 CardHero 下方、ConclusionBanner 上方。

内容区域：
- 顶部行：现价 + 日涨跌额 + 日涨跌幅% + 更新时间 + 手动刷新按钮
- 第二行：相对分析时价格的变化、相对成本价的变化
- 图表区：30 日日线收盘价折线图，叠加 4 条参考线
  - 成本价：灰色实线
  - 止损线：红色虚线（执行计划的 stopLoss，为 null 时不画）
  - 触发价：橙色虚线（第一个 price 类型触发条件的 priceValue，无 price 触发时不画）
  - 分析时刻：蓝色垂直标记（analyzedAt 日期）

### 4.2 折叠态（数据源不可用）

面板保留但折叠，显示：
- 警告图标 + "X 类资产实盘行情数据源待接入"
- 分析时刻价格（灰色标注 "分析时刻价格，非实时"）+ analyzedAt 时间戳
- 用户可点刷新按钮重试（AKShare sidecar 可能已上线）

### 4.3 加载态

Skeleton 占位，高度与正常态一致。

### 4.4 错误态

错误提示 + 重试按钮，不影响页面其余区块。

## 5 数据新鲜度

- 页面打开时拉取一次行情数据，缓存 120s（TanStack Query staleTime）
- 面板右上角手动刷新按钮，点击后立即绕过缓存重新拉取
- 不轮询、不推送
- 后端同样 120s 内存缓存，减少上游数据源请求

## 6 数据源路由

复用现有 datasource.Fetcher 的路由逻辑：
- `us_stock` → Yahoo Finance
- `gold_etf`（字母代码如 GLD）→ Yahoo Finance
- `gold_etf`（数字代码如 518880）→ AKShare
- `a_share_broad` / `a_share_industry` → AKShare
- 其他 → 返回 source="unavailable"

当上游返回空数据或网络超时时，HTTP 500 + 错误日志。
当 assetType 不在枚举范围内时，HTTP 400。

## 7 API 契约

### 请求

```
GET /api/v1/assets/:assetType/:assetCode/quote
Authorization: Bearer <jwt>
```

### 成功响应（200）

```json
{
  "data": {
    "assetCode": "GLD",
    "assetType": "us_stock",
    "source": "yahoo",
    "fetchedAt": "2026-04-12T06:32:11Z",
    "current": {
      "price": 234.56,
      "date": "2026-04-11T00:00:00Z",
      "changeAbs": 1.23,
      "changePct": 0.527
    },
    "history": [
      {
        "date": "2026-03-13T00:00:00Z",
        "open": 230.0,
        "high": 235.0,
        "low": 229.5,
        "close": 233.33,
        "volume": 8234100
      }
    ]
  }
}
```

### 不可用响应（200）

```json
{
  "data": {
    "assetCode": "518880",
    "assetType": "gold_etf",
    "source": "unavailable",
    "fetchedAt": "2026-04-12T06:32:11Z",
    "current": null,
    "history": []
  }
}
```

source 枚举值：`yahoo` | `stooq` | `akshare` | `unavailable` | `rate_limited`

## 8 位置与布局

面板位于 DecisionCardDetailPage 主内容区的 Space vertical stack 中：

```
CardHero
MarketContextPanel   <-- 新增
ConclusionBanner
ExecutionPlanFull
DimensionReasoning
MainRisks
```

面板跟随主内容区宽度（lg:18 列），响应式缩放。

## 9 可视化库策略

- **lightweight-charts**（~45KB）：金融时序图表，本次 feature 直接使用
- **ECharts + echarts-for-react**：统计图表，本次安装但不使用，为后续投资组合分析预留
- Vite 配置 manualChunks 将两个库分到独立 chunk，决策卡页面不加载 ECharts

## 10 国际化

namespace: `app`，键组: `decisionCard.marketContext.*`

需同步更新的文件：
- `src/i18n/locales/zh/app.json`
- `src/i18n/locales/en/app.json`

## 11 设计审查产物

### 11.1 状态空间表

| mode | hasHistory | action | 分类 | 行为 |
|---|---|---|---|---|
| unavailable | - | idle | Valid | 折叠态 |
| live | true | idle | Valid | 主 happy path |
| live | false | idle | Forbidden | 后端 supported 资产不应返回空 history |
| live | true | refreshing | Transient | 刷新中保留旧数据 |
| live | true | errored | Valid | Error toast + 保留旧数据 |
| live | false | errored | Valid | 首次加载失败 |
| unavailable | - | refreshing | Valid | 重试查询 |

### 11.2 Pre-mortem Top 5

1. 4 条价位线叠加不可读 → 每条线用 label + color 区分
2. 手动刷新不生效 → invalidateQueries 绕过 staleTime
3. A 股 unavailable 态误导为实时 → 明确标注"分析时刻价格"
4. StrictMode 双调用 canvas 叠加 → cleanup 执行 chart.remove()
5. 30 天 history 不足 → 后端拉 45 天缓冲

### 11.3 替代路径验证

| 路径 | 处理 |
|---|---|
| Back 导航 | TanStack Query 缓存 120s 内有效 |
| Retry after failure | invalidateQueries 无副作用 |
| Cross-session | in-memory cache 重置，重新 loading |
| Concurrent mutation | reanalyze 与 quote 不同 query key |
| Unavailable → 上线 | 手动刷新重试 |
| 快速连点 | TanStack Query 去重 |
| 分析时刻在图外 | 条件渲染垂直标记 |
