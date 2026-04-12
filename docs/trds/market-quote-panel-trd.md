# 决策卡实盘行情面板 TRD

## 1 后端架构

### 1.1 QuoteProvider 接口

```go
// internal/service/quote/provider.go
package quote

type QuoteProvider interface {
    FetchQuote(ctx context.Context, req QuoteRequest) (*QuoteSnapshot, error)
}

type QuoteRequest struct {
    AssetType string
    AssetCode string
    Days      int  // 默认 45
}

type QuoteSnapshot struct {
    Current   PricePoint
    Previous  PricePoint
    History   []datasource.PriceData
    Source    string
    FetchedAt time.Time
}

type PricePoint struct {
    Date  time.Time
    Close float64
}
```

### 1.2 FetcherAdapter

```go
// internal/service/quote/fetcher_adapter.go
// 包装 datasource.Fetcher，复用现有路由（yahoo/akshare/stooq fallback）
type FetcherAdapter struct {
    fetcher *datasource.Fetcher
}

func (a *FetcherAdapter) FetchQuote(ctx context.Context, req QuoteRequest) (*QuoteSnapshot, error) {
    days := req.Days
    if days <= 0 {
        days = 45
    }
    // 复用 FetchAssetData 但只取 Prices
    data, err := a.fetcher.FetchAssetData(ctx, req.AssetCode, req.AssetType)
    if err != nil {
        return nil, err
    }
    prices := data.Prices
    if len(prices) == 0 {
        return nil, fmt.Errorf("no price data")
    }
    snap := &QuoteSnapshot{
        Current:   PricePoint{Date: prices[len(prices)-1].Date, Close: prices[len(prices)-1].Close},
        History:   prices,
        Source:    resolveSourceName(req.AssetType, req.AssetCode),
        FetchedAt: time.Now(),
    }
    if len(prices) >= 2 {
        snap.Previous = PricePoint{Date: prices[len(prices)-2].Date, Close: prices[len(prices)-2].Close}
    }
    return snap, nil
}
```

注意：`FetchAssetData` 内部的 `defaultFetchDays=90`。这意味着无论我们请求多少天，它始终拉 90 天。FetcherAdapter 在返回前裁剪到请求的 days 数。这不是效率问题（90 天日线很小），但需要注意返回数据量。

`resolveSourceName` 根据 assetType + code 判断实际走了哪个源（yahoo/akshare），与 fetcher.go 的路由逻辑对齐。

### 1.3 内存缓存

```go
// internal/service/quote/memory_cache.go
type cacheEntry struct {
    dto       *QuoteDTO
    expiresAt time.Time
}

type memoryCache struct {
    mu    sync.RWMutex
    store map[string]cacheEntry
}

func (c *memoryCache) Get(key string) (*QuoteDTO, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    entry, ok := c.store[key]
    if !ok || time.Now().After(entry.expiresAt) {
        return nil, false
    }
    return entry.dto, true
}

func (c *memoryCache) Set(key string, dto *QuoteDTO, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.store[key] = cacheEntry{dto: dto, expiresAt: time.Now().Add(ttl)}
}
```

Cache key: `assetType:assetCode`。TTL: 120s。

### 1.4 QuoteService

```go
// internal/service/quote/service.go
type Service struct {
    provider QuoteProvider
    cache    *memoryCache
    logger   *zap.Logger
    cacheTTL time.Duration
}

func NewService(provider QuoteProvider, logger *zap.Logger) *Service {
    return &Service{
        provider: provider,
        cache:    newMemoryCache(),
        logger:   logger,
        cacheTTL: 120 * time.Second,
    }
}

func (s *Service) GetQuote(ctx context.Context, assetType, assetCode string) (*QuoteDTO, error) {
    key := assetType + ":" + assetCode
    if cached, ok := s.cache.Get(key); ok {
        return cached, nil
    }
    snap, err := s.provider.FetchQuote(ctx, QuoteRequest{
        AssetType: assetType,
        AssetCode: assetCode,
        Days:      45,
    })
    if errors.Is(err, datasource.ErrUnsupportedAssetType) {
        dto := &QuoteDTO{
            AssetCode: assetCode,
            AssetType: assetType,
            Source:    "unavailable",
            FetchedAt: time.Now().UTC(),
        }
        s.cache.Set(key, dto, s.cacheTTL)
        return dto, nil
    }
    if err != nil {
        return nil, err
    }
    dto := s.toDTO(snap, assetCode, assetType)
    s.cache.Set(key, dto, s.cacheTTL)
    return dto, nil
}
```

### 1.5 QuoteDTO（HTTP 响应结构）

