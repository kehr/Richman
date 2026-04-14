# 事件雷达数据源真实化 PRD

## 1. 背景与问题

事件雷达（Event Radar）是行情概览页（market-overview）和资产详情页（asset-detail）共用的一块"未来 7 天关键宏观事件"展示区。当前实现存在三个根本问题：

1. **数据是假的**：richson `api/events.py` 使用 `_FIXED_EVENTS` 五条硬编码列表，日期是 `today + day_offset`（3/5/7/10/14 天）拼出的"假未来"，每天打开页面看到的"事件日期"都在跟着今天滚动；CPI、PPI、非农等本应有官方公布日历的指标并不真的对应真实的发布日。
2. **没有权威来源**：每条事件不带任何 source_url / source_name，用户无法点击查看细节、无法验证正确性、无法跳转到原始 release 页面。
3. **资产详情页的事件日历是空 Card**：`pages/asset-detail/event-calendar.tsx` 是占位符，长期未实现，挂在 risk-tab 上呈现"—"。

本次彻底修复：把硬编码日历替换为 FRED Releases Calendar 真实日历 + Polymarket 实时市场，三端 DTO 增加 `sourceUrl/sourceName`，前端整行可点击跳转权威源（新标签），同时把资产详情页 event-calendar 改为复用同一份组件。

## 2. 目标与非目标

### 2.1 目标

- 事件雷达条目的日期来自真实的 FRED Releases Calendar，不再随今天滚动
- 每条事件携带 `sourceUrl`（可点击跳转）和 `sourceName`（如 "FRED" / "Polymarket" / "Federal Reserve"）
- 行情概览页和资产详情页 event-calendar 共用同一个组件、同一个 hook、同一份数据
- FRED key 缺失或失效时，事件雷达优雅降级为 Polymarket-only，不出现网络错误风暴
- 三端 DTO（richson Pydantic / backend Go / frontend TS）字段命名与可空性严格对齐

### 2.2 非目标（YAGNI）

- 不做事件详情 Drawer / 站内详情页（点击只是新标签外链）
- 不做资产→事件关联表（不按 `assetType` 过滤，所有资产共用同一份宏观事件列表）
- 不引入第三方付费经济日历（Trading Economics / Investing API）
- 不暴露时间窗口为查询参数（块定 7 天）
- 不做事件提醒、订阅、推送
- 不做事件历史查询

## 3. 用户故事

- 作为投资者，我打开行情概览页，看到"未来 7 天关键宏观事件"列表，每条事件的日期是 FRED 官方公布日历，让我可以信任它。
- 作为投资者，我看到 "US CPI Data Release 2026-04-19"，鼠标移到行上有 cursor 提示，点击后在新标签页打开 https://fred.stlouisfed.org/release?rid=10 看官方 release 页。
- 作为投资者，我打开任何一只资产的详情页，"近期事件"区域显示与行情概览页一致的宏观事件列表，而不是一个空 Card。
- 作为投资者，FRED key 没配置时（demo 环境），事件雷达仍然显示 Polymarket 市场预测条目，不刷红屏错误。

## 4. 信息架构与字段

### 4.1 三端 DTO 增量（仅新增字段，不破坏既有字段）

| 字段 | richson Pydantic | backend Go | frontend TS | 含义 |
|------|------------------|------------|-------------|------|
| `sourceUrl` | `source_url: str \| None = Field(default=None, alias="sourceUrl")` | `SourceUrl *string \`json:"sourceUrl"\`` | `sourceUrl: string \| null` | 该事件的权威外链 URL，前端用于整行 anchor。可空（极少数事件无外链） |
| `sourceName` | `source_name: str \| None = Field(default=None, alias="sourceName")` | `SourceName *string \`json:"sourceName"\`` | `sourceName: string \| null` | 数据源名称用于副标签展示，例如 `"FRED"` / `"Polymarket"` / `"Federal Reserve"` |
| `releaseId` | `release_id: int \| None = Field(default=None, alias="releaseId")` | `ReleaseId *int \`json:"releaseId"\`` | `releaseId: number \| null` | FRED release id；Polymarket 条目为 null。前端只用作 React key 稳定性，不展示 |

### 4.2 既有字段不变

