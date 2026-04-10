# 货币展示偏好 PRD

## 一、背景与目标

Richman 目前所有金额展示硬编码为人民币（¥），不适合同时持有 A 股、港股、美股的用户。本功能允许用户选择偏好的展示货币（CNY / USD / HKD），前端根据实时汇率将所有 CNY 存储值换算为目标货币后展示，保持数据存储层单一真实源不变。

目标：用户在设置中选择一次货币，所有金额展示场景（仪表盘、决策卡、持仓列表）无缝切换，不需要用户手动换算。

## 二、用户故事

- 作为港股投资者，我希望看到持仓金额以 HK$ 展示，这样我能直觉地与港股市值对比
- 作为美股投资者，我希望看到建议调仓金额以美元展示，方便估算实际操作成本
- 作为主做 A 股的用户，我希望默认保持人民币展示，切换货币不影响我的使用习惯

## 三、功能范围

### 3.1 支持的货币

| 代码 | 名称 | 符号 | 基准方向 |
|------|------|------|----------|
| CNY | 人民币 | ¥ | 基准货币，不换算 |
| USD | 美元 | $ | 1 CNY = X USD |
| HKD | 港元 | HK$ | 1 CNY = X HKD |

默认值：CNY（与现有行为一致，已有用户无感知变化）

### 3.2 换算范围

以下场景的所有金额跟随 displayCurrency 换算：

- 仪表盘顶部条（总资本展示、聚合 P&L 金额、持仓 P&L 金额）
- 决策卡摘要（持仓金额、市值、执行计划步骤金额）
- 卡片详情页 CardHero（持仓金额）
- 持仓列表（仓位金额列）
- 持仓交易页（加权成本、总买入、总卖出）
- Portfolio 总资本行

不换算的场景：
- 用户在设置页输入总资本时始终使用 CNY（输入层保持 CNY，展示层换算）
- 后端 API 全部返回 CNY 值，换算在前端完成

### 3.3 汇率数据

- 来源：扩展现有 Yahoo Finance 客户端，通过 `USDCNY=X`、`HKDCNY=X` ticker 获取
- 格式：`rates[currency]` = 1 CNY 等于多少该货币（乘法语义）
  - 例：`rates.USD = 0.1382` 表示 1 CNY = 0.1382 USD
  - 换算公式：`displayAmount = cnyAmount × rates[displayCurrency]`
- 缓存：后端内存 Lazy TTL 1 小时，`sync.RWMutex` 保护并发
- 失败降级：汇率不可用时返回空 rates，前端静默回退展示 CNY

## 四、后端设计

### 4.1 数据库变更

新增迁移文件，在 `users` 表添加一列：

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS display_currency VARCHAR(8) NOT NULL DEFAULT 'CNY';

ALTER TABLE users
    ADD CONSTRAINT chk_users_display_currency
    CHECK (display_currency IN ('CNY', 'USD', 'HKD'));
```

### 4.2 用户设置模型变更

`backend/internal/model/user.go` 和 `backend/internal/service/user_settings/service.go` 新增字段：

```go
// User model
DisplayCurrency string `json:"displayCurrency"`

// UserSettings DTO
DisplayCurrency string `json:"displayCurrency"`

// PatchUserSettings
DisplayCurrency *string `json:"displayCurrency,omitempty"`
```

验证规则（PatchUserSettings.validatePatch）：
- `displayCurrency` 只允许 "CNY" | "USD" | "HKD"
- 空字符串视为无效，拒绝

### 4.3 汇率服务

新建 `backend/internal/service/exchangerate/service.go`：

```
ExchangeRateService
  rates     map[string]float64   // "USD" → 0.1382, "HKD" → 1.0764
  updatedAt time.Time
  ttl       time.Duration        // 1 hour
  mu        sync.RWMutex
```

公开方法：
- `GetRates(ctx) (map[string]float64, error)`：若缓存未过期直接返回；否则调 Yahoo Finance 拉取两个 ticker，更新缓存后返回
- 拉取失败：返回上一次有效 rates（若首次失败返回空 map，不 panic）

Yahoo Finance 扩展：在现有 `backend/internal/datasource/yahoo/client.go` 新增：
- `FetchForexRate(ctx, fromTo string) (float64, error)`，参数如 `"USDCNY=X"`
- 内部复用现有 `FetchQuote` 逻辑，取 `closingPrice`

### 4.4 汇率 API 端点

新建 `backend/internal/api/v1/exchange_rates.go`：

```
GET /api/v1/exchange-rates
Authorization: Bearer <token>

Response 200:
{
  "rates": {
    "CNY": 1.0,
    "USD": 0.1382,
    "HKD": 1.0764
  },
  "updatedAt": "2026-04-11T10:00:00Z"
}
```

汇率获取失败时仍返回 200，`rates` 仅包含 `{"CNY": 1.0}`，`updatedAt` 为零值，前端据此判断降级。

## 五、前端设计

### 5.1 类型扩展

`frontend/src/features/user-settings/types.ts`：

```typescript
export type DisplayCurrency = "CNY" | "USD" | "HKD";

// UserSettings 新增
displayCurrency: DisplayCurrency;

// PatchUserSettings 新增
displayCurrency?: DisplayCurrency;
```

### 5.2 汇率 Hook

新建 `frontend/src/domain/money/useExchangeRates.ts`：

```typescript
interface ExchangeRates {
  rates: Partial<Record<DisplayCurrency, number>>;
  updatedAt: string;
}

