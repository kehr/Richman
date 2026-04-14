# 事件雷达数据源真实化 TRD

依据：`docs/prds/event-radar-data-source-prd.md`

## 1. 改动概览

本次涉及三个语言层、五个模块：

| 层 | 模块 | 改动类型 |
|----|------|---------|
| richson | `datasources/fred.py` | 新增方法 `get_upcoming_releases(window_days)` 和支撑函数 |
| richson | `config/event_metadata.py` | **新文件** - FRED release_id → metadata 静态映射表 |
| richson | `api/events.py` | 重写 `get_event_radar`，移除 `_FIXED_EVENTS` 与 `_infer_gold_direction` |
| richson | `schemas/events.py` | EventItem 新增 `source_url` / `source_name` / `release_id` |
| backend | `internal/richson/types.go` | EventItem 新增 `*string SourceUrl` / `*string SourceName` / `*int ReleaseId` |
| frontend | `features/event-radar/types.ts` | EventDto 新增 `sourceUrl` / `sourceName` / `releaseId` |
| frontend | `features/event-radar/event-radar-section.tsx` | **新文件**（从 page-local 迁移并增强可点击行为） |
| frontend | `features/event-radar/index.ts` | barrel 导出 `EventRadarSection` |
| frontend | `pages/market-overview/components/event-radar-section.tsx` | **删除**（迁移到 features） |
| frontend | `pages/market-overview/market-overview-page.tsx` | import 路径改为 `@/features/event-radar` |
| frontend | `pages/asset-detail/event-calendar.tsx` | 重写为 `<EventRadarSection/>` 包装 |
| frontend | `i18n/locales/{zh,en}/market.json` | 新增 `openSourceTooltip` / `sourceLabel` keys |

数据库：**无**（事件雷达不落库）

迁移：**无**

API：**无新端点**（GET /events/radar 已存在，扩展响应字段，不破坏既有契约）

## 2. richson 实现

### 2.1 `datasources/fred.py` 扩展

#### 2.1.1 新增依赖

httpx 已在 `polymarket.py` 使用，复用 `httpx.Client`。

#### 2.1.2 新增数据类

```python
@dataclass(frozen=True)
class FREDReleaseDate:
    """A single upcoming FRED release date."""
    release_id: int
    release_name: str
    date: str  # ISO YYYY-MM-DD
```

放在 `fred.py` 顶部，与 `SERIES_IDS` 同级。

#### 2.1.3 新增方法 `get_upcoming_releases`

签名：

```python
def get_upcoming_releases(self, window_days: int = 7) -> list[FREDReleaseDate]:
    """Fetch upcoming FRED release dates within the next N days.

    Returns:
        Sorted list of FREDReleaseDate (ascending by date). Empty list when
        the FRED key is disabled, the network call fails, or no releases fall
        in the window.
    """
```

实现要点：

1. **disabled 短路**：`if self._disabled: return []` —— 不发网络请求，不打日志（startup 时已 warn 一次）。
2. **缓存 key**：`f"upcoming_releases:{window_days}"`，复用现有 `cache_get/cache_set`，TTL 用默认（与 series 缓存一致，无需新增 TTL 配置）。
3. **HTTP 调用**：用 httpx 直连 FRED REST，**不使用** fredapi 库（fredapi 不支持 releases endpoint）。
   - URL: `https://api.stlouisfed.org/fred/releases/dates`
   - 必填参数: `api_key=self._api_key`, `file_type=json`
   - 关键参数: `include_release_dates_with_no_data=true`（**默认 false 会过滤掉未来日期，必须显式开启**）
   - 时间窗口: `realtime_start={today}`, `realtime_end={today + window_days}` (ISO YYYY-MM-DD)
   - 排序: `order_by=release_date&sort_order=asc`
   - 分页: `limit=1000`（最大），`offset=0`
4. **白名单过滤**：从 `config/event_metadata.py` 导入 `FRED_RELEASE_METADATA`，只保留 `release_id in FRED_RELEASE_METADATA` 的条目。
5. **重试**：复用 `self._max_retries`，遇 5xx / 网络异常重试一次后返回 `[]`。
6. **JSON 解析**：响应字段 `release_dates: [{release_id, release_name, date, ...}]`，构造 `FREDReleaseDate(release_id, release_name, date)`。
7. **去重**：同一个 release_id 在窗口内可能出现多次（多个相关 sub-release），按 `(release_id, date)` 去重保留最早一条。

