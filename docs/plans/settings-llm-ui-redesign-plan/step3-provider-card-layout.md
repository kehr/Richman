# Step 3: ProviderCardLayout 内部共享组件

**依赖：** Step 1 完成（需要 `EllipsisOutlined` 已在 eat barrel、`formatRelativeTime` 已存在）
**Phase 2，可与 Step 6 和 Step 7 并行**

## 任务目标

新建 `ProviderCardLayout.tsx` 作为 HealthyCard / FailingCard 的三段式骨架共享组件：
- Header：Provider 字母 Badge + 名称 + 相对时间 + Badge status 状态点 + Dropdown（仅含删除项）
- Body：由调用方传入 `bodyContent`
- Footer：由调用方传入 `footerContent`（含编辑按钮）

该文件不导出到 feature barrel，仅在同目录组件内使用。

## 涉及文件

- 创建：`frontend/src/features/settings-llm/ProviderCardLayout.tsx`

## 设计依据

- PRD §组件设计 / LLMHealthyCard：Header 布局、Dropdown 模式、三段式结构
- TRD §ProviderCardLayout 内部组件：Props 接口、`PROVIDER_BADGE_STYLE` 常量、failing 状态颜色覆盖、Card 结构（`styles={{ body: { padding: 0 } }}`）、Header 布局代码、Dropdown items 构造、`BADGE_STATUS` 映射
- TRD §实施注意点：类型导入例外（`MenuProps`、`BadgeProps` 可直接从 antd 导入）、`theme.useToken()` 用法

## 实施步骤

- [ ] **3.1** 创建文件，按 TRD Props 接口定义 `ProviderCardLayoutProps`
  - 包含：`providerType`、`lastProbeAt`、`healthStatus`、`onEdit`（unused in layout, passed through）、`onDelete`、`isDeleting`、`bodyContent`、`footerContent`、`data-testid`
  - 注意：`onEdit` prop 在 layout 中不使用，edit button 在 `footerContent` 内由调用方自行放置；但接口中保留以便未来扩展

  **实际实现时注意**：ProviderCardLayout 不需要 `onEdit` prop，因为 edit 按钮在 `footerContent` 中由 HealthyCard/FailingCard 自己放置。TRD 接口里有 `onEdit` 是为了让调用方可选传入，但 layout 组件本身不使用它。验证 TRD 接口后按实际情况决定是否保留。

- [ ] **3.2** 实现 `PROVIDER_BADGE_STYLE` 常量（参照 TRD 精确 hex 值），failing 状态颜色覆盖逻辑

- [ ] **3.3** 实现 `BADGE_STATUS` 映射（`healthy→"success"`、`failing→"error"`、`unknown→"default"`）

- [ ] **3.4** 用 `Card styles={{ body: { padding: 0 } }}` 构造三段式结构
  - Header：Provider 字母 Badge（内联 style，32×32px）、providerLabel 函数（与现有 HealthyCard/FailingCard 中相同）、相对时间（`formatRelativeTime`）、`<Badge status={...} text={...} />`、Dropdown 触发按钮（`<Button type="text" icon={<EllipsisOutlined />} size="small" />`）
  - Body：`{bodyContent}`
  - Footer：`{footerContent}`

- [ ] **3.5** 构造 Dropdown items，包含带 `<Popconfirm>` 的删除项（state-controlled，`danger: true`）
  - i18n key 使用 `llm.healthyCard.deleteConfirm.*` 和 `llm.healthyCard.deleteMenuLabel`
  - `okButtonProps={{ danger: true, loading: isDeleting }}`

- [ ] **3.6** `theme.useToken()` 替换所有颜色硬编码（border、text、secondary text）
  - `theme` 从 `@/ui-kit/eat` 导入

- [ ] **3.7** 运行 `cd frontend && pnpm lint:all`，修复全部错误

- [ ] **3.8** 提交
  - `git add frontend/src/features/settings-llm/ProviderCardLayout.tsx`
  - commit message: `feat(settings-llm): add ProviderCardLayout shared internal component`

## 验证标准

- 文件存在，TypeScript 无错误
- `ProviderCardLayout` 未被加入 `features/settings-llm/index.ts`（不对外导出）
- failing 状态时字母 Badge 显示错误色（`#fff2f0` 背景）
- Dropdown 中有且仅有删除项，点击弹 Popconfirm
- `pnpm lint:all` 通过