`date / title / category / impact / goldDirection / probability / probabilitySource / probabilityChange24h` 全部保留，命名和可空性不变。

### 4.3 数据来源映射

| 事件来源 | sourceName 取值 | sourceUrl 模式 |
|---------|----------------|---------------|
| FRED Releases Calendar | `"FRED"` | `https://fred.stlouisfed.org/release?rid={release_id}` |
| Polymarket 市场 | `"Polymarket"` | `https://polymarket.com/event/{slug}` |

注：FRED 单独的 "FOMC Minutes" 不存在，会议事件统一走 release_id=101（FOMC Press Release）；FOMC 计划页另在 federalreserve.gov，不进入本次范围。

## 5. 核心功能与交互

### 5.1 事件雷达条目展示

每条事件展示与现状一致：日期 / 标题 / impact tag / goldDirection tag / probability / 24h change。本次**不改任何视觉布局**，只把整行从 `<div>` 包成可点击的 anchor（带样式：hover 时背景变浅、cursor 改为 pointer），点击在新标签打开 `sourceUrl`。

`sourceName` 不在视觉上单独展示（保持现有视觉密度），但 `title` 属性（HTML title）含 "Source: FRED" 文案以提供 a11y 提示。

### 5.2 sourceUrl 缺失时的回退

`sourceUrl === null` 的条目（未来不会出现，但 DTO 允许）渲染为不可点击的 `<div>`，无 cursor / hover 样式。

### 5.3 资产详情页 event-calendar

`pages/asset-detail/event-calendar.tsx` 改为：
- 调用同一个 `useEventRadar()` hook
- 渲染同一个 `<EventRadarSection/>` 组件
- 套一层 i18n title 用 `assetDetail.risk.events.title`（"近期事件"）覆盖默认的 "事件雷达"

### 5.4 i18n 增量

| key | zh | en |
|-----|----|----|
| `market.overview.eventRadar.openSourceTooltip` | "在新标签页打开数据源" | "Open source in new tab" |
| `market.overview.eventRadar.sourceLabel` | "来源：{{name}}" | "Source: {{name}}" |

资产详情页继续用 `assetDetail.risk.events.title`，无需新 key。

## 6. 数据流

```
[FRED Releases Calendar API]   [Polymarket Gamma API]
   └── richson/datasources/fred.py            └── richson/datasources/polymarket.py
       get_upcoming_releases()                    get_gold_relevant_markets()  (现有)
                  ↓                                          ↓
       richson/api/events.py::get_event_radar()
            ├─ 查询 7 天窗口的 FRED releases
            ├─ 用 config/event_metadata.py 注入 impact/goldDirection/zh title
            ├─ 拼 sourceUrl: https://fred.stlouisfed.org/release?rid={rid}
            ├─ 合并 Polymarket 条目，拼 polymarket.com/event/{slug}
            └─ merge + sort by date + cap to 7 days
                  ↓
       richson/schemas/events.py::EventItem (+sourceUrl/sourceName/releaseId)
                  ↓ (FastAPI JSON 响应 with alias)
       backend/internal/api/v2/event.go (handler 纯透传)
                  ↓
       frontend/src/features/event-radar/api.ts (requestPublic)
                  ↓
       frontend/src/features/event-radar/hooks.ts::useEventRadar()
                  ↓
       frontend/src/features/event-radar/event-radar-section.tsx (组件，整行 anchor)
                  ↓ 引用方
            ├─ pages/market-overview/market-overview-page.tsx
            └─ pages/asset-detail/event-calendar.tsx (新)
```

## 7. 错误处理与降级

| 场景 | 行为 |
|------|------|
| FRED key 缺失或 placeholder | `FREDClient._disabled=true`，`get_upcoming_releases()` 返回 `[]`，事件雷达只剩 Polymarket 条目 |
| FRED 接口 429 / 5xx | richson 内部 retry 一次，仍失败返回 `[]`，记 warning 日志，不影响 Polymarket |
| Polymarket 接口失败 | 现有 try/except 已处理，返回 `[]`，FRED 部分不受影响 |
| FRED + Polymarket 都失败 | richson 返回空 `events: []`，前端渲染 "无事件" 提示（现有逻辑） |
| 后端代理 richson 失败 | 现有 handler 返回 5xx，前端 EventRadarSection 进入 isError，渲染 retry 按钮 |
| `sourceUrl` 空 | 行不可点击，无 hover，无 cursor 改变 |

