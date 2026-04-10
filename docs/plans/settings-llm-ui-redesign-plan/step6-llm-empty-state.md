# Step 6: LLMEmptyState 重构

**依赖：** Step 2 完成（i18n keys，需要 `llm.emptyState.description` 和更新后的 callout keys）
**Phase 2，可与 Step 3 和 Step 7 并行**

## 任务目标

重构 `LLMEmptyState.tsx`，移除 AntD `<Empty>` 组件（小鸟图案），改为居中布局：
- `Bot` 图标（lucide-react，直接导入）替换 Empty 图标
- 独立的标题、描述文字、CTA 按钮
- 条件渲染 callout（Alert type="info/warning"，或不渲染）

## 涉及文件

- 修改：`frontend/src/features/settings-llm/LLMEmptyState.tsx`

## 设计依据

- PRD §LLMEmptyState：居中布局结构、Bot 图标、callout 三分支逻辑
- TRD §LLMEmptyState：改动要点、`Bot` 导入方式、callout 渲染逻辑（IIFE 模式）、Alert 样式（`borderTop: "none", borderRadius: "0 0 8px 8px"`）

## 实施步骤

- [ ] **6.1** 修改 import：
  - 移除 `Empty`（来自 eat barrel）
  - 新增 `Alert` 导入（已在 eat barrel）
  - 新增 `import { Bot } from "lucide-react"`（直接导入，不经过 eat barrel）
  - 保留 `Button`、`Card`、`Typography` 导入

- [ ] **6.2** 保留 `LLMEmptyStateProps` 接口不变（`systemDefaultAvailable`、`useSystemDefaultConsent`、`onAddProvider`）

- [ ] **6.3** 重写 callout 逻辑：
  - 无系统默认（`!systemDefaultAvailable`）→ `null`（不渲染，无需文案）
  - 有系统默认且已同意 → `<Alert type="info" ...>`，message = `t("llm.emptyState.callout.systemConsentGiven")`
  - 有系统默认未同意 → `<Alert type="warning" ...>`，message = `t("llm.emptyState.callout.systemNoConsent")`
  - Alert style：`{ borderTop: "none", borderRadius: "0 0 8px 8px" }`（与 Card border 融合）

- [ ] **6.4** 重写渲染结构：
  - `<Card data-testid="llm-empty-state">`（不关闭 body padding）
  - 居中 div，内含：Bot 图标（size=32，color=token.colorTextQuaternary，marginBottom 12px）、Title level=5、Text type="secondary"（maxWidth 360px，`t("llm.emptyState.description")`）、Button type="primary" CTA
  - Card 内 callout 在居中 div 外部追加

- [ ] **6.5** 用 `theme.useToken()` 获取 token（`theme` 从 `@/ui-kit/eat`），Bot 图标 color 使用 `token.colorTextQuaternary`

- [ ] **6.6** 运行 `cd frontend && pnpm lint:all`，修复全部错误

- [ ] **6.7** 提交
  - `git add frontend/src/features/settings-llm/LLMEmptyState.tsx`
  - commit message: `feat(settings-llm): refactor LLMEmptyState with Bot icon and conditional callout`

## 验证标准

- `<Empty>` 不再出现
- `Bot` 图标来自 `lucide-react` 直接导入
- `systemDefaultAvailable=false` 时 callout 不渲染
- `systemDefaultAvailable=true && useSystemDefaultConsent=true` 时渲染 `type="info"` Alert
- `systemDefaultAvailable=true && useSystemDefaultConsent=false` 时渲染 `type="warning"` Alert
- `data-testid="llm-empty-state"` 和 `data-testid="llm-add-provider-button"` 保留
- `pnpm lint:all` 通过
