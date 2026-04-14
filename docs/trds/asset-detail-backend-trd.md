# Asset Detail Backend TRD

## 背景与目标

### 现状

`GET /api/v2/market/:code` 当前仅返回 11 个字段（code/name/nameEn/assetType/exchange/overallScore/signalLevel/scoreDelta/percentileLabel/dimensions/analyzedAt/analysisId），但前端 `AssetDetailDto`（`frontend/src/features/asset-detail/types.ts`）声明了 25+ 字段，包括 `currency` / `currentPrice` / `priceChangePercent` / `marketInterpretation` / `riskFactors[]` / `keyPriceLevels[]` / `drawdownReference` / `executionPlan` / `supports[]` / `resistances[]` / `sma200` / `majorChangeRecap` / `conflictType` / `conflictMessage` / `validDays` / `scoreBandLow/High` 等。

执行报告 Round 3-4 把这块标记为「契约缺口」并要求独立 TRD（详见 `docs/reports/richman-v2-plan-execution-report.md`「asset-detail 详情页大面积契约缺口」条目）。

### 目标

把 `AssetDetailDTO` 扩展为前端 `AssetDetailDto` 的 superset，所有前端期望字段都从已落地的数据源中聚合，避免前端兜底逻辑膨胀。

### 非目标

- 不引入新的 richson 接口；只消费现有 `/market/ohlcv/:code` 与 `rs_asset_analyses` JSONB 列。
- 不实现 per-dimension `summary` / `llmReason`（richson 当前不写入），保留为 null/空字符串，由 richson 后续 enhancement 单独立项。
- 不引入 OHLCV 持久缓存层；只在 service 内复用现有 1 小时 percentile 缓存的同款轻量 in-memory cache 思路。

## 数据源映射

下表把前端 `AssetDetailDto` 每个字段映射到后端来源。「源」列写明 SQL 列、JSONB 路径或 richson API。

| 前端字段 | 类型 | 源 | 备注 |
|----------|------|----|------|
| code, name, nameEn, assetType, exchange | string | `rm_assets` | 已落地 |
| currency | "USD"\|"CNY" | richson `GET /market/ohlcv/:code` 响应 `currency` | OHLCV 失败时按 `assetType` 回退（stock-cn→CNY，其他→USD） |
| usdExchangeRate | number\|null | `rs_asset_analyses.usd_exchange_rate` | 已存在列；CNY 资产为 NULL |
| currentPrice | number | OHLCV 最新 candle 的 `close` | OHLCV 失败时为 undefined |
| priceChangePercent | number | OHLCV 最近两根 candle close 计算 `(latest - prev) / prev * 100` | OHLCV 不足两根则 undefined |
| priceAtAnalysis | number | `rs_asset_analyses.price_at_analysis` | 已存在列 |
| overallScore | number | `rs_asset_analyses.overall_score` | 已落地 |
| scoreBandLow / scoreBandHigh | number | `rs_asset_analyses.confidence_band_low/high` | 已存在列，仅前端字段名映射 |
| signalLevel | string | `rs_asset_analyses.signal_level` | 已落地，richson 输出 `strong_bullish`/`moderate_bullish`/`neutral`/`moderate_bearish`/`strong_bearish` |
| percentileLabel | string\|null | service 计算（已实现） | 已落地 |
| marketInterpretation | string | `rs_asset_analyses.market_interpretation` | 已存在列 |
| scoreDelta | number\|null | `rs_asset_analyses.score_delta` | 已落地 |
| changeSummary | string\|null | `rs_asset_analyses.change_summary` | 已存在列 |
| majorChangeRecap | object\|null | 由 `major_change_recap`(text) + `score_delta` + `prev_analysis_id` 反查的 prev `overall_score` 拼装 | 详见「派生字段拼装」 |
| conflictType, conflictMessage | string\|null | `rs_asset_analyses.conflict_type/conflict_message` | 已存在列 |
| analyzedAt | ISO string | `rs_asset_analyses.analyzed_at` | 已落地 |
| validDays | number\|undefined | `rs_asset_analyses.demo_plan.valid_days` | JSONB 解码 |
| dimensions[] | DimensionDetailDto[] | `rs_asset_analyses` 的 dN_score/dN_base_score/dN_llm_adjustment/dN_weight + `rs_asset_analysis_dimensions` 行 | summary/llmReason 暂为空，subIndicators 由 dimension 行映射 |
| riskFactors[] | RiskFactorDto[] | `rs_asset_analyses.risk_factors`(list[str]) | 包装为 `{id: "rf-{idx}", description, severity: "medium"}`（severity 暂全部 medium，等 richson 升级 schema） |
| keyPriceLevels[] | KeyPriceLevelDto[] | `rs_asset_analyses.analysis_metadata.support_levels` + `resistance_levels`（OHLCV 不可用时回退） | 详见「派生字段拼装」 |
| drawdownReference | DrawdownReferenceDto\|null | `rs_asset_analyses.analysis_metadata.drawdown_reference` | snake_case → camelCase 转换 |
| executionPlan | ExecutionPlanDto\|null | `rs_asset_analyses.demo_plan` | JSONB 解码 + 字段重命名 |
| supports[] | number[] | OHLCV 响应 `supportLevels`（首选）或 `analysis_metadata.support_levels`（回退） | OHLCV 用 60 秒缓存 |
| resistances[] | number[] | 同上 `resistanceLevels` | 同上 |
| sma200 | number\|null | OHLCV 响应 `sma200` | OHLCV 失败为 null |

