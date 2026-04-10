# 前端编码规范

## 技术栈

- Vite 6（ESM 原生 dev server） + React 19 + React Router v7
- Ant Design 6 + @ant-design/pro-components（通过 ui-kit/eat barrel）
- TanStack Query v5（服务端状态） + 原生 React hooks（客户端状态）
- 自研 JSON i18n provider（`domain/i18n`）、pnpm + Vite env（`.env.[mode]`）
- Biome（lint + format）+ dependency-cruiser（架构边界检查）
- Vitest + React Testing Library + MSW（测试，参考 `docs/standards/testing.md`）


## 目录结构：Pages + Features 双层架构

```
frontend/src/
  config/
    routes.tsx              # 路由配置（声明式 RouteConfig[]）
    theme.ts                # Ant Design ThemeConfig（亮色/暗色）
  pages/                    # 页面组装层（纯组合，不含业务逻辑）
    dashboard/
      DashboardPage.tsx
    portfolio/
      PortfolioListPage.tsx
      PortfolioEditPage.tsx
    analysis/
      AnalysisPage.tsx
    decision-card/
      DecisionCardListPage.tsx
      DecisionCardDetailPage.tsx
    notification/
      NotificationSettingsPage.tsx
    auth/
      LoginPage.tsx
      RegisterPage.tsx
    settings/
      SettingsPage.tsx
  features/                 # 业务模块层（自包含）
    dashboard/
      api.ts
      useStats.ts
      index.ts
    portfolio/
      api.ts
      usePortfolio.ts
      useHoldings.ts
      index.ts
    analysis/
      api.ts
      useAnalysis.ts
      index.ts
    decision-card/
      api.ts
      useDecisionCard.ts
      DecisionCardView.tsx  # 内部组件（不通过 barrel 导出）
      index.ts
    notification/
      api.ts
      useNotification.ts
      index.ts
    auth/
      api.ts
      useAuth.ts
      index.ts
  domain/                   # 跨模块基础设施
    http/                   # API client（request 函数）
    auth/                   # 认证（token 存储、useCurrentUser、AuthGuard）
    i18n/                   # i18n 配置和 locale 文件
    ui/                     # 通用 UI 工具（格式化、图标等）
  layouts/
    MainLayout.tsx          # 主布局（ProLayout）
  ui-kit/
    eat/                    # Ant Design barrel 导出
      index.ts              # 所有 antd/pro/icons 组件的统一出口
    svg/                    # SVG 组件
```


## 依赖流向

```
App.tsx -> config/ -> pages/ -> features/ -> domain/ -> ui-kit/eat
```

**严格分层约束：**