## 8. 设计审查产物（design-review.md 5 passes）

### 8.1 Pass 1 状态空间表

事件雷达对每一条事件有以下维度：

| 维度 | 取值 |
|------|------|
| `source` | `fred` / `polymarket` |
| `impact` | `high` / `medium` / `low` |
| `goldDirection` | `bullish` / `bearish` / `neutral` / `null` |
| `probability` | `number(0-1)` / `null` |
| `sourceUrl` | `string` / `null` |

总组合 2 × 3 × 4 × 2 × 2 = 96，重要分类如下：

| 组合 | 分类 | 处理 |
|------|------|------|
| `source=fred, sourceUrl=null` | Forbidden | richson 必须为所有 FRED 条目拼出 sourceUrl，不可能为 null。代码中 `release_id is not None → sourceUrl = ".../release?rid={id}"` 是无条件路径 |
| `source=polymarket, sourceUrl=null` | Forbidden | richson 必须为所有 Polymarket 条目拼出 sourceUrl（slug 一定存在，否则该条不入榜） |
| `source=fred, probability=non-null` | Forbidden | FRED 数据不带概率，`probability` 永远为 null |
| `source=polymarket, probability=null` | Valid | Polymarket 概率字段缺失时允许，前端隐藏概率展示 |
| `source=fred, goldDirection=null` | Valid | event_metadata 表未命中的 release（如未列入白名单的 release）→ goldDirection=null |
| `impact 全部组合 × source` | Valid | 每个 source 都可能命中 high/medium/low |
| `sourceUrl=null` 整体 | Transient | DTO 允许，但实际数据流中不会产生。若产生，前端必须能渲染（不可点击 div） |

**前端 EventRow 组件渲染分支必须涵盖**：
- 可点击 + 概率 + 24h change（Polymarket 完整）
- 可点击 + 无概率 + 无 change（FRED 标准）
- 不可点击（防御 sourceUrl=null，理论不出现）

### 8.2 Pass 2 文件不变量影响表