未在前端 `AssetDetailDto` 中、但已经返回的字段（`analysisId`）保留向后兼容。

### 派生字段拼装

#### majorChangeRecap

- `rs_asset_analyses.major_change_recap` 是纯文本；前端期望 `{text, scoreDelta, previousScore, currentScore}`
- 拼装规则：
  - `text`: 取自 `major_change_recap`，为空则整个对象返回 null
  - `scoreDelta`: `rs_asset_analyses.score_delta`（已存在）
  - `currentScore`: `rs_asset_analyses.overall_score`
  - `previousScore`: 用 `prev_analysis_id` 反查 `rs_asset_analyses`，取 `overall_score`；查不到则 `currentScore - scoreDelta`
- prev_analysis_id 反查复用现有 `AssetAnalysisReadRepo.GetSecondLatestByAssetCode` 模式（新增 `GetByID` 方法即可）

#### keyPriceLevels

- 前端期望 `[{type: "support"|"resistance", price, distancePct, currency}]`
- 数据来源优先级：OHLCV.supportLevels/resistanceLevels（更新更频繁），无则用 `analysis_metadata.support_levels/resistance_levels`
- `distancePct`: 相对 `currentPrice` 的百分比 `(price - currentPrice) / currentPrice * 100`；无 currentPrice 时为 0
- `currency`: 沿用 DTO 的 `currency` 字段
- 排序：支撑按 distancePct 升序（最近的在前），阻力同理

#### dimensions[]

richson 写入两层数据：
- 顶层（`rs_asset_analyses.dN_score/dN_base_score/dN_llm_adjustment/dN_weight`）：每个 dimension 的总分、底分、LLM 调整、权重
- sub_indicator 层（`rs_asset_analysis_dimensions` 多行）：每个量化指标的 raw_value/percentile/normalized_score/weight_in_dimension

后端聚合规则：
- 4 个 dimension（d1..d4）固定输出 4 项；任何缺失（d4 无 llm_adjustment 列）则该字段为 null
- `name`: 取 i18n key `dimension.d1`..`dimension.d4` 由前端翻译；后端固定输出 `"d1"`..`"d4"` 作为 id；`name` 字段填英文短名（`Macro` / `Liquidity` / `Sentiment` / `Technical`）作为 fallback
- `signal`: 用 `signal_level_from_score(score)` 同款分桶逻辑（≥75 bullish / 60-75 bullish / 40-60 neutral / 25-40 bearish / <25 bearish）
- `summary`, `llmReason`: 暂为 null（richson 未写入；后续 enhancement）
- `subIndicators[]`: 用 `rs_asset_analysis_dimensions` 行映射，按 dimension 分组：
  - `name`: `sub_indicator`
  - `rawValue`: `raw_value`（数据库列是 *float64，转 number；无值时给 0）
  - `percentile`: `blended_percentile` 优先，否则 `percentile_1y`
  - `normalizedScore`: `normalized_score`
  - `weight`: `weight_in_dimension`

#### executionPlan

richson `demo_plan` JSONB 与前端 `ExecutionPlanDto` 的字段映射：

| richson key | frontend key | 说明 |
|-------------|--------------|------|
| action_label | recommendation | 已是人类可读的本地化字符串 |
| default_action | defaultAdvice | 无场景命中时的兜底文案 |
| stop_loss | stopLoss | 直接透传 |
| take_profit | takeProfit | 直接透传 |
| valid_days | validDays | 直接透传 |
| concentration_message | concentrationWarning | 集中度警告文案，concentration_level 不暴露给前端 |
| scenarios[] | scenarios[] | 子结构详见下表 |
| (无) | disclaimer | 后端注入固定文案（i18n key 由前端翻译，后端发英文 fallback） |

