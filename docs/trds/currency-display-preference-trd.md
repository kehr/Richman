# 货币展示偏好 TRD

设计依据：docs/prds/currency-display-preference-prd.md

## 一、架构总览

```
backend/db/migration/016_user_display_currency.up.sql
backend/internal/datasource/yahoo/client.go        (新增 FetchForexRate)
backend/internal/service/exchangerate/service.go   (新建)
backend/internal/api/v1/exchange_rates.go          (新建)
backend/internal/model/user.go                     (新增字段)
backend/internal/service/user_settings/service.go  (新增字段处理)
backend/internal/repo/user_repo.go                 (新增列 COALESCE)
backend/cmd/server/main.go                         (注册新 handler)

frontend/src/features/user-settings/types.ts       (新增 DisplayCurrency)
frontend/src/domain/money/api.ts                   (新建)
frontend/src/domain/money/useExchangeRates.ts      (新建)
frontend/src/domain/money/format.ts                (signature 扩展)
frontend/src/domain/money/format.test.ts           (新增 currency 测试)
frontend/src/domain/money/useMoney.ts              (读 currency + rates)
frontend/src/pages/settings/PreferencesTab.tsx     (货币选择 UI)
frontend/src/pages/dashboard/components/DashboardTopStrip.tsx  (修 capitalDisplay)
frontend/src/pages/portfolio/components/TotalCapitalRow.tsx    (改走 useMoney)
frontend/src/i18n/locales/zh/settings.json         (新增 key)
frontend/src/i18n/locales/en/settings.json         (新增 key)
```

## 二、后端接口设计

### 2.1 数据库迁移

文件：`backend/db/migration/016_user_display_currency.up.sql`

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS display_currency VARCHAR(8) NOT NULL DEFAULT 'CNY';

ALTER TABLE users
    ADD CONSTRAINT chk_users_display_currency
    CHECK (display_currency IN ('CNY', 'USD', 'HKD'));
```

文件：`backend/db/migration/016_user_display_currency.down.sql`

```sql
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_display_currency;
ALTER TABLE users DROP COLUMN IF EXISTS display_currency;
```

### 2.2 Yahoo Finance 客户端扩展

文件：`backend/internal/datasource/yahoo/client.go`，在现有方法后追加：

```go
// FetchForexRate fetches the latest exchange rate for a forex pair ticker.
// ticker uses Yahoo Finance format, e.g. "USDCNY=X" returns how many CNY
// equals 1 USD (the Close price of the pair). Callers that need the inverse
// rate (1 CNY = X USD) must compute 1/result themselves.
func (c *Client) FetchForexRate(ctx context.Context, ticker string) (float64, error) {
    quote, err := c.FetchQuote(ctx, ticker)
    if err != nil {
        return 0, fmt.Errorf("fetch forex rate for %s: %w", ticker, err)
    }
    if quote.Close <= 0 {
        return 0, fmt.Errorf("invalid forex rate %.6f for %s", quote.Close, ticker)
    }
    return quote.Close, nil
}
```

### 2.3 ExchangeRateService

文件：`backend/internal/service/exchangerate/service.go`（新建 package）

```go
package exchangerate

import (
    "context"
    "sync"
    "time"

    "github.com/yourorg/richman/backend/internal/datasource/yahoo"
    "go.uber.org/zap"
)

const (
    defaultTTL = time.Hour
    // Yahoo tickers: "XYZABC=X" means 1 XYZ = ? ABC
    // "USDCNY=X" → 1 USD = ? CNY   →  invert to get 1 CNY = ? USD
    // "HKDCNY=X" → 1 HKD = ? CNY   →  invert to get 1 CNY = ? HKD
    tickerUSD = "USDCNY=X"
    tickerHKD = "HKDCNY=X"
)

// Rates holds exchange rates expressed as "1 CNY = X foreign currency".
// CNY is always present with value 1.0. Fields missing from the map mean
// the rate is temporarily unavailable; callers should fall back to CNY display.
type Rates struct {
    Values    map[string]float64 `json:"rates"`
    UpdatedAt time.Time          `json:"updatedAt"`
}

