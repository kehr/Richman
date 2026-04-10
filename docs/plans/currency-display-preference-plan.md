# 货币展示偏好实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 在用户设置中新增展示货币偏好（CNY / USD / HKD），前端根据后端提供的实时汇率换算并展示所有金额。

**架构：** 后端新增 DB 列存偏好、扩展 Yahoo Finance 客户端获取汇率、新建 ExchangeRateService 做内存 Lazy TTL 缓存、新增 `/exchange-rates` REST 端点；前端 `useMoney` hook 读偏好和汇率后做换算，所有金额展示组件通过 `useMoney` 自动响应切换，不需要逐个修改。

**设计文档：** PRD `docs/prds/currency-display-preference-prd.md`，TRD `docs/trds/currency-display-preference-trd.md`

**Tech Stack：** Go / Gin / PostgreSQL（后端），React 19 / TanStack Query v5 / Ant Design 6 / TypeScript strict（前端）

**并行执行策略：**
- 后端 worktree：Task B1 → Task B5（并行）+ Task B2 → Task B3 → Task B4
- 前端 worktree：Task F1 → Task F2 + Task F3（并行）→ Task F4 → Task F5 → Task F6 + Task F7 + Task F8（并行）
- 两个 worktree 可同时开工，前端 F2 编写可先于后端 B4 完成（接口已在 TRD 定义）

## 后端任务

### Task B1：数据库迁移

**设计依据：** PRD §4.1 / TRD §2.1

**文件：**
- 创建：`backend/db/migration/016_user_display_currency.up.sql`
- 创建：`backend/db/migration/016_user_display_currency.down.sql`

**依赖：** 无

- [ ] 按 TRD §2.1 写 up/down 迁移文件，在 `users` 表添加 `display_currency VARCHAR(8) NOT NULL DEFAULT 'CNY'`，并加 CHECK 约束 `IN ('CNY', 'USD', 'HKD')`
- [ ] 执行迁移

```bash
cd backend && make migrate-up
```

- [ ] 验证列和约束存在

```bash
docker exec -it richman-postgres psql -U postgres richman \
  -c "\d users" | grep display_currency
# 期望：display_currency | character varying(8) | not null | default 'CNY'
```

- [ ] 提交

```bash
git add backend/db/migration/016_user_display_currency.up.sql \
        backend/db/migration/016_user_display_currency.down.sql
git commit -m "feat(db): add display_currency column to users"
```

### Task B2：Yahoo Finance FetchForexRate 扩展

**设计依据：** TRD §2.2

**文件：**
- 修改：`backend/internal/datasource/yahoo/client.go`

**依赖：** 无（与 B1 可并行）

- [ ] 在 `client.go` 末尾追加 `FetchForexRate` 方法，完整接口和实现见 TRD §2.2
  - 内部调用现有 `c.FetchQuote(ctx, ticker)` 取 `quote.Close`
  - ticker 如 `"USDCNY=X"` 表示 1 USD = ? CNY；返回值即 Close 原始价（调用方负责取倒数）
  - `quote.Close <= 0` 时返回 error

- [ ] 验证编译

```bash
cd backend && go build ./internal/datasource/yahoo/...
```

- [ ] 提交

```bash
git add backend/internal/datasource/yahoo/client.go
git commit -m "feat(yahoo): add FetchForexRate for forex pair tickers"
```

### Task B3：ExchangeRateService

**设计依据：** TRD §2.3

**文件：**
- 创建：`backend/internal/service/exchangerate/service.go`

**依赖：** B2（需要 `yahoo.Client.FetchForexRate`）

- [ ] 新建 package `exchangerate`，按 TRD §2.3 实现 `Service` struct、`NewService`、`GetRates`、`refresh` 方法
  - 常量：`defaultTTL = time.Hour`，ticker `"USDCNY=X"`，`"HKDCNY=X"`
  - 汇率公式：`rates["USD"] = 1.0 / yahooRate`（取倒数）
  - `GetRates`：读锁检查 TTL → 过期则调 `refresh` → 返回缓存
  - `refresh`：部分失败时保留旧值，记录 `zap.Warn`，不清除已有条目
  - 初始 cached 包含 `"CNY": 1.0`

