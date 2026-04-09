# Richman 前端国际化 TRD

本 TRD 承接 `docs/prds/i18n-default-en-prd.md`，聚焦核心模块的接口设计和架构约束。非核心的逐文件字符串迁移只给出约定和示例，不展开每个组件。

## 1. 依赖变更

新增 3 个运行时依赖：

```
pnpm add i18next react-i18next i18next-browser-languagedetector
```

不新增 devDependency。类型生成由 i18next 内置 TypeScript 支持完成（v23+ 自带 module augmentation 声明能力）。

## 2. i18next 初始化（核心模块）

### 2.1 文件：`src/i18n/config.ts`

```typescript
import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

// Static imports — all resources bundled, no async loading, no FOUC
import enCommon from "./locales/en/common.json";
import enAuth from "./locales/en/auth.json";
import enApp from "./locales/en/app.json";
import enSettings from "./locales/en/settings.json";
import zhCommon from "./locales/zh/common.json";
import zhAuth from "./locales/zh/auth.json";
import zhApp from "./locales/zh/app.json";
import zhSettings from "./locales/zh/settings.json";

export const resources = {
  en: { common: enCommon, auth: enAuth, app: enApp, settings: enSettings },
  zh: { common: zhCommon, auth: zhAuth, app: zhApp, settings: zhSettings },
} as const;

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    supportedLngs: ["en", "zh"],
    load: "languageOnly",            // zh-CN -> zh, en-US -> en
    defaultNS: "common",
    ns: ["common", "auth", "app", "settings"],

    interpolation: {
      escapeValue: false,            // React auto-escapes
    },

    react: {
      useSuspense: false,            // sync resources, no Suspense needed
    },

    detection: {
      order: ["localStorage", "navigator"],
      lookupLocalStorage: "richman_locale",
      caches: ["localStorage"],      // no cookie
    },
  });

// Sync <html lang> on every language change
i18n.on("languageChanged", (lng) => {
  document.documentElement.lang = lng;
});

export default i18n;
```

### 2.2 设计约束

- `resources` 必须是 `as const` 以支持类型推断
- `fallbackLng` 固定为 `"en"`，不可改为 `"zh"` 或条件值
- `load: "languageOnly"` 是防御性配置，不可删除（PRD §5.1 阻塞级 gap）
- `supportedLngs` 白名单外的语言值会被丢弃并回退到 `fallbackLng`
- `detection.lookupLocalStorage` 必须沿用 `"richman_locale"` 保持老用户偏好兼容
- 禁止使用 `i18next-http-backend` 或任何异步资源加载

## 3. TypeScript 类型安全

### 3.1 文件：`src/i18n/@types/i18next.d.ts`

```typescript
import type { resources } from "../config";

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: "common";
    resources: (typeof resources)["en"];
  }
}
```

### 3.2 效果

- `t("nav.dashboard")` 有字面量类型提示，拼错 key 会 TS 报错
- `t("auth:loginButton")` 跨 namespace 引用也有类型检查
- JSON 文件新增 key 后 TS 自动感知（因为 `as const` + static import）
- 不需要额外的代码生成步骤或 CLI 工具

### 3.3 约束

- 所有 namespace 的 en JSON 是类型的 source of truth（zh JSON 必须结构对齐，但不参与类型生成）
- 新增 key 时先加 en，再补 zh，TS 不会报 zh 缺 key（这是 i18next 的限制，用 i18next-cli 事后扫描弥补）

## 4. App.tsx 集成层（核心模块）

### 4.1 改造后的 App.tsx 结构

```typescript
import "./i18n/config";                       // side-effect: init i18n
import { createQueryClient } from "@/config/query-client";
import { getThemeConfig } from "@/config/theme";
import { useThemeMode } from "@/domain/ui/use-theme";
import { App as AntApp, ConfigProvider } from "@/ui-kit/eat";
import { antdLocaleMap } from "@/i18n/antd-locale";
import { QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { BrowserRouter } from "react-router";
import { AppRoutes } from "./routes";

export function App() {
  const [queryClient] = useState(() => createQueryClient());
  const { mode } = useThemeMode();
  const { i18n } = useTranslation();

  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider
        theme={getThemeConfig(mode)}
        locale={antdLocaleMap[i18n.language as "en" | "zh"]}
      >
        <AntApp>
          <BrowserRouter>
            <AppRoutes />
          </BrowserRouter>
        </AntApp>
      </ConfigProvider>
    </QueryClientProvider>
  );
}
```