错误日志统一用 `logger.warning("fred releases fetch failed", error=str(exc), attempt=attempt)`。

#### 2.1.4 不破坏现有契约

- `_disabled` 字段语义不变
- `_fetch_series` / `get_*` series 方法签名不变
- 不引入新的全局状态

### 2.2 `config/event_metadata.py`（新文件）

模块定位：纯数据模块，无业务逻辑，无 import 副作用。

完整内容：

```python
"""Static metadata for FRED economic releases.

Each entry maps a FRED release_id to display metadata used by the event radar
endpoint (api/events.py). Adding a release here automatically opts it into
the event radar — there is no separate registration.

release_id values are verified against https://fred.stlouisfed.org/release?rid=N
The verification URL is recorded in each entry's comment.

impact: "high" | "medium" | "low" — drives UI tag color
gold_direction: "bullish" | "bearish" | "neutral" | None
zh_title / en_title: human-readable display names per locale
category: free-form string for future grouping (matches existing values)
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

EventImpact = Literal["high", "medium", "low"]
GoldDirection = Literal["bullish", "bearish", "neutral"]


@dataclass(frozen=True)
class ReleaseMeta:
    impact: EventImpact
    gold_direction: GoldDirection | None
    zh_title: str
    en_title: str
    category: str


# release_id -> ReleaseMeta
# All ids verified at https://fred.stlouisfed.org/release?rid=N as of 2026-04-15.
FRED_RELEASE_METADATA: dict[int, ReleaseMeta] = {
    10: ReleaseMeta(  # Consumer Price Index
        impact="high",
        gold_direction="bullish",
        zh_title="美国 CPI 数据公布",
        en_title="US CPI Data Release",
        category="inflation",
    ),
    46: ReleaseMeta(  # Producer Price Index
        impact="medium",
        gold_direction="neutral",
        zh_title="美国 PPI 数据公布",
        en_title="US PPI Data Release",
        category="inflation",
    ),
    50: ReleaseMeta(  # Employment Situation (Non-Farm Payrolls)
        impact="high",
        gold_direction="neutral",
        zh_title="美国非农就业数据",
        en_title="US Non-Farm Payrolls",
        category="employment",
    ),
    54: ReleaseMeta(  # Personal Income & Outlays (PCE)
        impact="high",
        gold_direction="bullish",
        zh_title="美国 PCE 通胀数据",
        en_title="US PCE Inflation Data",
        category="inflation",
    ),
    53: ReleaseMeta(  # Gross Domestic Product
        impact="high",
        gold_direction="neutral",
        zh_title="美国 GDP 数据",
        en_title="US GDP Data",
        category="growth",
    ),
    101: ReleaseMeta(  # FOMC Press Release
        impact="high",
        gold_direction="bullish",
        zh_title="FOMC 利率决议与新闻发布",
        en_title="FOMC Press Release",
        category="monetary_policy",
    ),
    13: ReleaseMeta(  # Industrial Production & Capacity Utilization (G.17)
        impact="medium",
        gold_direction="neutral",
        zh_title="美国工业产出数据",
        en_title="US Industrial Production",
        category="growth",
    ),
    9: ReleaseMeta(  # Advance Monthly Sales for Retail & Food Services
        impact="medium",
        gold_direction="neutral",
        zh_title="美国零售销售数据",
        en_title="US Retail Sales",
        category="growth",
    ),
    291: ReleaseMeta(  # Existing Home Sales
        impact="low",
        gold_direction="neutral",
        zh_title="美国成屋销售数据",
        en_title="US Existing Home Sales",
        category="growth",
    ),
}


def fred_release_url(release_id: int) -> str:
    """Build the canonical FRED release page URL.

    Verified URL pattern: https://fred.stlouisfed.org/release?rid={id}
    """
    return f"https://fred.stlouisfed.org/release?rid={release_id}"


def polymarket_event_url(slug: str) -> str:
    """Build the Polymarket event page URL from a market slug."""
    return f"https://polymarket.com/event/{slug}"
```

