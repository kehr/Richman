# AI 配置页面 UI 重设计 PRD

## 背景与目标

### 背景

当前 AI 配置（settings-llm）页面功能已完整，但视觉设计存在以下问题：

- 信息展示使用 `<Text code>` 标签，视觉风格偏向开发者工具，与投资工具的专业调性不符
- 状态标签使用带颜色的 `<Tag>`，违反前端规范（状态值应用 Badge）
- 删除操作与编辑操作平级并排，危险操作过于突出
- 空状态使用 AntD 默认通用图案（小鸟），与 AI 语义无关
- 表单 Provider 类型选择使用 Select 下拉，交互层级多余（选项仅 3 个）
- 时间展示为完整 datetime 字符串，可读性低

### 目标

在不改变任何功能逻辑的前提下，重构 settings-llm feature 的所有 UI 组件，对齐：

1. 项目前端规范（Badge 状态点、Radio.Group 枚举选择、lucide-react AI 图标）
2. 行业设置页最佳实践（Stripe / Vercel / Linear 的 configured-service-card 模式）
3. 信息视觉层次：主要信息 > 次要信息 > 辅助信息

## 范围

本次改造涉及以下组件（全部在 `frontend/src/features/settings-llm/` 内）：

- `LLMHealthyCard.tsx`
- `LLMFailingCard.tsx`
- `LLMEmptyState.tsx`
- `LLMConfigForm.tsx`

新增：

- `frontend/src/features/settings-llm/utils/formatRelativeTime.ts`（相对时间工具函数）

修改：

- `frontend/src/ui-kit/eat/index.ts`（新增 `EllipsisOutlined` 导出）
- `frontend/src/i18n/locales/zh/settings.json` 和 `en/settings.json`（新增 i18n key）

不在本次范围：

- API / hooks / types 不变
- 后端不变
- LLMSection（容器）不变
- LLMProbeButton 不变

## 状态空间

settings-llm 页面共 3 种互斥状态，对应 3 个 UI 变体：

| 状态 | 条件 | 渲染组件 |
|---|---|---|
| 未配置 | `!configured` | LLMEmptyState |
| 已配置 + 健康 | `configured && healthStatus !== "failing"` | LLMHealthyCard |
| 已配置 + 失效 | `configured && healthStatus === "failing"` | LLMFailingCard |

切换逻辑在 LLMSection 中，本次不改变。

## 视觉规范

### 颜色语义

健康/失效状态严格遵守 Badge status 映射：

- healthy → `<Badge status="success" />`（绿色）
- failing → `<Badge status="error" />`（红色）
- unknown → `<Badge status="default" />`（灰色）

不使用 `<Tag color="success/error">`。

### Provider 字母 Badge

Header 左侧的 Provider 标识使用带颜色背景的字母 Badge（自定义 styled div），按 Provider 类型区分颜色：

| Provider | 背景色 | 文字色 | 字母 |
|---|---|---|---|
| claude | `#f0f5ff` + border `#d6e4ff` | `#2f54eb` | C |
| openai | `#f6ffed` + border `#d9f7be` | `#389e0d` | O |
| openai_compatible | `#f9f0ff` + border `#efdbff` | `#531dab` | O |

尺寸：32×32px，border-radius: 6px，字体 13px 加粗。

### 时间显示

所有 "最后测试时间" 均展示相对时间（"2 分钟前"、"1 小时前"、"3 天前"），不展示完整 datetime 字符串。

使用工具函数 `formatRelativeTime(date, lang)` 基于原生 `Intl.RelativeTimeFormat` 实现，无需新增依赖。

### AI Provider 空状态图标

使用 `lucide-react` 的 `Bot` 图标（size=32），直接导入，不经过 eat barrel。

## 组件设计

### LLMHealthyCard

三段式结构（Header / Body / Footer）：

**Header**（带 border-bottom）

- 左：Provider 字母 Badge + Provider 名称（font-weight 600）+ 相对时间（secondary 色，12px）
- 右：`<Badge status="success" text={t("llm.healthyCard.healthy")} />` + `<Dropdown>` 触发按钮（`<EllipsisOutlined />`）

Dropdown 菜单内容（items）：

```
- 删除 Provider 配置（danger 红色，点击触发 Popconfirm）
```

**Body**

- 2 列 info grid（CSS Grid，gap 12px 24px）：
  - 列 1：`模型`（label）/ `{config.model}`（value）
  - 列 2：`API Key`（label）/ `{config.apiKeyHint}`（value，muted 色）
  - 列 1 第 2 行：`Base URL`（label）/ `{config.baseUrl}`（value）（无 baseUrl 时不渲染该行）
  - 列 2 第 2 行：`失败降级`（label）/ 文字状态（"已开启"/"已关闭"）