### 4.2 AntD locale 映射：`src/i18n/antd-locale.ts`

```typescript
import type { Locale } from "antd/es/locale";
import enUS from "antd/locale/en_US";
import zhCN from "antd/locale/zh_CN";

export const antdLocaleMap: Record<string, Locale> = {
  en: enUS,
  zh: zhCN,
};
```

### 4.3 设计约束

- 删除 `<I18nProvider>` 包裹（旧自写 provider）。react-i18next 通过 `initReactI18next` 插件自动注入 context，不需要显式 Provider 组件
- `import "./i18n/config"` 必须在 App.tsx 最顶部，作为 side-effect import，确保 i18n 在首次 render 前就初始化完成
- ConfigProvider 的 `locale` prop 由 `useTranslation()` 的 `i18n.language` 驱动，语言变化时自动触发 ConfigProvider 重渲染
- AntD locale 包通过独立文件 `antd-locale.ts` 导入，不通过 eat barrel（因为 `antd/locale/zh_CN` 是子路径导入，Biome 的 barrel 规则不覆盖 locale 子包）

## 5. ui-kit/eat barrel 变更

无需为 AntD locale 包改 barrel。`antd/locale/zh_CN` 和 `antd/locale/en_US` 是独立子包入口，不在 Biome 的 `antd` 根导入限制范围内。`antd-locale.ts` 模块封装了这一层导入。

如果 Biome/dep-cruiser 规则在实施中报错，再在 `.dependency-cruiser.cjs` 里增加 `antd/locale` 的白名单。

## 6. 格式化工具重构（核心模块）

### 6.1 设计原则

格式化函数保持纯函数签名，接受 `locale` 参数，不在函数体内读取全局 i18next 单例。调用方在 React 层通过 `useTranslation().i18n.language` 获取 locale 并传入。

### 6.2 Intl 实例缓存

```typescript
// src/domain/money/intl-cache.ts
const cache = new Map<string, Intl.NumberFormat>();

export function getNumberFormat(locale: string, options: Intl.NumberFormatOptions): Intl.NumberFormat {
  const key = `${locale}:${JSON.stringify(options)}`;
  let fmt = cache.get(key);
  if (!fmt) {
    fmt = new Intl.NumberFormat(locale, options);
    cache.set(key, fmt);
  }
  return fmt;
}
```

### 6.3 `domain/money/format.ts` 改造

所有公开函数新增 `locale` 参数：

```typescript
export function formatAmount(amount: number, locale = "en"): string {
  const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
  const fmt = getNumberFormat(intlLocale, { maximumFractionDigits: 0 });
  // ... rest of formatting logic with "¥" prefix preserved
}
```

调用方改造示例：

```tsx
// Before
formatAmount(holding.marketValue)

// After
const { i18n } = useTranslation();
formatAmount(holding.marketValue, i18n.language)
```

### 6.4 `domain/ui/format.ts` 改造

同理所有函数加 `locale` 参数：

```typescript
export function formatCurrency(value: number, locale = "en", currency = "CNY"): string {
  const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
  return getNumberFormat(intlLocale, {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

export function formatDate(date: string | Date, locale = "en", format?: string): string {
  const intlLocale = locale === "zh" ? "zh-CN" : "en-US";
  const d = typeof date === "string" ? new Date(date) : date;
  if (format === "datetime") {
    return d.toLocaleString(intlLocale);
  }
  return d.toLocaleDateString(intlLocale);
}
```

### 6.5 约束

- locale 参数类型为 `string`（不限制 union），映射到 Intl locale tag 由函数内部完成
- 默认值 `"en"` 确保非 React 上下文（测试、Node 脚本）下也能工作
- `¥` 货币符号保持硬编码（PRD §4.3 明确不做货币符号切换）
- 缓存 Map 无容量上限（只有 2 个 locale x N 种 options 组合，实际缓存条目 < 10）

## 7. MainLayout 语言 Dropdown（核心模块）

### 7.1 menuFooterRender 改造