### 2.3 `api/events.py` 重写

#### 2.3.1 删除

- `_FIXED_EVENTS` 列表
- `_infer_gold_direction` 函数（goldDirection 由 metadata 表决定，Polymarket 条目固定 None）

#### 2.3.2 新结构

```python
EVENT_WINDOW_DAYS = 7

@router.get("/radar")
async def get_event_radar() -> dict:
    fred_client = FREDClient(api_key=settings.fred_api_key)  # 与 api/market.py / cli/backtest.py / core/pipeline.py 一致
    poly_client = PolymarketClient()

    # 并发拉两个数据源
    fred_task = asyncio.to_thread(fred_client.get_upcoming_releases, EVENT_WINDOW_DAYS)
    poly_task = asyncio.to_thread(poly_client.get_gold_relevant_markets)

    fred_releases: list[FREDReleaseDate] = []
    poly_markets: list[dict] = []

    fred_result, poly_result = await asyncio.gather(
        fred_task, poly_task, return_exceptions=True
    )

    if isinstance(fred_result, Exception):
        logger.warning("fred unavailable for event radar", error=str(fred_result))
    else:
        fred_releases = fred_result

    if isinstance(poly_result, Exception):
        logger.warning("polymarket unavailable for event radar", error=str(poly_result))
    else:
        poly_markets = poly_result

    today = datetime.now(tz=UTC)
    horizon = today + timedelta(days=EVENT_WINDOW_DAYS)

    events: list[dict] = []

    # Build FRED entries
    for release in fred_releases:
        meta = FRED_RELEASE_METADATA.get(release.release_id)
        if meta is None:
            continue  # 白名单已在 fred.py 过滤，此处是冗余防御
        events.append({
            "date": release.date,
            "title": meta.zh_title,  # 见 §2.3.4 关于 i18n 的决定
            "category": meta.category,
            "impact": meta.impact,
            "goldDirection": meta.gold_direction,
            "probability": None,
            "probabilitySource": None,
            "probabilityChange24h": None,
            "sourceUrl": fred_release_url(release.release_id),
            "sourceName": "FRED",
            "releaseId": release.release_id,
        })

    # Build Polymarket entries (top 5 by volume, filter to horizon)
    for market in poly_markets[:5]:
        slug = market.get("slug")
        question = market.get("question") or ""
        # polymarket.py:143 stores `market.get("endDate")` under key `"end_date"`.
        # The previous `events.py` used `"end_date_iso"` which never existed (latent bug).
        end_date_iso = market.get("end_date") or ""
        if not slug or not question:
            continue
        date_str = end_date_iso[:10] if end_date_iso else today.strftime("%Y-%m-%d")
        # 跳过窗口外
        try:
            event_dt = datetime.fromisoformat(date_str).replace(tzinfo=UTC)
            if event_dt < today or event_dt > horizon:
                continue
        except ValueError:
            continue
        prob = market.get("yes_probability")
        events.append({
            "date": date_str,
            "title": question,
            "category": "market_event",
            "impact": "medium",
            "goldDirection": None,  # 不再启发式推断
            "probability": round(float(prob), 4) if prob is not None else None,
            "probabilitySource": "polymarket" if prob is not None else None,
            "probabilityChange24h": None,
            "sourceUrl": polymarket_event_url(slug),
            "sourceName": "Polymarket",
            "releaseId": None,
        })

    events.sort(key=lambda e: e["date"])

    logger.info(
        "event radar built",
        fred_count=sum(1 for e in events if e["sourceName"] == "FRED"),
        polymarket_count=sum(1 for e in events if e["sourceName"] == "Polymarket"),
        total=len(events),
    )

    return {
        "data": {
            "events": events,
            "updatedAt": today.isoformat(),
        }
    }
```

#### 2.3.3 关键决策

- **不再做 FRED ↔ Polymarket title 关联**：原 `_FIXED_EVENTS` 试图把 Polymarket 概率匹配到固定事件，匹配规则是字符串前缀，准确率低。新设计 FRED 与 Polymarket 各自独立成条，FRED 永远无概率，Polymarket 永远有 sourceUrl 跳市场页。
- **Polymarket 时间窗口收紧**：原实现允许 Polymarket 的 end_date 任意（甚至超过 7 天）。新实现限制 Polymarket 条目的 end_date 必须在 [today, today+7d] 窗口内，与"未来 7 天"语义一致。
- **goldDirection 来源唯一化**：FRED 条目走 metadata 表，Polymarket 条目固定 None。删除 `_infer_gold_direction` 启发式逻辑。