- [ ] 验证编译

```bash
cd backend && go build ./internal/service/exchangerate/...
```

- [ ] 验证 `make check` 全绿

```bash
cd backend && make check
```

- [ ] 提交

```bash
git add backend/internal/service/exchangerate/service.go
git commit -m "feat(exchangerate): lazy TTL exchange rate service via Yahoo Finance"
```

### Task B4：ExchangeRatesHandler 与路由注册

**设计依据：** TRD §2.4

**文件：**
- 创建：`backend/internal/api/v1/exchange_rates.go`
- 修改：`backend/cmd/server/main.go`（第 286-334 行区域，在 dashboardHandler 注册后）

**依赖：** B3

- [ ] 新建 `ExchangeRatesHandler`，按 TRD §2.4 实现 `NewExchangeRatesHandler`、`RegisterRoutes`、`Get`
  - 路由：`GET /api/v1/exchange-rates`（需 auth middleware）
  - 响应格式：`{"data": {"rates": {...}, "updatedAt": "..."}}`（使用 `c.JSON(200, gin.H{"data": ...})`）

- [ ] 在 `main.go` 初始化 `exchangeRateService` 和 `exchangeRatesHandler`，并调用 `RegisterRoutes`
  - `yahooClient` 从 main.go 现有 yahoo 初始化处取共享实例（grep `yahoo.New` 或 `yahoo.NewClient`）

- [ ] 验证编译与服务启动

```bash
cd backend && go build ./cmd/server/... && make check
```

- [ ] 手动冒烟（需服务在跑）

```bash
curl -s -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/exchange-rates | jq .
# 期望：{"data":{"rates":{"CNY":1,"USD":0.13xx,"HKD":1.0x},"updatedAt":"..."}}
```

- [ ] 提交

```bash
git add backend/internal/api/v1/exchange_rates.go \
        backend/cmd/server/main.go
git commit -m "feat(api): GET /exchange-rates endpoint with lazy TTL cache"
```

### Task B5：UserSettings 后端字段接入

**设计依据：** TRD §2.5

**文件：**
- 修改：`backend/internal/model/user.go`
- 修改：`backend/internal/service/user_settings/service.go`
- 修改：`backend/internal/repo/user_repo.go`

**依赖：** B1（DB 列已存在）；可与 B2 并行开工，合入时 B1 需先合入主干

- [ ] `model/user.go`：`User` struct 在 `Language` 字段后新增 `DisplayCurrency string \`json:"displayCurrency"\``

- [ ] `service.go`：`UserSettings` DTO 和 `PatchUserSettings` 各新增 `DisplayCurrency` 字段，`validatePatch` 新增校验（只允许 "CNY" | "USD" | "HKD"），`GetUserSettings` 和 `PatchUserSettings` 均读写该字段，完整规格见 TRD §2.5

- [ ] `user_repo.go`：`UpdateUserSettings` 动态 SQL 在现有 COALESCE 链追加 `display_currency = COALESCE($N::VARCHAR, display_currency)`，`GetUserSettings` SELECT 列表新增 `display_currency`，映射到 `User.DisplayCurrency`

- [ ] 验证

```bash
cd backend && make check
```

- [ ] 集成验证（需服务在跑）

```bash
# PATCH 设置 USD
curl -s -X PATCH http://localhost:8080/api/v1/user/settings \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"displayCurrency":"USD"}' | jq .data.displayCurrency
# 期望："USD"

# GET 验证持久化
curl -s http://localhost:8080/api/v1/user/settings \
  -H "Authorization: Bearer <token>" | jq .data.displayCurrency
# 期望："USD"

# 非法值应被拒绝
curl -s -X PATCH http://localhost:8080/api/v1/user/settings \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"displayCurrency":"EUR"}' | jq .error
# 期望：400 错误信息
```

- [ ] 提交

```bash
git add backend/internal/model/user.go \
        backend/internal/service/user_settings/service.go \
        backend/internal/repo/user_repo.go
git commit -m "feat(settings): add displayCurrency field to user settings"
```

## 前端任务

### Task F1：DisplayCurrency 类型

**设计依据：** TRD §3.1

