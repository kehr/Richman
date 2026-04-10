# Step 5: LLMFailingCard 重构

**依赖：** Step 3 完成（ProviderCardLayout）+ Step 2 完成（i18n keys）
**Phase 3，可与 Step 4 并行**

## 任务目标

重构 `LLMFailingCard.tsx`，使用 `ProviderCardLayout` 替换现有结构，差异点：
- `healthStatus="failing"` 使字母 Badge 自动变错误色
- Body 顶部插入 `<Alert type="error">` 展示 lastProbeError
- Info grid 只有 2 项（模型 / Base URL），无 API Key
- Toggle 行改为一行 secondary 文字展示 fallbackCopy
- Footer 测试按钮文案改为"重新测试"

## 涉及文件

- 修改：`frontend/src/features/settings-llm/LLMFailingCard.tsx`

## 设计依据

- PRD §LLMFailingCard：与 HealthyCard 的差异点清单
- TRD §LLMFailingCard：改动要点、Alert、Info grid 2 项、fallbackCopy、footer
- TRD §Info Grid 规范：grid 结构（同 HealthyCard），items 仅 model + baseUrl

## 实施步骤

- [ ] **5.1** 修改 import，添加 `ProviderCardLayout`（相对路径）；移除 `Title`、`Tag`、`Popconfirm`；保留 `Alert`、`App`、`Button`、`Space`、`Typography`；保留 `useDeleteLLMSettings`

- [ ] **5.2** 保留 `handleDelete` 和 `fallbackCopy` 逻辑完全不变

- [ ] **5.3** 构造 `bodyContent`：
  - `<Alert type="error" showIcon message={t("llm.failingCard.connectivityFailed")} description={config.lastProbeError ?? t("llm.failingCard.unknown")} style={{ marginBottom: 14 }} />`
  - Info grid：仅 2 项（model / baseUrl），参照 TRD §Info Grid 规范，baseUrl 无值时只显示 model
  - `<Divider style={{ margin: "14px 0" }} />`
  - `<Text type="secondary" style={{ fontSize: 12 }}>{fallbackCopy}</Text>`

- [ ] **5.4** 构造 `footerContent`：
  - `<Space>` 包含 `<LLMProbeButton label={t("llm.failingCard.retestButton")} />` 和 `<Button onClick={onEdit}>{t("llm.failingCard.editButton")}</Button>`

- [ ] **5.5** 渲染 `<ProviderCardLayout>`，传入：
  - `healthStatus="failing"`（触发错误色覆盖）
  - `onDelete={handleDelete}`、`isDeleting={deleteMutation.isPending}`
  - `data-testid="llm-failing-card"`

- [ ] **5.6** 移除原有 `<Popconfirm>` + `<Button danger>` 组合

- [ ] **5.7** 运行 `cd frontend && pnpm lint:all`，修复全部错误

- [ ] **5.8** 提交
  - `git add frontend/src/features/settings-llm/LLMFailingCard.tsx`
  - commit message: `feat(settings-llm): refactor LLMFailingCard to three-section card layout`

## 验证标准

- `<Tag color="error">` 不再出现
- `<Text code>` 不再出现
- Body 顶部有 `<Alert type="error">`
- Info grid 只有模型和 Base URL，无 API Key 行
- `handleDelete`、`fallbackCopy` 逻辑代码与重构前完全相同
- `data-testid="llm-failing-card"` 保留
- `pnpm lint:all` 通过