```go
// internal/service/quote/dto.go
type QuoteDTO struct {
    AssetCode string           `json:"assetCode"`
    AssetType string           `json:"assetType"`
    Source    string           `json:"source"`
    FetchedAt time.Time        `json:"fetchedAt"`
    Current   *CurrentQuote    `json:"current"`
    History   []HistoryPoint   `json:"history"`
}

type CurrentQuote struct {
    Price     float64   `json:"price"`
    Date      time.Time `json:"date"`
    ChangeAbs float64   `json:"changeAbs"`
    ChangePct float64   `json:"changePct"`
}

type HistoryPoint struct {
    Date   time.Time `json:"date"`
    Open   float64   `json:"open"`
    High   float64   `json:"high"`
    Low    float64   `json:"low"`
    Close  float64   `json:"close"`
    Volume float64   `json:"volume"`
}
```

### 1.6 Handler

```go
// internal/api/v1/asset_quotes.go
type AssetQuoteHandler struct {
    service *quote.Service
}

func NewAssetQuoteHandler(service *quote.Service) *AssetQuoteHandler {
    return &AssetQuoteHandler{service: service}
}

var validAssetTypes = map[string]bool{
    "us_stock":         true,
    "gold_etf":         true,
    "a_share_broad":    true,
    "a_share_industry": true,
}

func (h *AssetQuoteHandler) RegisterRoutes(g *gin.RouterGroup, authMW gin.HandlerFunc) {
    g.GET("/quotes/:assetType/:assetCode", authMW, h.getQuote)
}

func (h *AssetQuoteHandler) getQuote(c *gin.Context) {
    assetType := c.Param("assetType")
    assetCode := c.Param("assetCode")
    if !validAssetTypes[assetType] {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": gin.H{
                "code":    "INVALID_ASSET_TYPE",
                "message": fmt.Sprintf("unsupported asset type: %s", assetType),
            },
        })
        return
    }
    if assetCode == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": gin.H{
                "code":    "MISSING_ASSET_CODE",
                "message": "asset code is required",
            },
        })
        return
    }
    dto, err := h.service.GetQuote(c.Request.Context(), assetType, assetCode)
    if err != nil {
        c.Error(err)
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": dto})
}
```

### 1.7 main.go 注册

在现有 handler 注册序列末尾追加：

```go
quoteSvc := quote.NewService(
    quote.NewFetcherAdapter(fetcher),
    zapLogger,
)
quoteHandler := v1.NewAssetQuoteHandler(quoteSvc)
quoteHandler.RegisterRoutes(apiV1, authMiddleware)
```

## 2 前端架构

### 2.1 features/market-quote 模块结构

```
features/market-quote/
    api.ts                     # fetchAssetQuote(assetType, assetCode)
    types.ts                   # AssetQuoteDTO, PriceLine, TimeMarker
    use-asset-quote.ts         # useAssetQuote hook, queryKey
    components/
        MarketQuoteChart.tsx   # lightweight-charts 封装
    index.ts                   # barrel: export useAssetQuote, MarketQuoteChart, types
```

### 2.2 TypeScript 接口

```typescript
// types.ts
export interface QuoteCurrentDTO {
  price: number;
  date: string;
  changeAbs: number;
  changePct: number;
}

export interface QuoteHistoryPoint {
  date: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface AssetQuoteDTO {
  assetCode: string;
  assetType: string;
  source: string;
  fetchedAt: string;
  current: QuoteCurrentDTO | null;
  history: QuoteHistoryPoint[];
}

export interface PriceLine {
  price: number;
  color: string;
  lineStyle: "solid" | "dashed";
  label: string;
}

export interface TimeMarker {
  time: string;
  label: string;
  color: string;
}
```

### 2.3 Hook

```typescript
// use-asset-quote.ts
export function assetQuoteQueryKey(assetType: string, assetCode: string) {
  return ["asset-quote", assetType, assetCode] as const;
}

export function useAssetQuote(assetType: string, assetCode: string) {
  return useQuery<AssetQuoteDTO>({
    queryKey: assetQuoteQueryKey(assetType, assetCode),
    queryFn: () => fetchAssetQuote(assetType, assetCode).then(r => r.data),
    enabled: !!assetType && !!assetCode,
    staleTime: 120_000,
  });
}
```

### 2.4 MarketQuoteChart 组件

```typescript
// components/MarketQuoteChart.tsx
interface MarketQuoteChartProps {
  history: QuoteHistoryPoint[];
  priceLines: PriceLine[];
  timeMarkers: TimeMarker[];
  height?: number;  // 默认 160
}
```