**文件：**
- 修改：`frontend/src/features/user-settings/types.ts`

**依赖：** 无

- [ ] 在 `types.ts` 新增 `DisplayCurrency = "CNY" | "USD" | "HKD"` 类型，`UserSettings` 新增 `displayCurrency: DisplayCurrency`，`PatchUserSettings` 新增 `displayCurrency?: DisplayCurrency`，完整规格见 TRD §3.1

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/features/user-settings/types.ts
git commit -m "feat(types): add DisplayCurrency and displayCurrency to UserSettings"
```

### Task F2：Exchange Rates API 函数

**设计依据：** TRD §3.2

**文件：**
- 创建：`frontend/src/domain/money/api.ts`

**依赖：** F1（需要 `DisplayCurrency` 类型）

- [ ] 新建 `api.ts`，按 TRD §3.2 定义 `ExchangeRatesData` interface 和 `getExchangeRates()` 函数
  - 使用 `request<ApiResponse<ExchangeRatesData>>("/exchange-rates")` 模式（同 `features/user-settings/api.ts`）
  - 导入 `request` from `"@/domain/http/client"`，`ApiResponse` from `"@/domain/http/types"`

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/domain/money/api.ts
git commit -m "feat(money): add getExchangeRates API function"
```

### Task F3：formatAmount 签名扩展与测试

**设计依据：** TRD §3.4

**文件：**
- 修改：`frontend/src/domain/money/format.ts`
- 修改：`frontend/src/domain/money/format.test.ts`

**依赖：** F1（`DisplayCurrency` 类型）；可与 F2 并行

- [ ] `format.ts`：新增内部函数 `getCurrencyLocale(currency, uiLocale)`（USD→`"en-US"`，HKD→`"zh-HK"`，CNY→locale 感知），更新 `formatAmount` 使用 `Intl.NumberFormat` `style:"currency"` + `currencyDisplay:"narrowSymbol"` + `maximumFractionDigits:0`，新增 `currency` 可选参数（默认 `"CNY"`），`formatPercentWithAmount` 和 `formatAmountOrNull` 同步透传 `currency` 参数，完整规格见 TRD §3.4

- [ ] `format.test.ts`：新增 USD 和 HKD 测试用例（不删除现有 CNY 测试）
  - `formatAmount(1234, "en", "USD")` → `"$1,234"`
  - `formatAmount(1234, "zh", "HKD")` → `"HK$1,234"`
  - `formatAmount(-500, "en", "USD")` → `"-$500"`

- [ ] 验证测试通过

```bash
cd frontend && pnpm test src/domain/money/format.test.ts
```

- [ ] 验证 lint

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/domain/money/format.ts \
        frontend/src/domain/money/format.test.ts
git commit -m "feat(format): support USD/HKD in formatAmount via Intl style:currency"
```

### Task F4：useExchangeRates Hook

**设计依据：** TRD §3.3

**文件：**
- 创建：`frontend/src/domain/money/useExchangeRates.ts`

**依赖：** F2

- [ ] 新建 `useExchangeRates.ts`，按 TRD §3.3 实现 hook
  - `queryKey: ["exchange-rates"]`，`staleTime: 30 * 60 * 1000`，`retry: 2`
  - `select: (d) => d.data`（解包 ApiResponse）
  - 返回 `{ rates: data?.rates ?? {} }`，加载中或失败时返回空对象

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/domain/money/useExchangeRates.ts
git commit -m "feat(money): useExchangeRates hook with 30min staleTime"
```

### Task F5：useMoney 扩展

**设计依据：** TRD §3.5

**文件：**
- 修改：`frontend/src/domain/money/useMoney.ts`

**依赖：** F3（formatAmount 含 currency），F4（useExchangeRates）

