# AI 配置页面 UI 重设计 TRD

参考 PRD：`docs/prds/settings-llm-ui-redesign-prd.md`

## 架构概览

所有改动均在 `frontend/src/features/settings-llm/` 内部，不影响外部接口。

新增文件：

```
features/settings-llm/
  utils/
    formatRelativeTime.ts       # 相对时间工具函数（新增）
  ProviderCardLayout.tsx        # 三段式 Card 内部共享组件（新增，不导出）
```

修改文件：

```
features/settings-llm/
  LLMHealthyCard.tsx
  LLMFailingCard.tsx
  LLMEmptyState.tsx
  LLMConfigForm.tsx
ui-kit/eat/index.ts
i18n/locales/zh/settings.json
i18n/locales/en/settings.json
```

## formatRelativeTime 函数

**文件：** `features/settings-llm/utils/formatRelativeTime.ts`

**签名：**

```ts
export function formatRelativeTime(
  date: string | Date | null | undefined,
  lang: string,
): string
```

**行为：**

- `date` 为 null / undefined 时返回 `"—"`
- 用 `Date.now() - new Date(date).getTime()` 计算秒级差值（正数 = 过去）
- 选取单位（秒/分/时/天/周/月/年），阈值如下：

| 差值范围 | 单位 | 除数 |
|---|---|---|
| < 60s | second | 1 |
| < 3600s | minute | 60 |
| < 86400s | hour | 3600 |
| < 604800s | day | 86400 |
| < 2592000s | week | 604800 |
| < 31536000s | month | 2592000 |
| >= 31536000s | year | 31536000 |

- 使用 `new Intl.RelativeTimeFormat(lang, { style: "long" }).format(-Math.round(diff / divisor), unit)` 生成字符串（负数表示过去）
- `lang` 传入 `"zh"` 时输出 "2 分钟前"，传入 `"en"` 时输出 "2 minutes ago"
- 调用方通过 `useTranslation` 的 `i18n.language` 获取当前语言传入

## ProviderCardLayout 内部组件

**文件：** `features/settings-llm/ProviderCardLayout.tsx`（不加入 `index.ts` barrel）

**Props 接口：**

```ts
interface ProviderCardLayoutProps {
  providerType: LLMSettingsDTO["providerType"];
  lastProbeAt: string | null | undefined;
  healthStatus: "healthy" | "failing" | "unknown";
  onEdit: () => void;
  onDelete: () => Promise<void>;
  isDeleting: boolean;
  bodyContent: ReactNode;
  footerContent: ReactNode;
  "data-testid"?: string;
}
```

**Provider 字母 Badge 渲染逻辑：**

```ts
const PROVIDER_BADGE_STYLE: Record<
  NonNullable<LLMSettingsDTO["providerType"]>,
  { bg: string; border: string; color: string; letter: string }
> = {
  claude:             { bg: "#f0f5ff", border: "#d6e4ff", color: "#2f54eb", letter: "C" },
  openai:             { bg: "#f6ffed", border: "#d9f7be", color: "#389e0d", letter: "O" },
  openai_compatible:  { bg: "#f9f0ff", border: "#efdbff", color: "#531dab", letter: "O" },
};
```

failing 状态时覆盖为错误色：`{ bg: "#fff2f0", border: "#ffccc7", color: "#cf1322" }`（letter 保持原值）。

**Badge 样式（内联 style，无 CSS 文件）：**

```ts
{
  width: 32, height: 32, borderRadius: 6,
  background: badge.bg, border: `1px solid ${badge.border}`,
  display: "flex", alignItems: "center", justifyContent: "center",
  fontSize: 13, fontWeight: 700, color: badge.color,
  flexShrink: 0,
}
```

**Card 结构：**

使用 `<Card styles={{ body: { padding: 0 } }}>` 关闭默认 padding，内部手动划分三区：

```tsx
<Card styles={{ body: { padding: 0 } }} data-testid={...}>
  {/* Header */}
  <div style={{ padding: "16px 20px", borderBottom: `1px solid ${token.colorBorderSecondary}` }}>
    ...
  </div>
  {/* Body */}
  <div style={{ padding: "16px 20px" }}>
    {bodyContent}
  </div>
  {/* Footer */}
  <div style={{ padding: "12px 20px", borderTop: `1px solid ${token.colorBorderSecondary}` }}>
    {footerContent}
  </div>
</Card>
```