实现约束：
- `useRef<HTMLDivElement>` 持容器
- `useEffect` 创建 chart：`createChart(ref, {width, height, layout, grid, timeScale})`
- `addLineSeries()` 渲染收盘价折线
- 遍历 `priceLines` 调用 `series.createPriceLine({price, color, lineWidth, lineStyle, axisLabelVisible, title})`
- 遍历 `timeMarkers` 调用 `series.setMarkers([{time, position, color, shape, text}])`
- cleanup: `chart.remove()`
- 独立 useEffect 响应 priceLines/timeMarkers 变化：清除旧 priceLine 实例 + 重建
- ResizeObserver 监听容器宽度变化 → `chart.applyOptions({width})`

### 2.5 MarketContextPanel 页面组件

```typescript
// pages/decision-cards/components/MarketContextPanel.tsx
interface MarketContextPanelProps {
  card: DecisionCardDTO;
}
```

职责：
1. 调用 `useAssetQuote(card.assetType, card.assetCode)`
2. 从 card DTO 提取 overlays：
   - `card.costPrice` → PriceLine (gray, solid, "Cost")
   - `card.recommendation.execution.stopLoss` → PriceLine (red, dashed, "SL") [null 时跳过]
   - 第一个 triggerType="price" 的 step 的 triggerPayload.priceValue → PriceLine (orange, dashed, "Trigger") [无 price 触发时跳过]
   - `card.analyzedAt` → TimeMarker (blue, "Analysis")
3. 分支渲染：
   - `isLoading` → Skeleton
   - `source === "unavailable"` → 折叠态
   - `isError` → Error + retry
   - else → 正常态
4. 刷新按钮 onClick: `queryClient.invalidateQueries({queryKey: assetQuoteQueryKey(...)})`

### 2.6 DecisionCardDetailPage 集成

在 Space children 中 CardHero 后面插入一行：

```tsx
<CardHero card={card} />
<MarketContextPanel card={card} />
<ConclusionBanner card={card} prevCard={prevCard} />
```

新增 import: `import { MarketContextPanel } from "./components/MarketContextPanel";`

### 2.7 依赖安装

```bash
pnpm add lightweight-charts echarts echarts-for-react
```

### 2.8 Vite chunk 配置

在 vite.config.ts 的 build.rollupOptions.output.manualChunks 中追加：

```typescript
manualChunks: {
  'chart-lightweight': ['lightweight-charts'],
  'chart-echarts': ['echarts', 'echarts-for-react'],
}
```

### 2.9 i18n 键

zh/app.json:
```json
"decisionCard": {
  "marketContext": {
    "title": "实盘行情",
    "currentPrice": "现价",
    "todayChange": "今日",
    "vsAnalysis": "较分析",
    "vsCost": "较成本",
    "updatedAt": "更新于 {{time}}",
    "refresh": "刷新行情",
    "refreshing": "刷新中",
    "unavailable": {
      "title": "{{assetType}} 实盘行情数据源待接入",
      "analysisPrice": "分析时刻价格 (非实时)",
      "analysisTime": "分析于 {{time}}"
    },
    "error": {
      "title": "行情数据加载失败",
      "retry": "重试"
    },
    "overlay": {
      "cost": "成本",
      "stopLoss": "止损",
      "trigger": "触发",
      "analysis": "分析"
    },
    "outsideRange": "分析时刻早于此区间"
  }
}
```

en/app.json:
```json
"decisionCard": {
  "marketContext": {
    "title": "Market Quote",
    "currentPrice": "Price",
    "todayChange": "Today",
    "vsAnalysis": "vs Analysis",
    "vsCost": "vs Cost",
    "updatedAt": "Updated at {{time}}",
    "refresh": "Refresh",
    "refreshing": "Refreshing",
    "unavailable": {
      "title": "{{assetType}} market data source pending",
      "analysisPrice": "Price at analysis time (not live)",
      "analysisTime": "Analyzed at {{time}}"
    },
    "error": {
      "title": "Failed to load market data",
      "retry": "Retry"
    },
    "overlay": {
      "cost": "Cost",
      "stopLoss": "Stop Loss",
      "trigger": "Trigger",
      "analysis": "Analysis"
    },
    "outsideRange": "Analysis time is outside chart range"
  }
}
```

## 3 文件契约影响表

| 文件 | 现有契约 | 改动 |
|---|---|---|
| DecisionCardDetailPage.tsx | 5 block stack + sidebar | 插入 MarketContextPanel，不改逻辑 |
| features/decision-card/types.ts | DTO 镜像后端 | 不修改 |
| backend/cmd/server/main.go | handler 注册链 | 末尾追加 quoteHandler |
| backend/internal/datasource/fetcher.go | FetchAssetData 路由 | 不修改，通过 FetcherAdapter 调用 |
| backend/internal/datasource/types.go | PriceData 等类型 | 不修改 |
| frontend/package.json | 依赖列表 | 追加 3 个 deps |
| frontend/vite.config.ts | build 配置 | 追加 manualChunks |
| i18n zh/en app.json | 双文件同步 | 追加 marketContext 键组 |