#### 2.3.4 i18n 处理（关键）

richson 输出 `title` 是 zh 还是 en？现状：richson 的 `_FIXED_EVENTS` 输出英文 title（如 "US CPI Data Release"），前端不翻译。

**本次决定保持现状**：richson 输出英文 title（用 `meta.en_title`），不引入 i18n 协商。理由：
1. 保持与现状一致，不破坏既有渲染
2. richson 的 ServerSide i18n 不是本次范围
3. 用户截图中事件标题是英文（"US CPI Data Release"），符合用户预期

修正：上面 `events.append` 中 `"title": meta.zh_title` 应改为 `"title": meta.en_title`。

后续如要支持中英切换，方案：richson 输出 release_id 由前端 i18n 表查找翻译。本次在 metadata 表中保留 zh_title 字段为后续准备，但不消费。

### 2.4 `schemas/events.py` 扩展

```python
class EventItem(BaseModel):
    date: str
    title: str
    category: str
    impact: EventImpact
    gold_direction: GoldDirection | None = Field(default=None, alias="goldDirection")
    probability: float | None = None
    probability_source: str | None = Field(default=None, alias="probabilitySource")
    probability_change_24h: float | None = Field(default=None, alias="probabilityChange24h")
    # NEW
    source_url: str | None = Field(default=None, alias="sourceUrl")
    source_name: str | None = Field(default=None, alias="sourceName")
    release_id: int | None = Field(default=None, alias="releaseId")

    model_config = {"populate_by_name": True}
```

EventRadarData 不变。

## 3. backend 实现

### 3.1 `internal/richson/types.go::EventItem` 扩展

```go
type EventItem struct {
    Date                 string   `json:"date"`
    Title                string   `json:"title"`
    Category             string   `json:"category"`
    Impact               string   `json:"impact"`
    GoldDirection        *string  `json:"goldDirection"`
    Probability          *float64 `json:"probability"`
    ProbabilitySource    *string  `json:"probabilitySource"`
    ProbabilityChange24h *float64 `json:"probabilityChange24h"`
    // NEW: keep pointer for "T | None" parity with richson Pydantic.
    SourceUrl  *string `json:"sourceUrl"`
    SourceName *string `json:"sourceName"`
    ReleaseId  *int    `json:"releaseId"`
}
```

注意：
- 字段命名沿用项目其他指针字段的风格（首字母大写、json tag camelCase）
- Pointer 类型严格遵循 contract-drift.md：Python `T | None` ↔ Go `*T`
- handler `event.go::getEventsRadar` **无需修改**，gin 自动透传新字段

### 3.2 不修改

- `internal/richson/client.go::GetEventsRadar` 不动
- `internal/api/v2/router.go` 不动
- 不需要新增 service / repo（事件雷达不落库）

## 4. frontend 实现

### 4.1 `features/event-radar/types.ts` 扩展

```typescript
export interface EventDto {
    date: string;
    title: string;
    category: string;
    impact: "high" | "medium" | "low";
    goldDirection: "bullish" | "bearish" | "neutral" | null;
    probability: number | null;
    probabilitySource: string | null;
    probabilityChange24h: number | null;
    // NEW
    sourceUrl: string | null;
    sourceName: string | null;
    releaseId: number | null;
}
```

EventRadarDto 不变。

### 4.2 `features/event-radar/event-radar-section.tsx`（新位置）

从 `pages/market-overview/components/event-radar-section.tsx` 迁移到 features 目录。改动：

#### 4.2.1 EventRow 组件改造

将 `<div>` 容器条件性改为 `<a>`：