scenarios 子映射：
| richson | frontend |
|---------|----------|
| condition | condition |
| action | action |
| rationale | rationale |
| priority | priority |
| (合成) | id = `scenario-{idx}` |

`is_demo_plan` / `current_position` / `target_position` / `lot_count` / `exclusion_group` / `no_trigger_note` 字段不暴露到前端 DTO（前端不消费）。

## 接口设计

### Service 层接口

`internal/service/market/service.go` 中 `GetAssetDetail(ctx, code)` 方法签名不变，返回的 `*AssetDetailDTO` 结构体扩展。

新增依赖：
- `richsonClient *richson.Client`（用于 OHLCV 拉取）

构造函数 `NewService` 增加 richsonClient 参数；`main.go` 注入。

### 内部辅助方法

```go
// 内部辅助方法（service 包私有）
func (s *Service) buildExecutionPlan(raw json.RawMessage, locale string) *ExecutionPlanDTO
func (s *Service) buildDimensions(analysis *model.AssetAnalysis, dims []model.AnalysisDimension) []DimensionDTO
func (s *Service) buildKeyPriceLevels(supports, resistances []float64, currentPrice *float64, currency string) []KeyPriceLevelDTO
func (s *Service) buildRiskFactors(raw json.RawMessage) []RiskFactorDTO
func (s *Service) buildMajorChangeRecap(ctx context.Context, analysis *model.AssetAnalysis) *MajorChangeRecapDTO
func (s *Service) fetchOHLCVForDetail(ctx context.Context, code string) *ohlcvSnapshot // nil on failure, logged warning
func (s *Service) deriveDimensionSignal(score *float64) string // bullish/neutral/bearish
```

OHLCV 失败时返回 nil，service 层降级（currentPrice 等字段为 undefined），不阻塞主流程；依据 richman-backend-v2-trd SS3「richson 不可用降级」原则。

### DTO 类型新增

`internal/service/market/service.go` 新增以下 struct，与前端 TS interface 一一对应（json tag 用 camelCase）：

- `ExecutionPlanDTO`, `ExecutionScenarioDTO`
- `RiskFactorDTO`
- `KeyPriceLevelDTO`
- `DrawdownReferenceDTO`
- `MajorChangeRecapDTO`
- `DimensionDTO`, `DimensionSubIndicatorDTO`
- `AssetDetailDTO` 扩字段

所有 optional 字段用指针类型 + `omitempty`，遵循 `docs/standards/contract-drift.md` 的 `T | None ↔ *T` 规则。

### Repo 层新增

`AssetAnalysisReadRepo` 新增方法：
- `GetByID(ctx, id) (*model.AssetAnalysis, error)`：用于反查 prev analysis 的 overall_score（majorChangeRecap 拼装）

`AnalysisDimensionReadRepo` 无需变更。

### JSONB 反序列化

新增内部包 `internal/service/market/jsonb.go`：

```go
// 与 richson schemas 字段名一致的内部类型
type rawDemoPlan struct {
    Action               string                  `json:"action"`
    ActionLabel          string                  `json:"action_label"`
    DefaultAction        string                  `json:"default_action"`
    Scenarios            []rawDemoPlanScenario   `json:"scenarios"`
    StopLoss             *float64                `json:"stop_loss"`
    TakeProfit           *float64                `json:"take_profit"`
    ValidDays            *int                    `json:"valid_days"`
    ConcentrationMessage *string                 `json:"concentration_message"`
}

type rawAnalysisMetadata struct {
    DrawdownReference *rawDrawdownReference `json:"drawdown_reference"`
    SupportLevels     []float64             `json:"support_levels"`
    ResistanceLevels  []float64             `json:"resistance_levels"`
}

type rawDrawdownReference struct {
    CurrentBullRunStart    *string  `json:"currentBullRunStart"`
    MaxDrawdown            *float64 `json:"maxDrawdown"`
    MaxDrawdownDate        *string  `json:"maxDrawdownDate"`
    HistoricalAvgDrawdown  *float64 `json:"historicalAvgDrawdown"`
}
```

注：drawdown_reference 内部已是 camelCase（richson `drawdown.py` 输出），其余两块是 snake_case。

`unmarshalDemoPlan(raw json.RawMessage) *rawDemoPlan` / `unmarshalAnalysisMetadata(...)` 在 raw=nil 或解码失败时返回 nil，service 层把 fallback 处理放在 build* 辅助方法里。