| 文件 | 现有契约 | 本次改动影响 |
|------|---------|------------|
| `richson/src/richson/datasources/fred.py` | `_disabled=True` 时所有 `_fetch_series` 短路返回 None；fredapi `Fred` client 是 lazy init | 新增 `get_upcoming_releases()` 方法必须沿用 `_disabled` 检查，返回 `[]` 而非抛异常；不能用 fredapi（fredapi 没有 releases endpoint），改为 httpx 直连 `https://api.stlouisfed.org/fred/releases/dates`，复用 self._api_key + self._timeout + self._max_retries |
| `richson/src/richson/datasources/polymarket.py` | `get_gold_relevant_markets()` 已 cache 15 分钟，slug 已在 raw 字段 | 不修改 client，只在 events.py 里读 `market["slug"]` 拼 URL |
| `richson/src/richson/api/events.py` | `_FIXED_EVENTS` 是入榜源；返回 dict 的 keys 与 EventItem alias 对齐 | **完全删除 `_FIXED_EVENTS`**；改为 `await fred_client.get_upcoming_releases(7)`；返回 dict 增加 sourceUrl/sourceName/releaseId 字段 |
| `richson/src/richson/schemas/events.py` | `EventItem` 字段名通过 alias 映射 snake_case ↔ camelCase；`populate_by_name=True` 允许两种形式 | 新增 3 个字段，全部用 `Field(alias="...")` 模式；保持 `populate_by_name=True` |
| `backend/internal/richson/types.go` | Pointer 类型用于 nullable（已有先例：GoldDirection/Probability/ProbabilitySource/ProbabilityChange24h） | 新增 `*string SourceUrl` / `*string SourceName` / `*int ReleaseId`，与现有 nullable 字段一致使用指针 |
| `backend/internal/api/v2/event.go` | handler 纯透传 richson 响应，不改字段 | 不需要修改，pointer 类型自动透传 null |
| `frontend/src/features/event-radar/types.ts` | 字段命名 camelCase，与后端 json tag 完全一致 | 新增 3 个字段，可空字段用 `\| null` 而非 `?:`，与现有 nullable 字段保持一致 |
| `frontend/src/features/event-radar/api.ts` | 用 `requestPublic` 调 `/events/radar`，无需 token | 不修改 |
| `frontend/src/features/event-radar/hooks.ts` | TanStack Query，staleTime / cacheTime 已配置 | 不修改（只要 EventDto 类型扩展即可） |
| `frontend/src/features/event-radar/event-radar-section.tsx` | EventRow 是 `<div>`，无点击；title/date/tags 布局已稳定；通过 useToken() 取颜色 | 整行 div 改为 `<a href={sourceUrl} target="_blank" rel="noopener noreferrer">` 当 sourceUrl 非 null；保持视觉布局不变；增加 hover 样式（背景色 colorBgTextHover） |
| `frontend/src/features/event-radar/index.ts` | barrel 导出 EventRadarSection / useEventRadar / EventDto / EventRadarDto | 检查是否需要新增导出（如果 asset-detail 用 barrel 导入） |
| `frontend/src/pages/asset-detail/event-calendar.tsx` | 占位 Card，返回 "—"，无 props | 改为 `<EventRadarSection data={...} isLoading={...} isError={...} onRetry={...}/>`，调 `useEventRadar()`；保留 Card title 用 `assetDetail.risk.events.title` |
| `frontend/src/pages/asset-detail/risk-tab.tsx` | 引用 `<EventCalendar/>`，无 props | 不修改（EventCalendar 内部自己 hook） |
| `frontend/src/pages/market-overview/components/event-radar-section.tsx` | 行情概览页本地组件 | **删除该文件**（迁移到 features/event-radar/ 作为统一组件），market-overview-page.tsx 改为从 features barrel 导入 |
| `frontend/src/i18n/locales/{zh,en}/market.json` | namespace 拆分模式 | 新增 `openSourceTooltip` / `sourceLabel` keys |

### 8.3 Pass 3 替代路径验证

| 路径 | 设计如何处理 |
|------|------------|
| 用户在新标签打开 sourceUrl 后点 Back | 不影响事件雷达页面（新标签独立），原页面状态完全保留 |
| FRED 接口连续超时 / 429 | richson 内部 retry max=2，全部失败返回 []，日志 warning，不抛异常；上游响应仍包含 Polymarket 条目；前端不进入 isError |
| Polymarket cache 命中但 FRED 实时拉新 | 同请求内部并发：FRED 用 `await asyncio.to_thread`，Polymarket 也是 `await asyncio.to_thread`；两个独立 try/except，互不影响 |
| 同一会话内连续打开多个资产详情页 | useEventRadar 使用统一 query key `["events", "radar"]`，TanStack Query 跨页面共享缓存，不会重复请求 |
| FRED key 被运行时切换（dev/prod） | richson 启动时初始化 `_disabled`，不支持热切换。运行中 key 失效会从 `disabled=False` 走 retry 失败路径，仍返回 []。可接受 |
| 用户禁用 JavaScript / 浏览器拦截弹窗 | anchor 是原生 `<a target="_blank">`，浏览器原生行为，不依赖 JS。被拦截时浏览器会提示用户允许 |
| sourceUrl 是 javascript: / data: 等 XSS scheme | richson 端拼 URL 时硬编码 `https://` 前缀，不接受外部输入。前端额外用 `rel="noopener noreferrer"` 防 reverse tabnabbing |
| 移动端触屏点击 | anchor 在移动端原生支持触屏，与桌面端 cursor 行为无关 |

### 8.4 Pass 4 Pre-mortem

上线后最可能的 5 个 bug：