| 层 | 职责 | 可依赖 | 不可依赖 |
|---|------|--------|---------|
| config/ | 路由和主题配置（纯声明） | pages（引用）、ui-kit/eat | features、domain |
| pages/ | 页面组装 | features/*/index（barrel）、domain、ui-kit/eat、layouts | feature 内部文件 |
| features/ | 自包含业务模块 | domain、ui-kit/eat | **其他 features**、pages |
| domain/ | 跨模块基础设施 | ui-kit/eat、第三方库 | features、pages |
| layouts/ | 页面布局 | config、ui-kit/eat、React Router | features、domain、pages |
| ui-kit/ | Ant Design 封装 | antd 包 | 任何业务代码 |

**核心约束：**
- features 之间互相隔离，不可跨 feature 导入
- pages 只通过 barrel（index.ts）消费 features
- domain 不依赖 features 和 pages
- ui-kit 不依赖任何业务代码


## Feature 模块创建模式

每个 feature 模块包含三个核心文件：

```typescript
// features/portfolio/api.ts -- API 函数 + DTO 类型
export interface HoldingDto {
	holdingId: number;
	assetName: string;
	costPrice: number;
	positionRatio: number;
}

export function fetchHoldings() {
	return request<{ data: HoldingDto[] }>("/api/v1/holdings");
}

export function createHolding(data: CreateHoldingInput) {
	return request<{ data: HoldingDto }>("/api/v1/holdings", {
		method: "POST",
		body: JSON.stringify(data),
	});
}

// features/portfolio/usePortfolio.ts -- TanStack Query hooks
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchHoldings, createHolding } from "./api.js";

export function useHoldings() {
	return useQuery({
		queryKey: ["holdings"],
		queryFn: fetchHoldings,
	});
}

export function useCreateHolding() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: createHolding,
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ["holdings"] });
		},
	});
}

// features/portfolio/index.ts -- Barrel 导出（仅公开 API）
export { useHoldings, useCreateHolding } from "./usePortfolio.js";
export type { HoldingDto } from "./api.js";
```


## UI 组件导入规则

**所有 Ant Design 组件必须通过 ui-kit/eat 导入。**

```typescript
// 正确
import { Card, ProTable, Tag, Space } from "@/ui-kit/eat";

// 错误 -- Biome noRestrictedImports 会报错
import { Card } from "antd";
import { ProTable } from "@ant-design/pro-components";
import { UserOutlined } from "@ant-design/icons";
```

ui-kit/eat/index.ts 是唯一允许直接导入 antd 包的文件。


## 图标使用规范

项目采用双图标库方案：

| 库 | 适用场景 | 导入方式 |
|----|---------|---------|
| `@ant-design/icons` | 通用 UI、导航、操作按钮、状态图标 | 必须通过 `@/ui-kit/eat` |
| `lucide-react` | AI 相关、补充性图标（Sparkles、BrainCircuit、Zap 等） | 直接导入，无需经过 eat |

```typescript
// @ant-design/icons -- 通过 eat
import { UserOutlined, BellOutlined } from "@/ui-kit/eat";

// lucide-react -- 直接导入
import { Sparkles, BrainCircuit, Zap } from "lucide-react";
```

**尺寸对齐：** lucide-react 默认 size 为 24，在 Ant Design 菜单/导航场景中使用 `size={14}` 与 AntD icon 尺寸保持一致。其他场景按实际视觉效果调整。

**选型原则：**
- 优先使用 `@ant-design/icons`（已有、风格统一）
- AntD 没有合适图标时（尤其是 AI 相关语义）使用 `lucide-react`
- 不引入第三个图标库


## 表单组件选型规则

| 场景 | 使用 | 禁止 |
|------|-----|------|
| 枚举型单选（选项 <= 5） | `Radio.Group` + 普通 `Radio`（默认圆形样式） | `Select`、`Radio.Button`、`Radio.Group optionType="button"` |
| 枚举型单选（选项 > 5） | `Select` | - |
| 多选 | `Checkbox.Group` | - |
| 数值输入 | `InputNumber` | `Input` |

**Radio.Group 默认原则：**
- 枚举型单选统一用 `Radio.Group` + 普通 `Radio`（默认圆形样式）
- 禁止使用 `Radio.Button`、`optionType="button"`、`buttonStyle="solid"`
- 只在产品设计稿明确要求按钮样式时才使用 button 变体
- 选项不超过 5 个且语义清晰时优先 Radio 而非 Select（减少交互层级）


**Badge 状态点原则：**
- 可枚举的状态值（方向、健康状态、连接状态等）统一用 `<Badge status={...} text={label} />`，不用带颜色的 `Tag`
- Tag 只用于非状态语义的标记（分类、过滤标签、用户自定义标签等）
- status 映射示例（方向类）：bullish/upward → "success"，bearish/downward → "error"，neutral → "default"
- status 映射示例（健康类）：healthy → "success"，degraded → "warning"，failed → "error"，unknown → "default"


## Pro 组件优先规则

| 场景 | 使用 | 备选 |
|------|-----|------|
| 卡片容器 | Card（eat，默认 borderless） | - |
| 卡片网格 | Card + Row/Col | - |
| 统计指标 | StatisticCard | Card + Statistic |
| 描述列表 | ProDescriptions（columns 模式） | Descriptions |
| 数据表格 | ProTable | Table |
| 分段切换 | Segmented | Tabs |
| 页面布局 | ProLayout | Layout + Sider + Menu |

**Card 默认原则：**
- eat/Card 默认 `variant="borderless"`（首选）
- 网格布局用 Row/Col + Card，不用 ProCard 的 colSpan/gutter
- 仅在需要复杂效果时用 ProCard（ProCard.Divider、ProCard.Group）


## 页面文件命名

| 类型 | 规则 | 示例 |
|------|------|------|
| 列表页 | XxxListPage.tsx | `PortfolioListPage.tsx` |
| 详情页 | XxxDetailPage.tsx | `DecisionCardDetailPage.tsx` |
| 编辑页 | XxxEditPage.tsx | `PortfolioEditPage.tsx` |
| 创建页 | XxxCreatePage.tsx | `HoldingCreatePage.tsx` |
| 设置页 | XxxSettingsPage.tsx | `NotificationSettingsPage.tsx` |


## 状态管理

| 类型 | 方案 | 说明 |
|------|------|------|
| 服务端状态 | TanStack Query v5 | 缓存、重试、后台刷新 |
| 客户端状态 | React hooks | useState、useReducer、Context |
| 持久化客户端状态 | useLocalStorage（domain/storage） | 跨会话存活的 UI 状态 |
| 全局状态 | 不用 Redux/Zustand | 除非复杂度明确需要 |

**TanStack Query 配置：**
- 默认缓存时间：30 秒
- 默认重试：1 次
- 错误处理：全局 QueryCache/MutationCache 捕获，显示 toast


## 客户端持久化存储

所有 localStorage 读写必须通过 `domain/storage/` 的两层抽象，**禁止在组件、hook、feature 模块中直接调用 `localStorage`**。

**两层结构：**

```
domain/storage/
  local-storage.ts    # 原语层：StorageKeys 注册表 + 安全读写删函数
  use-local-storage.ts  # Hook 层：React 组件专用
```

**Key 注册表（local-storage.ts）**

所有 key 集中定义在 `StorageKeys` 常量对象，不允许在业务代码中出现字符串字面量 key：

```typescript
// 正确
import { StorageKeys } from "@/domain/storage/local-storage";
useLocalStorage(StorageKeys.lastAnalysisTaskId, null);

// 错误
localStorage.getItem("richman_last_task_id");
useLocalStorage("richman_last_task_id", null);
```

新增持久化字段时，先在 `StorageKeys` 注册，再使用。

**React 组件 / Hook 中使用 `useLocalStorage`**

接口与 `useState` 完全相同，第三个返回值是删除函数：

```typescript
import { StorageKeys } from "@/domain/storage/local-storage";
import { useLocalStorage } from "@/domain/storage/use-local-storage";

// 替代 useState + 手动读写 localStorage
const [taskId, setTaskId] = useLocalStorage<string | null>(
  StorageKeys.lastAnalysisTaskId,
  null,
);

// 清除 key（重置为 initialValue）
const [dismissed, setDismissed, clearDismissed] = useLocalStorage<boolean>(
  StorageKeys.onboardingNudgeDismissed,
  false,
);
```

**非组件上下文（mutation 回调、HTTP 拦截器等）使用原语函数**

当代码不在 React 渲染树内（如 `useMutation.onSuccess`、`domain/auth/`），Hook 无法使用，改用低层原语：

```typescript
import { StorageKeys, storageGet, storageRemove, storageSet } from "@/domain/storage/local-storage";

// mutation 回调中清除 flag
storageRemove(StorageKeys.onboardingNudgeDismissed);

// HTTP 拦截器中读取 token
const token = storageGet<string>(StorageKeys.authToken);
```

**判断用哪一层的规则：**

| 调用上下文 | 使用 |
|---|---|
| React 组件、自定义 hook | `useLocalStorage` |
| TanStack Query mutation/query 回调 | `storageGet / storageSet / storageRemove` |
| domain/auth、domain/http 等非组件模块 | `storageGet / storageSet / storageRemove` |

**值的序列化**

原语层统一用 `JSON.stringify / JSON.parse`，因此 `T` 必须是 JSON 兼容类型（string、number、boolean、object、array）。不可存储 `Date`、`Map`、`Set`、`undefined` 等非 JSON 类型——需要时在调用方做转换。

**存储不可用的处理**

隐私模式或配额溢出时，读操作返回 `initialValue`，写/删操作静默忽略，不抛异常。调用方无需额外防御。


## 样式

- 仅使用 Ant Design v6 CSS-in-JS
- 通过 ConfigProvider theme tokens 定制
- 不使用 Tailwind / UnoCSS / CSS Modules
- 主题切换通过 ConfigProvider algorithm（defaultAlgorithm / darkAlgorithm）


## API Client

所有请求通过 domain/http/ 的 request() 函数：

```typescript
// domain/http/client.ts
export async function request<T>(url: string, options?: RequestInit): Promise<T> {
	const response = await fetch(`${API_BASE}${url}`, {
		headers: {
			"Content-Type": "application/json",
			...getAuthHeaders(),
		},
		...options,
	});
	if (!response.ok) {
		throw new ApiError(response);
	}
	return response.json();
}
```

ProTable 的 request prop 直接对接 feature API：

```typescript
<ProTable
	request={async (params) => {
		const res = await fetchHoldings(params);
		return { data: res.data, total: res.pagination.total, success: true };
	}}
/>
```


## 代码风格（Biome）

| 项 | 值 |
|---|---|
| 缩进 | Tab |
| 行宽 | 100 字符 |
| 引号 | 双引号 `"` |
| 分号 | 始终使用 |
| import 排序 | Biome 自动管理 |


## 架构边界检查

dependency-cruiser 规则（.dependency-cruiser.web.cjs）：
- features 之间隔离
- domain 不依赖 features/pages
- ui-kit 不依赖业务代码
- layouts 限制依赖范围
- config 是纯声明层