`token` 通过 `const { token } = theme.useToken()` 获取（`theme` 从 `@/ui-kit/eat` 导入）。

**Header 布局：**

```tsx
<div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
  {/* 左侧 */}
  <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
    {/* Provider 字母 Badge */}
    <div style={badgeStyle}>{badge.letter}</div>
    <div>
      <div style={{ fontSize: 15, fontWeight: 600, color: token.colorText }}>
        {providerLabel(providerType)}
      </div>
      <div style={{ fontSize: 12, color: token.colorTextSecondary, marginTop: 1 }}>
        {t("llm.healthyCard.lastProbedAt", { time: formatRelativeTime(lastProbeAt, i18n.language) })}
      </div>
    </div>
  </div>
  {/* 右侧 */}
  <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
    <Badge status={badgeStatus} text={badgeText} />
    <Dropdown menu={{ items: dropdownItems }} trigger={["click"]}>
      <Button type="text" icon={<EllipsisOutlined />} size="small" />
    </Dropdown>
  </div>
</div>
```

**Dropdown items 构造：**

```ts
const dropdownItems: MenuProps["items"] = [
  {
    key: "delete",
    danger: true,
    label: (
      <Popconfirm
        title={t("llm.healthyCard.deleteConfirm.title")}
        description={t("llm.healthyCard.deleteConfirm.description")}
        okText={t("llm.healthyCard.deleteConfirm.ok")}
        cancelText={t("llm.healthyCard.deleteConfirm.cancel")}
        okButtonProps={{ danger: true, loading: isDeleting }}
        onConfirm={onDelete}
      >
        <span>{t("llm.healthyCard.deleteMenuLabel")}</span>
      </Popconfirm>
    ),
  },
];
```

`MenuProps` 从 `antd` 导入类型（类型导入可直接用 antd，无需经过 eat barrel）。

**healthStatus → Badge status 映射：**

```ts
const BADGE_STATUS: Record<string, BadgeProps["status"]> = {
  healthy: "success",
  failing: "error",
  unknown: "default",
};
```

`BadgeProps` 从 `antd` 导入类型。

## LLMHealthyCard

**改动要点：**

1. 删除原有 `<Title>`、`<Tag>`、`<Space>` 结构，改用 `<ProviderCardLayout>`
2. `bodyContent`：
   - Info grid（见下方 info grid 规范）：4 项：模型 / API Key / Base URL（有则显示）/ 失败降级状态
   - `<Divider style={{ margin: "14px 0" }} />`
   - Toggle 行：`<div style={{ display:"flex", alignItems:"flex-start", justifyContent:"space-between", gap:12 }}>`，左侧文字区（标题 + 12px secondary 描述），右侧 `<Switch>`
3. `footerContent`：`<Space>`，包含 `<LLMProbeButton />` 和编辑 `<Button>`

失败降级状态文字：

```ts
const fallbackText = config.fallbackToSystemDefaultOnFailure
  ? t("llm.healthyCard.fallbackOn")
  : t("llm.healthyCard.fallbackOff");
```

**`handleToggleFallback` 保持不变**（仅 UI 结构改变，逻辑不动）。

**删除 `<Popconfirm>` + `<Button danger>` 组合**（移入 ProviderCardLayout Dropdown）。

`handleDelete` 逻辑保持不变，通过 `onDelete` prop 传给 `ProviderCardLayout`，`isDeleting={deleteMutation.isPending}`。

## LLMFailingCard

**改动要点：**

1. 改用 `<ProviderCardLayout>` （`healthStatus="failing"` 时字母 Badge 自动变错误色）
2. `bodyContent`：
   - `<Alert type="error" showIcon message={...} description={config.lastProbeError ?? ...} />`，margin-bottom 14px
   - Info grid（只有 2 项）：模型 / Base URL（无 Base URL 时只显示模型）
   - `<Divider style={{ margin: "14px 0" }} />`
   - `<Text type="secondary" style={{ fontSize: 12 }}>{fallbackCopy}</Text>`