- label 样式：11px、灰色（`#8c8c8c`）、全大写、letter-spacing 0.4px
- value 样式：13px、正常字重，去掉 `<Text code>` 包裹
- Divider
- Toggle 行：左侧文字区（标题 + 描述 12px secondary）+ 右侧 `<Switch>`

**Footer**（带 border-top）

- `<LLMProbeButton />`（已有组件，保持不变）
- `<Button>` 编辑

删除按钮从 Footer 移出，进入 Dropdown。

### LLMFailingCard

与 LLMHealthyCard 共享三段式结构，差异点：

- Header 的字母 Badge 使用错误色（背景 `#fff2f0`，边框 `#ffccc7`，文字 `#cf1322`）
- Header 的状态改为 `<Badge status="error" text={t("llm.failingCard.failing")} />`
- Body 顶部插入 `<Alert type="error" showIcon>` 展示 lastProbeError
- Info grid 只展示 `模型` / `Base URL`（去掉 API Key hint）
- 无 Toggle 行，改为一行 secondary 文字展示降级说明（fallbackCopy）
- Footer 同 HealthyCard，测试按钮文案改为"重新测试"

**共享组件提取**

提取内部组件 `ProviderCardLayout`（不通过 barrel 导出），承载三段式 Card 框架和 Provider 字母 Badge 逻辑，Healthy/Failing 组件传入 body 和 footer 内容。

### LLMEmptyState

去掉 `<Empty>` 组件，改为居中布局：

```
Bot 图标（lucide-react，size=32，color="#8c8c8c"，margin-bottom 12px）
标题：尚未配置 AI Provider（Title level=5）
描述：配置你自己的 LLM Provider 以获得 AI 驱动的投资解读。未配置时分析走规则引擎。（Text secondary，最大宽度 360px）
CTA：添加 LLM Provider（Button type="primary"，margin-top 16px）

callout（条件渲染，margin-top 16px）：
  - 有系统默认 + 已同意 → Alert type="info"，展示系统默认可用提示
  - 有系统默认 + 未同意 → Alert type="warning"，提示同意后可用
  - 无系统默认 → 不渲染 callout
```

### LLMConfigForm

**Provider 类型**：`<Select>` 改为 `<Radio.Group>`（plain，默认圆形样式，不加 optionType="button"）。

Radio.Group 下方增加 `<Form.Item help={...}>` 展示当前 Provider 的一行描述文字：

| 选中值 | help 文字 |
|---|---|
| claude | Claude API（Anthropic）|
| openai | OpenAI 官方 API |
| openai_compatible | Ollama / LM Studio / 自建 OpenAI 兼容服务 |

**表单分区**：增加一条 `<Divider />` 分隔凭证区和行为设置区：

- 上段（凭证）：Provider 类型 / Base URL / API Key / 模型
- 下段（行为）：失败自动降级 Toggle

其余逻辑不变（校验规则、edit 模式 API Key 可留空、probe: true 保存逻辑）。

## eat barrel 变更

`frontend/src/ui-kit/eat/index.ts` 新增一行导出：

```ts
EllipsisOutlined,
```

（来源：`@ant-design/icons`，已有该 icon，只需加入 barrel export）

## i18n 变更

中英文两个 settings namespace 文件均需新增以下 key：

```json
"llm": {
  "healthyCard": {
    "lastProbedAt": "测试于 {{time}}",
    "fallbackOn": "已开启",
    "fallbackOff": "已关闭"
  },
  "emptyState": {
    "callout": {
      "systemConsentGiven": "当前已开启系统默认 AI Provider，分析将正常运行。",
      "systemNoConsent": "系统默认 Provider 可用，但未同意使用。当前分析走规则引擎。"
    }
  }
}
```

移除：原 `emptyState.callout.noSystem`（未配置且无系统默认时不渲染 callout，无需文案）。

## 替代路径与决策记录

**Q: 为什么不用 ProDescriptions 展示 info grid？**

ProDescriptions 适合数据展示页面，引入会带来额外的行距和 label 样式约束，不如轻量 CSS Grid 灵活。本次信息密度低（4 项），用自定义 grid 实现更可控。

**Q: 为什么 Provider 类型选 Radio.Group 而不是 Segmented？**

前端规范明确：枚举型单选（选项 ≤ 5）使用 Radio.Group 默认圆形样式。Segmented 用于「分段视图切换」场景（如 list/grid 切换），语义不同。

**Q: 为什么删除移入 Dropdown 而不是放在 Footer 末尾？**

行业最佳实践（Stripe、Vercel、GitHub）均将破坏性低频操作从主操作区隔离。删除是低频操作，与「编辑」平级会造成误操作风险。Dropdown 保留可访问性的同时降低视觉权重。

**Q: 为什么不用 date-fns 实现相对时间？**

项目无时间处理库。原生 `Intl.RelativeTimeFormat` 已支持 zh/en，实现相对时间展示足够，不引入新依赖。