| 现象 | 根因 | 设计防御 |
|------|------|---------|
| 事件雷达突然空了，前端只显示"无事件" | FRED key 失效 + Polymarket 也间歇性失败 | richson 在 events.py 末尾加日志，记录 fred_count / polymarket_count / total_count，便于排查；前端"无事件"文案保留为现有 i18n key |
| 点击 FRED 事件跳到 fred.stlouisfed.org/release?rid=10 但页面不存在 | release_id 误用（如 GDP 实际是 53 不是别的） | event_metadata.py 表中每条注释里写明从 https://fred.stlouisfed.org/release?rid=N 验证过；新增 release 必须在 PR description 附验证截图 URL |
| 资产详情页和行情概览页事件列表不一致 | 两处独立 query key 或独立组件 | 强制共用同一个 hook + 同一个组件；EventRadarSection 从 `features/event-radar/` 唯一来源导出 |
| Pyodide / SSR 环境下整行 anchor 报错 | 现在没 SSR，但未来如果引入会触发 | anchor 是纯 HTML，无客户端依赖；ssr 环境下天然兼容 |
| sourceUrl 在 backend 序列化时丢失 | Go pointer 字段忘了 `omitempty` 或 alias 不对 | TRD 强制要求 backend `*string` + json tag 与 frontend camelCase 完全一致；contract-drift 三端对齐 checklist 在 Plan 的最后一个 step 强制执行 |

### 8.5 Pass 5 自反驳

**推荐"FRED 完全替换 _FIXED_EVENTS"的反驳**：
- 担忧：FRED 不提供 FOMC Meeting 本身（只提供 Press Release rid=101），用户可能习惯了"FOMC Meeting Speeches"这条
- 防御：event_metadata.py 表中可以为 rid=101 设置 zh_title="FOMC 利率决议与新闻发布"；FOMC Speeches 这种非 release 事件本来就不该混在 release calendar，去掉合理

**推荐"不按 assetType 过滤"的反驳**：
- 担忧：A 股资产详情页显示 US CPI 看起来不相关
- 防御：黄金资产、美股资产用同一份 macro 列表是合理的（CPI 影响所有美元资产）；A 股用户也关心美联储决策（联动汇率和 ETF）；后续如确实需要 A 股专属事件再单独做（YAGNI）

**推荐"整行 anchor 跳外链"的反驳**：
- 担忧：用户可能希望站内查看摘要而不是被弹到外站
- 防御：FRED 页面信息密度足够，做站内 Drawer 等于复制 FRED 内容；外链是最低维护成本最高信任度的方案；后续真的需要再加 Drawer

### 8.6 Pass 审查中发现并已修复的 gap

1. 初版未考虑 `market-overview/components/event-radar-section.tsx` 是 page-local 组件，与 features 目录 EventRadarSection 重名；如果两处独立维护必然漂移。修复：本次把行情概览页的 page-local 组件**删除**，统一从 `features/event-radar/` 导出。
2. 初版未考虑 fredapi Python 库本身不提供 releases endpoint（fredapi 是 series-only wrapper）；拼 URL 直接调用 FRED API 必须用 httpx。修复：TRD 中明确 `get_upcoming_releases` 走 httpx 直连，不走 fredapi。
3. 初版未考虑 FRED `releases/dates` 默认参数 `include_release_dates_with_no_data=false` 会**排除未来日期**（因为未来还没有数据）。修复：必须传 `include_release_dates_with_no_data=true` 才能拿到未来 7 天的 release 计划。
4. 初版未明确 sourceUrl 的安全性（XSS、reverse tabnabbing）。修复：URL 全部由 richson 拼 `https://` 硬编码前缀；前端 anchor 必须 `rel="noopener noreferrer"`。
5. 初版未考虑 release_id 的 React key 稳定性问题；用 `${date}-${title}-${idx}` 作 key 在数据更新时容易 reorder。修复：新增 releaseId 字段后 FRED 条目的 key 用 `fred-{releaseId}-{date}`，Polymarket 条目用 `poly-{slug}`。

### 8.7 剩余待用户决策的 gap

无。用户已授权全权处理。

## 9. 成功指标

- 事件雷达条目日期与 FRED 官网 https://fred.stlouisfed.org/releases/calendar 完全一致
- 鼠标悬停事件行有视觉反馈，点击在新标签打开正确的 release 页（如 CPI → rid=10）
- 资产详情页 event-calendar 不再显示 "—"，与行情概览页内容一致
- FRED key 删除时 richson 日志无 ERROR 级风暴，事件雷达仍能加载（仅显示 Polymarket 条目）
- 三端类型检查全通过；端到端 Network 抓包字段名对齐