3. `footerContent`：`<Space>`，包含 `<LLMProbeButton label={t("llm.failingCard.retestButton")} />` 和编辑 `<Button>`

`handleDelete` 同 HealthyCard，通过 `onDelete` prop 传给 `ProviderCardLayout`。

## Info Grid 规范

两个 Card 共用的信息网格结构（在各自 bodyContent 内内联渲染，不提取为独立组件）：

```tsx
<div style={{
  display: "grid",
  gridTemplateColumns: "1fr 1fr",
  gap: "12px 24px",
  marginBottom: 14,
}}>
  {items.map(({ label, value, muted }) => (
    <div key={label} style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span style={{
        fontSize: 11,
        color: token.colorTextQuaternary,
        textTransform: "uppercase",
        letterSpacing: "0.4px",
      }}>
        {label}
      </span>
      <span style={{
        fontSize: 13,
        color: muted ? token.colorTextTertiary : token.colorText,
      }}>
        {value}
      </span>
    </div>
  ))}
</div>
```

`items` 是 `{ label: string; value: string; muted?: boolean }[]`，在 render 时按需过滤（如 baseUrl 为空则不加入数组）。

## LLMEmptyState

**改动要点：**

1. 删除 `<Empty>` 组件
2. 改为 `<Card>` 内居中布局：

```tsx
<Card data-testid="llm-empty-state">
  <div style={{ textAlign: "center", padding: "24px 20px" }}>
    <Bot size={32} color={token.colorTextQuaternary} style={{ marginBottom: 12 }} />
    <Title level={5} style={{ margin: "0 0 6px" }}>
      {t("llm.emptyState.title")}
    </Title>
    <Text type="secondary" style={{ display: "block", maxWidth: 360, margin: "0 auto 16px", fontSize: 13 }}>
      {t("llm.emptyState.description")}
    </Text>
    <Button type="primary" onClick={onAddProvider} data-testid="llm-add-provider-button">
      {t("llm.emptyState.addButton")}
    </Button>
  </div>
  {calloutNode}
</Card>
```

`Bot` 从 `lucide-react` 直接导入：`import { Bot } from "lucide-react";`

**callout 渲染逻辑：**

```ts
const calloutNode = (() => {
  if (!systemDefaultAvailable) return null;
  if (useSystemDefaultConsent) {
    return (
      <Alert
        type="info"
        showIcon
        message={t("llm.emptyState.callout.systemConsentGiven")}
        style={{ borderTop: "none", borderRadius: "0 0 8px 8px" }}
      />
    );
  }
  return (
    <Alert
      type="warning"
      showIcon
      message={t("llm.emptyState.callout.systemNoConsent")}
      style={{ borderTop: "none", borderRadius: "0 0 8px 8px" }}
    />
  );
})();
```

## LLMConfigForm

**Provider 类型字段改造：**

将 `<Select>` 替换为 `<Radio.Group>`（plain，不加 `optionType`）：

```tsx
<Form.Item name="providerType" label={t("llm.configForm.providerType")} rules={rules.providerType}>
  <Radio.Group>
    <Radio value="claude">{t("llm.configForm.providerOptions.claude")}</Radio>
    <Radio value="openai">{t("llm.configForm.providerOptions.openai")}</Radio>
    <Radio value="openai_compatible">{t("llm.configForm.providerOptions.openai_compatible")}</Radio>
  </Radio.Group>
</Form.Item>
```

`Radio.Group` 下方通过 Form.Item `help` prop 渲染当前 Provider 描述文字（响应 `providerType` watch 值）：

```ts
const providerHelpText: Record<LLMProviderType, string> = {
  claude: t("llm.configForm.providerHelp.claude"),
  openai: t("llm.configForm.providerHelp.openai"),
  openai_compatible: t("llm.configForm.providerHelp.openai_compatible"),
};
```

将 `help={providerHelpText[providerType]}` 放在 Form.Item 上（`providerType` 已由 `Form.useWatch` 获取）。

**表单分区：**

在 `<Form.Item name="model" ...>` 和 `<Form.Item name="fallbackToSystemDefaultOnFailure" ...>` 之间插入：

```tsx
<Divider style={{ margin: "8px 0 16px" }} />
```