export function useExchangeRates(): { rates: Partial<Record<DisplayCurrency, number>> }
```

- React Query，queryKey: `["exchange-rates"]`
- staleTime: 30 分钟，retry: 2
- 失败或加载中：返回 `{ rates: {} }`

同步新增 API 函数 `getExchangeRates()` 在 `frontend/src/domain/money/api.ts`（新建文件）。

### 5.3 formatAmount 升级

`frontend/src/domain/money/format.ts` 的 `formatAmount` 改为：

```typescript
formatAmount(amount: number, locale?: string, currency?: DisplayCurrency): string
```

内部改用 `Intl.NumberFormat` 的 `style: "currency"` + `currencyDisplay: "narrowSymbol"`，按货币固定 locale：
- USD → `en-US`
- HKD → `zh-HK`
- CNY → `locale === "zh" ? "zh-CN" : "en-US"`

`formatPercentWithAmount` 和 `formatAmountOrNull` 同步透传 `currency` 参数。

### 5.4 useMoney 扩展

`frontend/src/domain/money/useMoney.ts`：

- 读取 `settings.displayCurrency`（默认 "CNY"）
- 读取 `useExchangeRates().rates`
- 新增内部纯函数 `convertCny(amountCny, currency, rates)`：
  - `currency === "CNY"` → 直接返回原值
  - 否则 `amountCny * (rates[currency] ?? 0)` 若 rate 为 0 则返回原值（降级）
- `format` 和 `formatAmountOnly` 在格式化前先调用 `convertCny`
- 新增属性 `currency: DisplayCurrency` 供少数需要感知货币的组件读取

### 5.5 需修复的直接调用点

以下两处绕过了 `useMoney`，需要修复以使货币切换生效：

| 文件 | 问题 | 修复方式 |
|------|------|----------|
| `DashboardTopStrip.tsx` 第 77 行 | `capitalDisplay` 手写 `¥ + Intl.NumberFormat("zh-CN")` | 改用 `money.formatAmountOnly(totalCapitalCny)` |
| `TotalCapitalRow.tsx` | 调用 `formatAmount(totalCapital, i18n.language)` 不传 currency | 改用 `useMoney().formatAmountOnly` |

### 5.6 设置页面 UI

在 `frontend/src/pages/settings/PreferencesTab.tsx` 语言选择项下方新增"展示货币"：

- 组件：`Radio.Group`，选项：CNY / USD / HKD
- 变更后调用 `patchUserSettings({ displayCurrency })`
- hint 文字：汇率每小时自动更新，金额为参考值

### 5.7 i18n

同步更新 `zh/settings.json` 和 `en/settings.json`：

```json
"displayCurrency": "展示货币" / "Display Currency",
"displayCurrencyHint": "金额将根据实时汇率换算展示，每小时更新一次" / "Amounts are converted using live exchange rates, updated hourly",
"currencyOptions": {
  "CNY": "人民币（¥）" / "Chinese Yuan (¥)",
  "USD": "美元（$）" / "US Dollar ($)",
  "HKD": "港元（HK$）" / "Hong Kong Dollar (HK$)"
}
```

## 六、状态空间

| displayCurrency | rates 可用 | 展示结果 |
|-----------------|-----------|----------|
| CNY | 任意 | ¥，不换算（现有行为）|
| USD | 已加载 | $，乘以 rates.USD |
| USD | 加载中/失败 | ¥ 降级，不报错 |
| HKD | 已加载 | HK$，乘以 rates.HKD |
| HKD | 加载中/失败 | ¥ 降级，不报错 |
| 任意 | rates.X = 0 | ¥ 降级（0 汇率视为不可用）|

## 七、决策记录

**D1：换算在前端完成，不在后端**
理由：后端换算会影响 Privacy Guard（`positionAmount` 字段保护逻辑），且分析管道强制要求"仅百分比，不见绝对金额"。前端是展示层，换算属于展示逻辑。

**D2：汇率不入库，内存 Lazy TTL**
理由：汇率是短期数据，重启后重拉即可。数据库持久化引入额外迁移且收益低，对投资组合展示场景 1 小时精度已足够。

**D3：扩展现有 Yahoo Finance 客户端**
理由：项目已有 Yahoo Finance 集成，`USDCNY=X` ticker 可用现有 HTTP 客户端直接获取，零额外依赖。

**D4：格式化改用 Intl style:currency**
理由：手动拼接 `"¥" + format()` 无法正确处理负号位置、locale 差异。Intl 标准 API 已覆盖所有目标浏览器环境（Vite + React 19）。

**D5：设置页输入总资本始终用 CNY**
理由：`totalCapitalCny` 是后端存储字段，换算方向单一（CNY 为基准）。若允许以外币输入，需要"写入时反换算"逻辑，增加复杂度且意义不大。

## 八、替代路径与降级

- 汇率 API 失败：静默回退 CNY，用户不感知错误（非核心功能）
- 用户切换货币后页面无刷新：React Query 的 `displayCurrency` 变化自动触发 `useMoney` 重新 memo，所有金额组件随之重渲染
- 汇率精度：存储 6 位有效数字，展示时 `maximumFractionDigits: 0` 取整，满足投资组合展示精度要求