- [ ] 更新 `useMoney.ts`，按 TRD §3.5：
  - 新增 `import { useExchangeRates } from "./useExchangeRates"`
  - 读取 `settings?.displayCurrency ?? "CNY"` 作为 `currency`
  - 读取 `useExchangeRates().rates`
  - 新增内部函数 `convertCny(amountCny, currency, rates)`（currency=CNY 直接返回；rate 为 0 或 undefined 则降级返回原值）
  - `format` 和 `formatAmountOnly` 在调用 `formatPercentWithAmount` / `formatAmountOrNull` 前先 `convertCny`，并透传 `currency` 参数
  - 返回对象新增 `currency: DisplayCurrency` 属性
  - `useMemo` 依赖数组加入 `currency` 和 `rates`

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/domain/money/useMoney.ts
git commit -m "feat(money): useMoney reads displayCurrency and converts via exchange rates"
```

### Task F6：Settings UI 货币选择器

**设计依据：** TRD §3.8 / §3.9 / PRD §5

**文件：**
- 修改：`frontend/src/pages/settings/PreferencesTab.tsx`
- 修改：`frontend/src/i18n/locales/zh/settings.json`
- 修改：`frontend/src/i18n/locales/en/settings.json`

**依赖：** F1（DisplayCurrency 类型），F5（useMoney，切换后立即更新展示）；可与 F5 并行写 UI，但全量 lint 需等 F5 完成

- [ ] 两个 i18n 文件在 `preferences` 节点下新增 `displayCurrency`、`displayCurrencyHint`、`currencyOptions.CNY/USD/HKD` key，zh 和 en 必须同步添加，key 规格见 TRD §3.9

- [ ] `PreferencesTab.tsx` 在语言选择 `Form.Item` 后新增货币选择 `Form.Item`：
  - 参照现有 language radio 实现模式（useState + useEffect 同步 settings + patchSettings onChange）
  - 完整交互规格见 TRD §3.8

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 视觉验证：设置页"偏好"标签下能看到货币选择器，切换后所有页面金额更新

- [ ] 提交

```bash
git add frontend/src/pages/settings/PreferencesTab.tsx \
        frontend/src/i18n/locales/zh/settings.json \
        frontend/src/i18n/locales/en/settings.json
git commit -m "feat(settings): add display currency selector to preferences tab"
```

### Task F7：修复 DashboardTopStrip capitalDisplay

**设计依据：** TRD §3.6

**文件：**
- 修改：`frontend/src/pages/dashboard/components/DashboardTopStrip.tsx`（第 74-77 行）

**依赖：** F5（useMoney 已支持 formatAmountOnly 换算）

- [ ] 将 `capitalDisplay` 的硬编码 `¥ + Intl.NumberFormat("zh-CN")` 替换为 `money.formatAmountOnly(totalCapitalCny)`，完整替换规格见 TRD §3.6
  - `money` 已在组件顶部声明，无需新增 import
  - 保留 `const hasCapital = totalCapitalCny != null`（用于条件渲染）

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/pages/dashboard/components/DashboardTopStrip.tsx
git commit -m "fix(dashboard): capitalDisplay uses useMoney for currency-aware formatting"
```

### Task F8：修复 TotalCapitalRow

**设计依据：** TRD §3.7

**文件：**
- 修改：`frontend/src/pages/portfolio/components/TotalCapitalRow.tsx`

**依赖：** F5（useMoney）；可与 F7 并行

- [ ] 按 TRD §3.7 替换 `formatAmount(totalCapital, i18n.language)` 为 `useMoney().formatAmountOnly(totalCapital)`
  - 在组件顶部添加 `const money = useMoney()`
  - 移除不再需要的 `import { formatAmount }` 和多余的 `i18n` 解构（若 `t` 还在用则保留 `useTranslation`）

- [ ] 验证

```bash
cd frontend && pnpm lint:all
```

- [ ] 提交

```bash
git add frontend/src/pages/portfolio/components/TotalCapitalRow.tsx
git commit -m "fix(portfolio): TotalCapitalRow uses useMoney for currency-aware total capital"
```

## 自检清单（实施完成后）

- [ ] `cd backend && make check` 全绿
- [ ] `cd frontend && pnpm lint:all` 全绿
- [ ] 设置页可切换 CNY / USD / HKD，切换后 Dashboard 和 Portfolio 所有金额即时更新
- [ ] 切换为 USD 时，总资本 ¥100,000 显示为 $13,xxx
- [ ] 汇率接口不可用时，金额降级为 CNY 展示，页面不崩溃
- [ ] 新用户默认 CNY，与现有行为一致