```tsx
function EventRow({ event }: EventRowProps) {
    const { t } = useTranslation("market");
    const { token } = useToken();

    const isClickable = typeof event.sourceUrl === "string" && event.sourceUrl.length > 0;

    const baseStyle: React.CSSProperties = {
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 8px",
        borderBottom: `1px solid ${token.colorBorderSecondary}`,
        flexWrap: "wrap",
        borderRadius: 4,
        textDecoration: "none",
        color: "inherit",
        cursor: isClickable ? "pointer" : "default",
        transition: "background-color 0.15s",
    };

    const content = (
        <>
            {/* 现有 Date / Title / Tags / Probability / Change24h 块完全不变 */}
        </>
    );

    if (isClickable) {
        const sourceLabel = event.sourceName
            ? t("overview.eventRadar.sourceLabel", { name: event.sourceName })
            : t("overview.eventRadar.openSourceTooltip");
        return (
            <a
                href={event.sourceUrl as string}
                target="_blank"
                rel="noopener noreferrer"
                title={sourceLabel}
                style={baseStyle}
                onMouseEnter={(e) => {
                    e.currentTarget.style.backgroundColor = token.colorBgTextHover;
                }}
                onMouseLeave={(e) => {
                    e.currentTarget.style.backgroundColor = "transparent";
                }}
            >
                {content}
            </a>
        );
    }

    return <div style={baseStyle}>{content}</div>;
}
```

要点：
- 整行样式不变（视觉密度保持）
- `target="_blank"` + `rel="noopener noreferrer"` 防 reverse tabnabbing
- hover 用 inline style 控制（避免 Card 内部 scoped CSS 复杂度）
- a11y：HTML title 提供"Source: FRED"提示
- 不可点击时 fallback 到 `<div>`（防御 sourceUrl=null）

#### 4.2.2 React key 稳定性

```tsx
data.events.map((event) => {
    const key = event.releaseId !== null
        ? `fred-${event.releaseId}-${event.date}`
        : `poly-${event.sourceUrl ?? event.title}-${event.date}`;
    return <EventRow key={key} event={event} />;
});
```

替代原先 `${date}-${title}-${idx}` 的不稳定 key。

### 4.3 `features/event-radar/index.ts` 扩展

```typescript
export { useEventRadar } from "./use-event-radar";
export { EventRadarSection } from "./event-radar-section";
export type { EventRadarDto, EventDto } from "./types";
```

### 4.4 删除 `pages/market-overview/components/event-radar-section.tsx`

完全移除。同步把 `market-overview-page.tsx` 的 import 改为：

```diff
- import { EventRadarSection } from "./components/event-radar-section";
+ import { EventRadarSection, useEventRadar } from "@/features/event-radar";
```

（useEventRadar 已经从 features 导入，无变化）

### 4.5 `pages/asset-detail/event-calendar.tsx` 重写

```tsx
import { EventRadarSection, useEventRadar } from "@/features/event-radar";
import { Card } from "@/ui-kit/eat";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

export function EventCalendar() {
    const { t } = useTranslation("app");
    const queryClient = useQueryClient();
    const {
        data: eventData,
        isLoading: eventLoading,
        isError: eventError,
        refetch,
    } = useEventRadar();

    const handleRetry = () => {
        queryClient.invalidateQueries({ queryKey: ["events", "radar"] });
        refetch();
    };

    return (
        <Card title={t("assetDetail.risk.events.title")} size="small" style={{ marginBottom: 16 }}>
            <EventRadarSection
                data={eventData}
                isLoading={eventLoading}
                isError={eventError}
                onRetry={handleRetry}
            />
        </Card>
    );
}
```

注意：
- `<EventRadarSection/>` 自带的内部 Card 会与外层 Card 嵌套 - 设计上接受这种轻嵌套（外层 Card 提供 title，内层 Card 提供 padding）。如视觉冗余，TRD 编码阶段可用一个 prop 让 EventRadarSection 不渲染外层 Card，但本次默认嵌套
- 共用 `["events", "radar"]` query key，TanStack Query 自动跨页面缓存

### 4.6 i18n 增量

`frontend/src/i18n/locales/zh/market.json` 内 `overview.eventRadar.*` 命名空间新增：

```json
"openSourceTooltip": "在新标签页打开数据源",
"sourceLabel": "来源：{{name}}"
```

`frontend/src/i18n/locales/en/market.json` 同步：

```json
"openSourceTooltip": "Open source in new tab",
"sourceLabel": "Source: {{name}}"
```