```tsx
// Inside MainLayout, menuFooterRender function body
const { t, i18n } = useTranslation();

const languageMenu = {
  items: [
    { key: "en", label: "English" },
    { key: "zh", label: "中文" },
  ],
  selectedKeys: [i18n.language],
  onClick: ({ key }: { key: string }) => i18n.changeLanguage(key),
};

// In JSX, between avatar block and help link:
<Dropdown menu={languageMenu} placement="topLeft">
  <Tooltip title={t("nav.switchLanguage")}>
    <Space style={{ cursor: "pointer" }}>
      <GlobalOutlined />
      <span>{i18n.language === "zh" ? "中文" : "EN"}</span>
    </Space>
  </Tooltip>
</Dropdown>
```

### 7.2 约束

- Dropdown 选项 label 硬写各自母语，不走 `t()`（同 PRD §5.5.1 Radio 规则）
- `selectedKeys` 绑定 `i18n.language`，保证与 Settings Radio 状态同源
- Tooltip 内容走 `t("nav.switchLanguage")`（en: "Switch language" / zh: "切换语言"）
- MainLayout 必须调用 `useTranslation()` 让自身对语言变化 reactive（sidebar menu 的 name 也要 i18n）

### 7.3 menuRoutes i18n

```typescript
// Before (hardcoded)
const menuRoutes = { path: "/", routes: [
  { path: "/dashboard", name: "Dashboard", icon: <DashboardOutlined /> },
  ...
]};

// After (inside component body, reactive to language)
const { t } = useTranslation();
const menuRoutes = useMemo(() => ({
  path: "/",
  routes: [
    { path: "/dashboard", name: t("nav.dashboard"), icon: <DashboardOutlined /> },
    { path: "/portfolio", name: t("nav.portfolio"), icon: <PieChartOutlined /> },
    { path: "/settings", name: t("nav.settings"), icon: <SettingOutlined /> },
  ],
}), [t]);
```

## 8. PreferencesTab 迁移

### 8.1 改造

- 删除 `import { useLocale } from "@/domain/i18n/provider"`
- 改为 `import { useTranslation } from "react-i18next"`
- Radio `value` 绑定 `i18n.language`
- Radio `onChange` 调用 `i18n.changeLanguage(e.target.value)`
- 所有硬编码中文 label 换成 `t("settings:preferencesLanguage")` 等

### 8.2 Radio 选项 label 不走 t()

```tsx
<Radio value="zh">中文</Radio>
<Radio value="en">English</Radio>
```

这两行保持硬写，因为选项展示的是目标语言的母语名称。

## 9. HelpPage 迁移

### 9.1 改造

```typescript
// Before
const { locale } = useLocale();

// After
const { i18n } = useTranslation();
const locale = i18n.language as "en" | "zh";
const content = useMemo(() => getHelpContent(locale), [locale]);
```

### 9.2 Help 模块位置

保持 `src/i18n/help/` 不动。文件结构不变：`{en,zh}.json` + `types.ts` + `index.ts`。`getHelpContent` 函数 API 不变。只是消费方从 `useLocale()` 换到 `useTranslation().i18n.language`。

### 9.3 Section ID 稳定性约束

`help/en.json` 和 `help/zh.json` 的 `sections` 数组必须满足：

- 长度相同
- 每个位置的 `section.id` 值完全一致
- 顺序一致

违反会导致 IntersectionObserver 和 deep link 失效。此约束由人工 review 保证，后续可加 vitest 校验。

## 10. 测试工具更新

### 10.1 `test/utils.tsx`

```tsx
import i18n from "i18next";
import { I18nProvider } from "react-i18next"; // 注意：这里不是 I18nextProvider
import { initReactI18next } from "react-i18next";
import { resources } from "@/i18n/config";

// Create a test-only i18n instance (isolated from app singleton)
const testI18n = i18n.createInstance();
testI18n.use(initReactI18next).init({
  resources,
  lng: "en",
  fallbackLng: "en",
  ns: ["common", "auth", "app", "settings"],
  defaultNS: "common",
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
});

export function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>
        <I18nProvider i18n={testI18n}>
          <ConfigProvider>
            <AntApp>{ui}</AntApp>
          </ConfigProvider>
        </I18nProvider>
      </QueryClientProvider>,
    ),
  };
}
```

### 10.2 测试策略

- **单元测试**：用上述 `renderWithProviders`，加载真实翻译。`t()` 返回翻译后的文案，测试断言用英文文案匹配
- **Mock 策略备选**：如果某些测试只关心结构不关心文案，可以 mock `react-i18next`：`vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (k: string) => k, i18n: { language: "en", changeLanguage: vi.fn() } }) }))`
- 两种策略按需选用，不强制统一