// Service fetches and caches forex exchange rates with a TTL-based lazy refresh.
// The cache is warm-on-first-request; no background goroutine is used.
type Service struct {
    yahoo  *yahoo.Client
    logger *zap.Logger

    mu        sync.RWMutex
    cached    Rates
    fetchedAt time.Time
    ttl       time.Duration
}

// NewService constructs a Service with a 1-hour TTL.
// The initial cache contains only CNY=1.0 until the first GetRates call.
func NewService(yahooClient *yahoo.Client, logger *zap.Logger) *Service {
    return &Service{
        yahoo:  yahooClient,
        logger: logger,
        ttl:    defaultTTL,
        cached: Rates{Values: map[string]float64{"CNY": 1.0}},
    }
}

// GetRates returns cached rates when fresh; otherwise fetches from Yahoo Finance.
// On fetch failure, returns the last known rates so callers degrade gracefully.
// The returned map always contains "CNY": 1.0.
func (s *Service) GetRates(ctx context.Context) Rates

// refresh fetches USDCNY=X and HKDCNY=X from Yahoo Finance, inverts the
// pair prices to produce "1 CNY = X foreign" rates, and updates the cache.
// Partial success is accepted: if one ticker fails, the other still updates.
func (s *Service) refresh(ctx context.Context) error
```

`refresh` 实现要点：
- 并发获取两个 ticker（`sync.WaitGroup` 或顺序均可，流量极低）
- 汇率公式：`rates["USD"] = 1.0 / usdcnyClose`，`rates["HKD"] = 1.0 / hkdcnyClose`
- 任一 ticker 失败则保留旧值，记录 warn 日志，不清除已有 USD/HKD 条目

### 2.4 Exchange Rates Handler

文件：`backend/internal/api/v1/exchange_rates.go`（新建）

```go
package v1

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/yourorg/richman/backend/internal/service/exchangerate"
)

type ExchangeRatesHandler struct {
    svc *exchangerate.Service
}

func NewExchangeRatesHandler(svc *exchangerate.Service) *ExchangeRatesHandler {
    return &ExchangeRatesHandler{svc: svc}
}

func (h *ExchangeRatesHandler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
    rg.GET("/exchange-rates", auth, h.Get)
}

// Get handles GET /api/v1/exchange-rates.
// Returns rates as "1 CNY = X currency". On rate fetch failure, rates contains
// only {"CNY": 1.0} so the frontend can detect degraded mode.
func (h *ExchangeRatesHandler) Get(c *gin.Context) {
    rates := h.svc.GetRates(c.Request.Context())
    c.JSON(http.StatusOK, gin.H{
        "data": gin.H{
            "rates":     rates.Values,
            "updatedAt": rates.UpdatedAt,
        },
    })
}
```

`main.go` 新增（在 `dashboardHandler` 注册后）：

```go
exchangeRateService := exchangerate.NewService(yahooClient, zapLogger)
exchangeRatesHandler := v1.NewExchangeRatesHandler(exchangeRateService)
exchangeRatesHandler.RegisterRoutes(apiV1, authMiddleware)
```

`yahooClient` 是 `yahoo.NewClient(...)` 的现有实例，从 main.go 中取出共享。

### 2.5 User Settings 变更

**model/user.go**，`User` struct 新增字段（按现有字段顺序添加在 `Language` 后）：

```go
DisplayCurrency string `json:"displayCurrency"`
```

**service/user_settings/service.go**，`UserSettings` DTO 和 `PatchUserSettings` 各新增：

```go
// UserSettings DTO
DisplayCurrency string `json:"displayCurrency"`

// PatchUserSettings
DisplayCurrency *string `json:"displayCurrency,omitempty"`
```

`validatePatch` 新增校验：

```go
if patch.DisplayCurrency != nil {
    switch *patch.DisplayCurrency {
    case "CNY", "USD", "HKD":
        // valid
    default:
        return fmt.Errorf("displayCurrency must be one of: CNY, USD, HKD")
    }
}
```

`GetUserSettings` 和 `PatchUserSettings` 的查询 / 映射逻辑均需包含 `display_currency` 列。

**repo/user_repo.go**，`UpdateUserSettings` 动态 SQL 在现有 COALESCE 链中新增：

```go
// 在 SET 子句中添加（伪代码，参照现有 COALESCE 模式）:
// display_currency = COALESCE($N::VARCHAR, display_currency)
// 当 patch.DisplayCurrency != nil 时传入具体值，否则传 nil
```

`GetUserSettings` 的 SELECT 列表新增 `display_currency`，映射到 `User.DisplayCurrency`。

## 三、前端接口设计

### 3.1 UserSettings 类型

文件：`frontend/src/features/user-settings/types.ts`

```typescript
export type DisplayCurrency = "CNY" | "USD" | "HKD";