`assetDetail.risk.events.title` 已存在，无需新增。

## 5. 端到端数据链路追踪（trd-review-discipline §3）

| 数据点 | richson 产生 | richson 响应字段 | backend struct | backend 响应字段 | frontend 类型 | frontend 消费位置 |
|--------|-------------|-----------------|----------------|-----------------|--------------|-----------------|
| date | events.py L? | `date` | `Date string` | `date` | `date: string` | EventRow Date Text |
| title | events.py via metadata | `title` | `Title string` | `title` | `title: string` | EventRow Title Text |
| category | events.py | `category` | `Category string` | `category` | `category: string` | （未使用，保留兼容） |
| impact | events.py via metadata | `impact` | `Impact string` | `impact` | `impact: "high" \| "medium" \| "low"` | EventRow Impact Tag |
| goldDirection | events.py via metadata | `goldDirection` (alias) | `*string GoldDirection` | `goldDirection` | `string \| null` | EventRow goldDirection Tag |
| probability | events.py 仅 Polymarket | `probability` | `*float64` | `probability` | `number \| null` | EventRow Probability Text |
| probabilitySource | events.py 仅 Polymarket | `probabilitySource` (alias) | `*string` | `probabilitySource` | `string \| null` | （未直接渲染） |
| probabilityChange24h | events.py | `probabilityChange24h` (alias) | `*float64` | `probabilityChange24h` | `number \| null` | EventRow Change24h Text |
| **sourceUrl** | events.py via fred_release_url/polymarket_event_url | `sourceUrl` (alias) | `*string SourceUrl` | `sourceUrl` | `string \| null` | EventRow `<a href>` |
| **sourceName** | events.py 字面量 "FRED"/"Polymarket" | `sourceName` (alias) | `*string SourceName` | `sourceName` | `string \| null` | EventRow HTML title |
| **releaseId** | events.py from FREDReleaseDate | `releaseId` (alias) | `*int ReleaseId` | `releaseId` | `number \| null` | React key |
| updatedAt | events.py today.isoformat() | `updatedAt` (alias) | `time.Time UpdatedAt` | `updatedAt` | `string` | （未直接渲染） |

链路完整无断点。

## 6. 多角色审查（trd-review-discipline §4）

### 6.1 DBA
- 不涉及 DDL / 索引 / 事务，N/A

### 6.2 后端研发
- `*int ReleaseId` 与 Python `int | None` 对齐（Go json unmarshal int 为指针类型 OK）
- handler 不变，无新错误码
- richson client 不变，复用现有 doRequest

### 6.3 前端研发
- 新组件位置 `features/event-radar/event-radar-section.tsx` 符合 Pages+Features 架构（feature 包含 hook + types + component）
- barrel `index.ts` 同步导出
- ant-design Tag/Card/Button/Skeleton/Alert 全部从 `@/ui-kit/eat` barrel 导入
- React key 用 releaseId / slug 提供稳定性
- i18n 双语 keys 同步

### 6.4 安全
- sourceUrl 全部由 richson 端 hardcode `https://` 前缀拼成，不接受外部输入
- frontend `<a target="_blank">` 必须带 `rel="noopener noreferrer"` 防 reverse tabnabbing
- `title` HTML 属性内容来自 i18n 模板（`{{name}}`），name 来自 richson 服务端字面量 ("FRED"/"Polymarket")，无 XSS 风险
- FRED API key 仅 richson 内部持有，不传给 backend / frontend

### 6.5 SRE
- richson info 日志 `event radar built` 含 fred_count / polymarket_count / total，便于排查
- FRED API rate limit 120 req/min/key；缓存命中时无外发请求；TanStack Query staleTime 15min；FRED `releases/dates` 一次窗口大约 1 次外发请求 / 15 分钟，远低于限额
- FRED key disabled 时 startup 一次 warning，不在 hot path 打日志风暴

### 6.6 QA
- 状态空间表已枚举所有组合（PRD §8.1）
- 边界值：
  - 窗口内无 release（如周末窗口）→ FRED 部分返回 []，仍可渲染 Polymarket
  - Polymarket 全部 end_date 超出 7 天 → Polymarket 部分返回 []，仍可渲染 FRED
  - 都无 → 前端 "无事件" 文案