`Divider` 已在 eat barrel，无需新增。

**其余逻辑全部保持不变：**

- `useEffect` reset 逻辑
- 校验规则（`rules` object）
- `handleOk` 提交逻辑
- `mode === "edit"` 时 apiKey 可留空
- `probe: true` 保存时触发连通性测试

## eat barrel 变更

`frontend/src/ui-kit/eat/index.ts` 在 `@ant-design/icons` 导出块新增：

```ts
EllipsisOutlined,
```

## i18n 完整变更

### 新增 key（zh/settings.json）

```json
"llm": {
  "healthyCard": {
    "lastProbedAt": "测试于 {{time}}",
    "fallbackOn": "已开启",
    "fallbackOff": "已关闭",
    "deleteMenuLabel": "删除 Provider 配置"
  },
  "emptyState": {
    "description": "配置你自己的 LLM Provider 以获得 AI 驱动的投资解读。未配置时分析走规则引擎。",
    "callout": {
      "systemConsentGiven": "当前已开启系统默认 AI Provider，分析将正常运行。",
      "systemNoConsent": "系统默认 Provider 可用，但未同意使用。当前分析走规则引擎。"
    }
  },
  "configForm": {
    "providerHelp": {
      "claude": "Claude API（Anthropic）",
      "openai": "OpenAI 官方 API",
      "openai_compatible": "Ollama / LM Studio / 自建 OpenAI 兼容服务"
    }
  }
}
```

### 新增 key（en/settings.json）

```json
"llm": {
  "healthyCard": {
    "lastProbedAt": "Tested {{time}}",
    "fallbackOn": "Enabled",
    "fallbackOff": "Disabled",
    "deleteMenuLabel": "Delete Provider"
  },
  "emptyState": {
    "description": "Configure your own LLM Provider for AI-powered investment insights. Without one, analysis uses the rule engine.",
    "callout": {
      "systemConsentGiven": "System default AI Provider is active. Analysis will continue normally.",
      "systemNoConsent": "System default Provider is available but consent not given. Analysis uses rule engine."
    }
  },
  "configForm": {
    "providerHelp": {
      "claude": "Claude API (Anthropic)",
      "openai": "OpenAI official API",
      "openai_compatible": "Ollama / LM Studio / self-hosted OpenAI-compatible service"
    }
  }
}
```

### 移除 key

以下 key 在重构后不再使用，从两个 locale 文件中删除：

| 文件路径 | 移除的 key |
|---|---|
| `llm.healthyCard.lastProbed` | 替换为 `lastProbedAt`（含插值） |
| `llm.failingCard.lastProbed` | Header 相对时间由 ProviderCardLayout 统一渲染 |
| `llm.emptyState.callout.noSystem` | 无系统默认时不渲染 callout |

### 保留 key（不变）

其余所有现有 key 保持不变，包括：`healthyCard.healthy/fallbackToggle/fallbackHint/fallbackUpdated/fallbackUpdateError/probeButton/editButton/deleteButton/deleteSuccess/deleteError/deleteConfirm.*`、`failingCard.*`、`configForm.*`（除新增部分）、`probeButton.*`。

## 实施注意点

1. **`ProviderCardLayout` 不导出**：不加入 `features/settings-llm/index.ts`，仅在同目录的 HealthyCard/FailingCard 内 import。

2. **`formatRelativeTime` 不导出到 feature barrel**：同样仅在 HealthyCard/FailingCard/ProviderCardLayout 内 import，工具函数是实现细节。

3. **类型导入例外**：`MenuProps`、`BadgeProps` 等纯类型可直接从 `antd` import（`import type { MenuProps } from "antd"`），不需要经过 eat barrel（eat barrel 规范针对运行时组件，不针对类型）。

4. **`theme.useToken()` 使用**：`theme` 已在 eat barrel 导出，在组件顶层调用 `const { token } = theme.useToken()`，用 token 替代所有硬编码颜色值（`colorBorderSecondary`、`colorTextQuaternary`、`colorTextTertiary`、`colorText`、`colorTextSecondary`）。

5. **Biome 行宽**：行宽限制 100 字符，内联 style 对象如果超长需换行。