// 在 UserSettings interface 新增（Language 字段后）：
displayCurrency: DisplayCurrency;

// 在 PatchUserSettings interface 新增：
displayCurrency?: DisplayCurrency;
```

### 3.2 汇率 API 函数

文件：`frontend/src/domain/money/api.ts`（新建）

```typescript
import { request } from "@/domain/http/client";
import type { ApiResponse } from "@/domain/http/types";
import type { DisplayCurrency } from "@/features/user-settings";

export interface ExchangeRatesData {
    rates: Partial<Record<DisplayCurrency, number>>;
    updatedAt: string;
}

export function getExchangeRates(): Promise<ApiResponse<ExchangeRatesData>> {
    return request<ApiResponse<ExchangeRatesData>>("/exchange-rates");
}
```

### 3.3 useExchangeRates Hook

文件：`frontend/src/domain/money/useExchangeRates.ts`（新建）

```typescript
import { useQuery } from "@tanstack/react-query";
import type { DisplayCurrency } from "@/features/user-settings";
import { getExchangeRates } from "./api";

export const EXCHANGE_RATES_QUERY_KEY = ["exchange-rates"] as const;

// useExchangeRates fetches and caches exchange rates from the backend.
// staleTime 30 minutes; backend refreshes its cache hourly from Yahoo Finance.
// Returns { rates: {} } when loading or on error — callers degrade to CNY display.
export function useExchangeRates(): { rates: Partial<Record<DisplayCurrency, number>> } {
    const { data } = useQuery({
        queryKey: EXCHANGE_RATES_QUERY_KEY,
        queryFn: getExchangeRates,
        staleTime: 30 * 60 * 1000,
        retry: 2,
        select: (d) => d.data,
    });
    return { rates: data?.rates ?? {} };
}
```

### 3.4 formatAmount 签名扩展

文件：`frontend/src/domain/money/format.ts`

新增内部辅助（未导出）：

```typescript
// getCurrencyLocale maps a DisplayCurrency to the Intl locale that produces
// the most natural symbol for that currency regardless of the UI language.
// USD → "en-US" ($), HKD → "zh-HK" (HK$), CNY → ui locale (¥)
function getCurrencyLocale(currency: DisplayCurrency, uiLocale: string): string {
    if (currency === "USD") return "en-US";
    if (currency === "HKD") return "zh-HK";
    return uiLocale === "zh" ? "zh-CN" : "en-US";
}
```

更新后的 `formatAmount`（现有调用方无需改动，参数向后兼容）：

```typescript
export function formatAmount(
    amount: number,
    locale = "en",
    currency: DisplayCurrency = "CNY",
): string {
    if (Number.isNaN(amount) || amount === 0) {
        // format 0 through Intl so symbol is always correct
        const fmt = getNumberFormat(getCurrencyLocale(currency, locale), {
            style: "currency",
            currency,
            maximumFractionDigits: 0,
            currencyDisplay: "narrowSymbol",
        });
        return fmt.format(0);
    }
    const fmt = getNumberFormat(getCurrencyLocale(currency, locale), {
        style: "currency",
        currency,
        maximumFractionDigits: 0,
        currencyDisplay: "narrowSymbol",
    });
    return fmt.format(amount);
}
```

同步更新两个公开函数的签名（透传 `currency` 参数）：

```typescript
export function formatPercentWithAmount(
    pct: number,
    amount: number | null | undefined,
    hasCapital: boolean,
    locale = "en",
    currency: DisplayCurrency = "CNY",
): string