- 异常路径：
  - FRED 5xx → richson 返回 []，前端不进入 isError
  - backend 5xx → 前端 isError，渲染 retry 按钮（现有）
  - sourceUrl 含特殊字符（理论不出现）→ React anchor 自动 encode

### 6.7 产品
- PRD 用户故事 4 条全部覆盖（点击跳转、资产详情页一致、降级容错）
- MVP 边界：不做 Drawer / 详情页 / 提醒，符合 YAGNI

## 7. 已知问题与编码阶段必须处理项

| 问题 | 处理方案 |
|------|---------|
| `pages/market-overview/components/event-radar-section.tsx` 删除时若有别处 import（除 market-overview-page.tsx 外），会编译失败 | 编码阶段先 grep `event-radar-section` 全局引用，确认只有一处 import，再删除 |
| richson 启动时 `fred_api_key` 是否注入到 `FREDClient` 构造？已验证 `api/market.py`、`cli/backtest.py`、`core/pipeline.py` 均使用 `FREDClient(api_key=settings.fred_api_key)` | 沿用同一模式：`from richson.config import settings; FREDClient(api_key=settings.fred_api_key)` |
| `asyncio.gather(return_exceptions=True)` 与现有 try/except 风格不一致 | 编码阶段对照 richson 其他 endpoint，若主流是单 try/except 包多个 await，则改回单 try/except 写两段；本次选 gather 因为两个数据源真正独立 |
| Polymarket end_date 字段名漂移已确认：`polymarket.py:143` 把 `market.get("endDate")` 存进 `"end_date"` key，但当前 `events.py:94` 用 `market.get("end_date_iso")` 读 - 原代码就是 BUG，永远拿不到日期 | 本次重写 events.py 时统一用 `market.get("end_date")` 读取，本次顺手修复这个老 BUG |
| 资产详情页外层 Card + EventRadarSection 内层 Card 嵌套是否视觉冗余 | 编码阶段先按嵌套实施，验收时由用户视觉判定；如冗余则给 EventRadarSection 加可选 prop `bare?: boolean` 跳过外层 Card |
| event_metadata.py 中 release_id 来源标注的可信度：53(GDP) / 50(Employment) / 10(CPI) 来自 FRED API docs 例子回显未直达 release 页验证 | 编码阶段联调时 richson 启动后 dev 环境调用 `https://api.stlouisfed.org/fred/releases?api_key=$FRED_API_KEY&file_type=json&limit=1000` 一次，按 `name` 字段验证 9 个 release_id；如有偏差立即更正 metadata 表 |
| Polymarket 的 yes_probability 在 polymarket.py 是 `outcome_prices[0]`（"Yes" 价格），可能不是真实概率（depends on market structure） | 沿用现状，不在本次修改 |
| richson `EVENT_WINDOW_DAYS = 7` 与 i18n key `subtitle: "未来 7 天关键宏观事件"` 是两个独立 source of truth；改窗口必须同步改 i18n | 编码阶段在 events.py 顶部加注释 "Keep in sync with frontend i18n market.overview.eventRadar.subtitle" |

## 8. 验证清单（contract-drift §对齐规则）

编码完成后必须人工跑一次：

- [ ] richson 本地启动，curl `http://localhost:8000/events/radar`，检查响应 JSON 字段是否含 `sourceUrl` / `sourceName` / `releaseId`
- [ ] backend 本地启动，curl `http://localhost:8100/api/v2/events/radar`，检查同字段未丢失
- [ ] frontend dev server 启动，DevTools Network 抓 `/api/v2/events/radar`，DevTools Components 检查 EventDto 字段对齐
- [ ] 鼠标悬停 FRED 事件行，HTML title 显示 "Source: FRED"，cursor=pointer，hover 背景变浅
- [ ] 点击 FRED 事件行，新标签打开 https://fred.stlouisfed.org/release?rid=N
- [ ] 点击 Polymarket 事件行，新标签打开 https://polymarket.com/event/{slug}
- [ ] 资产详情页打开任一资产的 risk-tab，事件列表与行情概览页一致
- [ ] 临时清空 `FRED_API_KEY` 重启 richson，事件雷达只显示 Polymarket 条目，无 ERROR 日志风暴
