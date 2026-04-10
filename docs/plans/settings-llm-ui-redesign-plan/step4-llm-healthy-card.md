# Step 4: LLMHealthyCard 重构

**依赖：** Step 3 完成（ProviderCardLayout）+ Step 2 完成（i18n keys）
**Phase 3，可与 Step 5 并行**

## 任务目标

重构 `LLMHealthyCard.tsx`，使用 `ProviderCardLayout` 替换现有的扁平结构：
- 移除 `<Title>`、`<Tag color="success">`、`<Space>` 扁平结构
- Body：info grid（模型 / API Key / Base URL / 失败降级状态）+ Divider + Toggle 行
- Footer：`<LLMProbeButton />` + 编辑 `<Button>`
- 删除操作从 Footer 移出，由 `ProviderCardLayout` 的 Dropdown 接管

## 涉及文件

- 修改：`frontend/src/features/settings-llm/LLMHealthyCard.tsx`

## 设计依据

- PRD §LLMHealthyCard：三段式结构、Body 内容、Footer 内容、删除移入 Dropdown
- TRD §LLMHealthyCard：改动要点、fallbackText 计算、handleToggleFallback 逻辑保持不变
- TRD §Info Grid 规范：`gridTemplateColumns: "1fr 1fr"`、label 样式（11px uppercase）、value 样式（13px）、token 颜色

## 实施步骤

- [ ] **4.1** 修改 import，添加 `ProviderCardLayout`（相对路径）、移除不再需要的 `Tag`、`Popconfirm`；保留 `Switch`、`Divider`、`Button`、`App`、`Space`；保留 `useDeleteLLMSettings`、`useUpsertLLMSettings`

- [ ] **4.2** 保留 `handleToggleFallback` 和 `handleDelete` 逻辑完全不变

- [ ] **4.3** 构造 `bodyContent`：
  - Info grid：4 项（model / apiKeyHint / baseUrl（有则显示）/ fallbackText）
  - 参照 TRD §Info Grid 规范构造 grid div，items 数组按需过滤
  - `fallbackText` 根据 `config.fallbackToSystemDefaultOnFailure` 选用 `llm.healthyCard.fallbackOn/Off`
  - `<Divider style={{ margin: "14px 0" }} />`
  - Toggle 行：左侧文字区（`fallbackToggle` 标题 + 12px secondary `fallbackHint`）+ 右侧 `<Switch>`

- [ ] **4.4** 构造 `footerContent`：
  - `<Space>` 包含 `<LLMProbeButton />` 和 `<Button onClick={onEdit}>{t("llm.healthyCard.editButton")}</Button>`

- [ ] **4.5** 渲染 `<ProviderCardLayout>` 传入所有 props：
  - `providerType={config.providerType}`、`lastProbeAt={config.lastProbeAt}`、`healthStatus="healthy"`
  - `onDelete={handleDelete}`、`isDeleting={deleteMutation.isPending}`
  - `bodyContent={...}`、`footerContent={...}`
  - `data-testid="llm-healthy-card"`

- [ ] **4.6** 移除原有 `<Popconfirm>` + `<Button danger>` 组合（删除功能已转移到 ProviderCardLayout）

- [ ] **4.7** 运行 `cd frontend && pnpm lint:all`，修复全部错误

- [ ] **4.8** 提交
  - `git add frontend/src/features/settings-llm/LLMHealthyCard.tsx`
  - commit message: `feat(settings-llm): refactor LLMHealthyCard to three-section card layout`

## 验证标准

- `<Tag color="success">` 不再出现
- `<Text code>` 不再出现（info grid 中的 value 直接渲染字符串）
- 删除 `<Popconfirm>` 和 `<Button danger>` 从 Footer 中移除
- `handleToggleFallback`、`handleDelete` 逻辑代码与重构前完全相同
- `data-testid="llm-healthy-card"` 保留
- `pnpm lint:all` 通过