export function formatAmountOrNull(
    amount: number | null | undefined,
    hasCapital: boolean,
    locale = "en",
    currency: DisplayCurrency = "CNY",
): string | null
```

`format.test.ts` 新增针对 USD / HKD 的测试用例（不删除现有 CNY 测试）。

### 3.5 useMoney 扩展

文件：`frontend/src/domain/money/useMoney.ts`

新增内部纯函数：

```typescript
// convertCny converts a CNY amount to the target display currency.
// Returns amountCny unchanged when: currency is "CNY", rate is missing, or rate is 0.
// Returns null when amountCny is null/undefined (preserves null semantics).
function convertCny(
    amountCny: number | null | undefined,
    currency: DisplayCurrency,
    rates: Partial<Record<DisplayCurrency, number>>,
): number | null {
    if (amountCny == null) return null;
    if (currency === "CNY") return amountCny;
    const rate = rates[currency];
    if (!rate) return amountCny; // degraded: show CNY value
    return amountCny * rate;
}
```

更新后的 hook：

```typescript
export function useMoney() {
    const { data: settings } = useUserSettings();
    const { rates } = useExchangeRates();
    const { i18n } = useTranslation();

    const hasCapital = settings?.totalCapitalCny != null;
    const currency: DisplayCurrency = settings?.displayCurrency ?? "CNY";
    const locale = i18n.language;

    return useMemo(
        () => ({
            hasCapital,
            currency,
            format: (pct: number, amountCny?: number | null) =>
                formatPercentWithAmount(
                    pct,
                    convertCny(amountCny, currency, rates),
                    hasCapital,
                    locale,
                    currency,
                ),
            formatAmountOnly: (amountCny?: number | null) =>
                formatAmountOrNull(
                    convertCny(amountCny, currency, rates),
                    hasCapital,
                    locale,
                    currency,
                ),
        }),
        [hasCapital, currency, rates, locale],
    );
}
```

`rates` 对象引用：React Query 在 staleTime 内返回同一个对象引用，不会每次渲染都触发 useMemo 重算。

### 3.6 DashboardTopStrip 修复

文件：`frontend/src/pages/dashboard/components/DashboardTopStrip.tsx`

```typescript
// 删除（第 74-77 行）：
const hasCapital = totalCapitalCny != null;
const capitalDisplay = hasCapital
    ? `¥${new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 0 }).format(totalCapitalCny as number)}`
    : null;

// 替换为：
const hasCapital = totalCapitalCny != null;
const capitalDisplay = money.formatAmountOnly(totalCapitalCny);
```

`money` 已在组件顶部通过 `const money = useMoney()` 获取，无需新增 import。`capitalDisplay` 为 null 时的渲染逻辑（显示"设置资本" CTA）保持不变。

### 3.7 TotalCapitalRow 修复

文件：`frontend/src/pages/portfolio/components/TotalCapitalRow.tsx`

```typescript
// 新增 hook 调用（在组件顶部）：
const money = useMoney();

// 删除：
import { formatAmount } from "@/domain/money/format";
const { t, i18n } = useTranslation("app");  // i18n 仅用于 formatAmount，删掉该解构

// 替换金额格式化（hasCapital 分支内）：
// 旧：{formatAmount(totalCapital, i18n.language)}
// 新：{money.formatAmountOnly(totalCapital)}
```

`money.formatAmountOnly(totalCapital)` 在 `hasCapital` 分支内必然返回非 null（因为 `totalCapital != null` 保证了 `money.hasCapital = true`）。

### 3.8 Settings PreferencesTab UI

文件：`frontend/src/pages/settings/PreferencesTab.tsx`

新增 state 和 handler（参照现有 language 实现模式）：

```typescript
const [currency, setCurrency] = useState<DisplayCurrency>(
    () => settings?.displayCurrency ?? "CNY",
);

useEffect(() => {
    if (settings?.displayCurrency) setCurrency(settings.displayCurrency);
}, [settings?.displayCurrency]);

const handleCurrencyChange = (e: RadioChangeEvent) => {
    const val = e.target.value as DisplayCurrency;
    setCurrency(val);
    patchSettings({ displayCurrency: val });
};
```

JSX（插入在语言 `Form.Item` 之后）：

```tsx
<Form.Item
    label={t("settings.preferences.displayCurrency")}
    extra={t("settings.preferences.displayCurrencyHint")}
