# 决策卡实盘行情面板 Plan

PRD: docs/prds/market-quote-panel-prd.md
TRD: docs/trds/market-quote-panel-trd.md

## Step 1: 后端 quote service + handler

### 任务目标
创建 QuoteProvider 接口、FetcherAdapter、内存缓存、QuoteService、HTTP handler，在 main.go 注册路由。

### 涉及文件
- 创建 `backend/internal/service/quote/provider.go`
- 创建 `backend/internal/service/quote/fetcher_adapter.go`
- 创建 `backend/internal/service/quote/memory_cache.go`
- 创建 `backend/internal/service/quote/dto.go`
- 创建 `backend/internal/service/quote/service.go`
- 创建 `backend/internal/api/v1/asset_quotes.go`
- 修改 `backend/cmd/server/main.go`

### 设计依据
- TRD 1.1-1.7
- PRD 6, 7

### 验证标准
- `make check` 通过（lint + test + build）
- 手动 curl `/api/v1/assets/us_stock/GLD/quote` 返回 200 + 正确 JSON（需后端运行时验证，编码阶段以 build 通过为基本验证）
- curl `/api/v1/assets/invalid_type/X/quote` 返回 400

### 依赖
无前置依赖，可与 Step 2 并行执行

## Step 2: 前端依赖安装 + features/market-quote 模块

### 任务目标
安装 lightweight-charts + echarts + echarts-for-react，创建 features/market-quote 模块（types, api, hook, MarketQuoteChart 组件, barrel），配置 Vite chunk 分割。

### 涉及文件
- 修改 `frontend/package.json`（pnpm add）
- 修改 `frontend/vite.config.ts`（manualChunks）
- 创建 `frontend/src/features/market-quote/types.ts`
- 创建 `frontend/src/features/market-quote/api.ts`
- 创建 `frontend/src/features/market-quote/use-asset-quote.ts`
- 创建 `frontend/src/features/market-quote/components/MarketQuoteChart.tsx`
- 创建 `frontend/src/features/market-quote/index.ts`

### 设计依据
- TRD 2.1-2.4
- PRD 9

### 验证标准
- `pnpm lint:all` 通过
- barrel export 正确（import { useAssetQuote, MarketQuoteChart } from "@/features/market-quote" 不报 TS 错误）
- lightweight-charts 在 bundle 中独立 chunk

### 依赖
无前置依赖，可与 Step 1 并行执行

## Step 3: 页面集成 + i18n + 最终验证

### 任务目标
创建 MarketContextPanel 页面组件，集成到 DecisionCardDetailPage，添加 i18n 双语键，执行 lint 全量检查。

### 涉及文件
- 创建 `frontend/src/pages/decision-cards/components/MarketContextPanel.tsx`
- 修改 `frontend/src/pages/decision-cards/DecisionCardDetailPage.tsx`
- 修改 `frontend/src/i18n/locales/zh/app.json`
- 修改 `frontend/src/i18n/locales/en/app.json`

### 设计依据
- TRD 2.5-2.6, 2.9
- PRD 4, 8, 10

### 验证标准
- `pnpm lint:all` 通过
- `pnpm build` 通过
- i18n 中英文键对称（相同的键结构）
- MarketContextPanel 在 CardHero 和 ConclusionBanner 之间

### 依赖
依赖 Step 2（需要 features/market-quote 模块和 lightweight-charts 依赖）

## 并行策略

```
Step 1 (backend)  ────────────────────┐
                                      ├──> Step 3 (integration)
Step 2 (frontend module) ────────────┘
```

Step 1 和 Step 2 无共享文件，可并行执行。
Step 3 依赖 Step 2 的 feature 模块输出，必须等 Step 2 完成。
Step 3 不直接依赖 Step 1（前端可以先写完组件，后端接口运行时验证）。