## 11. Key 命名约定

### 11.1 格式

`{namespace}:{section}.{element}.{qualifier}`

- namespace 用 colon 分隔（i18next 默认分隔符）
- section/element 用 dot 分隔
- 全部 camelCase

### 11.2 示例

```
common:nav.dashboard
common:nav.portfolio
common:action.save
common:action.cancel
common:status.loading
common:status.noData
auth:login.title
auth:login.emailLabel
auth:login.passwordLabel
auth:register.inviteCodeLabel
app:dashboard.topStrip.totalValue
app:portfolio.holdingTable.costPrice
app:decisionCard.confidence
settings:preferences.languageLabel
settings:llm.configForm.providerLabel
```

### 11.3 复数形式

涉及数量的 key 提供 `_one` / `_other` 后缀：

```json
{
  "portfolio": {
    "holdingCount_one": "{{count}} holding",
    "holdingCount_other": "{{count}} holdings"
  }
}
```

中文 JSON 可以两个后缀指向同一值（中文无语法复数）：

```json
{
  "portfolio": {
    "holdingCount_one": "{{count}} 个持仓",
    "holdingCount_other": "{{count}} 个持仓"
  }
}
```

### 11.4 Trans 组件使用边界

当翻译文本中需要嵌入 React 组件时使用 `<Trans>`：

```tsx
// Text with embedded link
<Trans i18nKey="common:disclaimer.text">
  All analysis is for reference only. See <a href="/help#disclaimer">details</a>.
</Trans>
```

对应 JSON：
```json
{
  "disclaimer": {
    "text": "All analysis is for reference only. See <1>details</1>."
  }
}
```

**边界**：仅在「翻译文本中间需要穿插组件」时用 Trans。纯文本一律用 `t()`。

## 12. 字符串迁移约定

### 12.1 迁移步骤（每个组件）

1. 在组件顶部加 `const { t, i18n } = useTranslation("namespace")`
2. 将硬编码中文替换为 `t("namespace:section.key")`
3. 在 en JSON 和 zh JSON 中分别添加对应 key
4. 如果组件调用 format helpers（formatAmount、formatDate 等），传入 `i18n.language` 作为 locale 参数
5. 删除不再需要的旧 `useLocale` import
6. 确保 Biome lint 通过

### 12.2 扫描命令

```bash
# Find all remaining Chinese in TSX (should shrink to zero post-migration)
rg -n '[\u4e00-\u9fff]' frontend/src --type tsx

# Find all remaining Chinese in TS (types are OK to keep)
rg -n '[\u4e00-\u9fff]' frontend/src --type ts
```

### 12.3 Form rule message 约定

Form rules 的 `message` 必须在组件 body 内计算（因为 `t()` 需要 React context）：

```tsx
function MyForm() {
  const { t } = useTranslation("auth");

  const rules = useMemo(() => ({
    email: [{ required: true, message: t("auth:login.emailRequired") }],
    password: [{ required: true, message: t("auth:login.passwordRequired") }],
  }), [t]);

  return <Form><Form.Item rules={rules.email}>...</Form.Item></Form>;
}
```

## 13. 废弃文件

实施完成后删除：

| 文件 | 原因 |
|------|------|
| `src/domain/i18n/provider.tsx` | 被 react-i18next 替代 |
| `src/domain/i18n/en.json` | 内容拆入 `src/i18n/locales/en/*.json` |
| `src/domain/i18n/zh.json` | 内容拆入 `src/i18n/locales/zh/*.json` |

删除后检查 `domain/i18n/` 目录是否为空，若空则删除目录。

## 14. 事后工具（非本期必须，预留脚本入口）

在 `package.json` scripts 中预留：

```json
{
  "i18n:check": "echo 'TODO: integrate i18next-cli --ci for missing/orphan key detection'"
}
```

实际集成 i18next-cli 在后续 PR 中完成。本期只保留 script 入口占位，不引入工具依赖。

## 15. dayjs locale 说明

当前项目不直接 import dayjs（仅有 `DayjsLike` 类型别名）。AntD DatePicker 的日历文案（月份名、星期名）由 `ConfigProvider.locale` 的内置数据控制，不需要单独调用 `dayjs.locale()`。

如果未来项目直接 import dayjs 做日期运算，则需要在 `i18n.on("languageChanged")` 回调中同步 `dayjs.locale()`。当前不需要。