>
    <Radio.Group value={currency} onChange={handleCurrencyChange}>
        <Radio value="CNY">{t("settings.preferences.currencyOptions.CNY")}</Radio>
        <Radio value="USD">{t("settings.preferences.currencyOptions.USD")}</Radio>
        <Radio value="HKD">{t("settings.preferences.currencyOptions.HKD")}</Radio>
    </Radio.Group>
</Form.Item>
```

### 3.9 i18n Key 规格

两个文件均需同步添加，key 路径挂在现有 `settings.preferences` 节点下：

```jsonc
// zh/settings.json
"preferences": {
    // ... 现有 key ...
    "displayCurrency": "展示货币",
    "displayCurrencyHint": "金额将根据实时汇率换算展示，每小时自动更新",
    "currencyOptions": {
        "CNY": "人民币（¥）",
        "USD": "美元（$）",
        "HKD": "港元（HK$）"
    }
}

// en/settings.json
"preferences": {
    // ... 现有 key ...
    "displayCurrency": "Display Currency",
    "displayCurrencyHint": "Amounts are converted using live exchange rates, updated hourly",
    "currencyOptions": {
        "CNY": "Chinese Yuan (¥)",
        "USD": "US Dollar ($)",
        "HKD": "Hong Kong Dollar (HK$)"
    }
}
```

## 四、数据流总览

```
用户在 PreferencesTab 选 USD
    → patchUserSettings({ displayCurrency: "USD" })
    → React Query invalidate "user-settings"
    → useUserSettings() 返回新 settings，displayCurrency="USD"
    → useMoney() memo 重算：currency="USD"，从 useExchangeRates 读 rates
    → 所有调用 money.format / money.formatAmountOnly 的组件重渲染
    → convertCny(amountCny, "USD", rates) = amountCny * rates["USD"]
    → formatAmount(convertedAmount, locale, "USD")
       → Intl.NumberFormat("en-US", {style:"currency", currency:"USD"}).format(...)
       → "$13,820"
```

## 五、降级行为规格

| 场景 | `rates[currency]` | `convertCny` 输出 | 展示结果 |
|------|-------------------|--------------------|----------|
| displayCurrency=CNY | 不读取 | 原值（直接返回） | ¥1,234 |
| displayCurrency=USD，rates 已加载 | 0.1382 | amountCny * 0.1382 | $170 |
| displayCurrency=USD，rates 加载中 | undefined | 返回 amountCny（降级）| ¥1,234 |
| displayCurrency=USD，rate=0（异常） | 0 (falsy) | 返回 amountCny（降级）| ¥1,234 |
| displayCurrency=HKD，Yahoo 拉取失败 | undefined | 返回 amountCny（降级）| ¥1,234 |

## 六、实现注意事项

**Yahoo Finance forex ticker 方向**：`USDCNY=X` 表示"1 USD = ? CNY"（Close ≈ 7.24）。要得到"1 CNY = ? USD"需取倒数：`rates["USD"] = 1.0 / yahooRate`。

**`Intl.NumberFormat` narrowSymbol 兼容性**：`currencyDisplay: "narrowSymbol"` 在 Chrome 80+ / Firefox 79+ / Safari 14.1+ 均支持，符合项目 Vite + React 19 目标环境。

**`useMemo` 依赖项 `rates`**：React Query 在 staleTime 内始终返回同一个对象引用（referential equality），不触发无效重算。TTL 过期后重新 fetch 会产生新引用，正确触发 memo 更新。

**`formatAmount` 向后兼容**：新增的 `currency` 参数默认值为 `"CNY"`，所有现有直接调用（如 `ExecutionPlanStrip` 中的 `formatAmount(amount, i18n.language)`）行为不变。

**`format.test.ts` 变更范围**：现有 CNY 测试的期望输出不变（Intl 与手动拼接结果一致）。需新增：`formatAmount(1234, "en", "USD")` 期望 `"$1,234"`，`formatAmount(1234, "zh", "HKD")` 期望 `"HK$1,234"`。