## 缓存策略

- `getPercentileLabel` 已有 1 小时 in-memory 缓存（保留）
- 新增 `ohlcvCache map[string]ohlcvCacheEntry`，TTL 60 秒：
  - 同样用 `sync.Mutex`
  - 缓存项含 `currentPrice / priceChangePercent / sma200 / supports / resistances / currency`，方便复用
  - 失败响应不缓存（避免短暂 richson 抖动锁住一分钟）
- 不引入 redis；与现有 percentile 缓存采用同一进程内 map，service 重启后冷启动可接受

理由：asset detail 是公开页，预期 QPS 低；进程内缓存足以；引入 redis 会增加 SS（运维）复杂度，与 MVP 目标相悖。

## 错误处理

| 失败场景 | 行为 |
|----------|------|
| `rm_assets` 不存在 | 返回 404（既有逻辑） |
| `rs_asset_analyses` 无记录 | 返回基础信息（既有逻辑），analysis 字段全部 omit |
| OHLCV 调用失败 | currentPrice/priceChangePercent/sma200 omit，supports/resistances 回退到 analysis_metadata；warn log，不报 5xx |
| JSONB 解码失败 | 对应字段 nil/omit；error log 含 asset_code 与列名 |
| prev analysis 反查失败 | majorChangeRecap.previousScore 用 `currentScore - scoreDelta` 兜底 |

## 测试策略

按项目 `testing.md` 标准：
- 后端 `service/market/service_test.go` 新增表驱动单测：
  - `Test_buildExecutionPlan_DemoPlanComplete`（完整 JSONB 解码）
  - `Test_buildExecutionPlan_NilOrInvalid`（兜底 nil）
  - `Test_buildDimensions_Aggregation`（dimension 顶层 + sub_indicator 行聚合）
  - `Test_buildKeyPriceLevels_DistancePctAndOrder`（排序 + distance 计算）
  - `Test_buildMajorChangeRecap_PrevLookup_Fallback`（prev 查不到时用 `current - delta`）
  - `Test_deriveDimensionSignal`（5 个分桶边界）
- repo 层 `GetByID` 写一个 happy-path + nil 测试
- 前端 MVP 阶段不写 .test.tsx（按 `frontend.md`）

## 兼容性与回滚

- DTO 仅做加字段，所有新字段都是 optional + omitempty；前端旧版本无影响
- 出现严重问题可通过环境开关回滚：保留旧版 `GetAssetDetail` 路径不可行（重构成本高），改用单独 feature flag `ASSET_DETAIL_RICH_PAYLOAD=false` 时跳过新字段填充。MVP 阶段不实现此 flag，缺陷直接修复；若上线后发现重大 regression 才补 flag

## 实施依赖

- 不依赖 richson 改动
- 不依赖 DB schema 改动
- 不依赖前端改动（前端 `AssetDetailDto` 已声明所有字段，今天填的是兜底；后端补齐后只是字段从 undefined 变成实数据）

## 风险与对策

| 风险 | 影响 | 对策 |
|------|------|------|
| richson `demo_plan` 实际 schema 与本 TRD 描述漂移 | 字段 nil/前端展示空 | 在 unmarshal 失败时 warn log；上线后用真实数据回归 |
| OHLCV 60s 缓存导致股票变动延迟 | 用户看到的价格滞后最多 1 分钟 | 可接受（asset detail 是研究页非交易页） |
| 反查 prev_analysis_id 增加单次响应 SQL 数（+1） | 单页 P95 上升 ~5ms | 可接受；若日后压力上升再批量 join |
| `risk_factors` 全部 severity=medium 失真 | UI 红黄绿三色都是黄色 | 短期可接受，记入 richson enhancement backlog |
| 前端 `signalLevel` 注释写的是 `bullish` 而 richson 实际写 `moderate_bullish` | i18n key 找不到 | 已在前端 score-summary 与 sticky-header 处通过 `t(key, fallback)` 兜底；本 TRD 不解决 i18n key 升级，留独立任务 |

## 引用

- richman-backend-v2-trd SS3（richson 降级原则）、SS4.1（v2 路由清单）
- richson-service-trd SS5（execution_agent demo_plan 字段）、SS7（drawdown_reference 计算）
- frontend-v2-trd SS17（AssetDetail 三 Tab 渲染）
- docs/standards/contract-drift.md（DTO 三端对齐）
- docs/standards/abstraction-reuse.md（service 层职责）
- docs/reports/richman-v2-plan-execution-report.md「asset-detail 详情页大面积契约缺口」
