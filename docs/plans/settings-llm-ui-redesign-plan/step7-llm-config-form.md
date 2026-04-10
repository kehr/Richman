# Step 7: LLMConfigForm 重构

**依赖：** Step 2 完成（i18n keys，需要 `llm.configForm.providerHelp.*`）
**Phase 2，可与 Step 3 和 Step 6 并行**

## 任务目标

重构 `LLMConfigForm.tsx` 中的 Provider 类型选择和表单布局：
- 将 `<Select>` 替换为 `<Radio.Group>`（plain 圆形样式，不加 `optionType`）
- Radio.Group 下方动态显示当前 Provider 的描述文字（`Form.Item help` prop）
- 在模型字段和降级 Toggle 之间插入 `<Divider>` 分隔凭证区和行为区

## 涉及文件

- 修改：`frontend/src/features/settings-llm/LLMConfigForm.tsx`

## 设计依据

- PRD §LLMConfigForm：Radio.Group 替代 Select、help 描述文字、Divider 分区
- TRD §LLMConfigForm：`Radio.Group` 渲染代码、`providerHelpText` 映射、`Form.Item help` 用法、Divider 位置（model 和 fallback item 之间）、其余逻辑保持不变

## 实施步骤

- [ ] **7.1** 修改 import：
  - 移除 `Select` 导入
  - 新增 `Radio`、`Divider` 导入（均已在 eat barrel，验证后加入 import）
  - 保留其余所有 import 不变

- [ ] **7.2** 移除 `providerOptions` useMemo（`<Select>` 的 options 数组）

- [ ] **7.3** 将 Provider 类型 `<Select>` 替换为 `<Radio.Group>`：
  - 三个 `<Radio value="...">` 对应 claude / openai / openai_compatible
  - label 使用现有 `t("llm.configForm.providerOptions.*")` key（已存在，不变）
  - `Form.Item` 的 `help` prop 传入 `providerHelpText[providerType]`（`providerType` 由 `Form.useWatch` 获取，已在文件中存在）

- [ ] **7.4** 新增 `providerHelpText` 映射（参照 TRD §LLMConfigForm，key 为 `llm.configForm.providerHelp.*`）
  - 注意：`providerType` 可能为 `undefined`（初始化前），需处理 undefined 时 help 为 `undefined` 的情况

- [ ] **7.5** 在 model `Form.Item` 和 fallback `Form.Item` 之间插入 `<Divider style={{ margin: "8px 0 16px" }} />`

- [ ] **7.6** 验证其余所有逻辑完全不变：
  - `useEffect` reset 逻辑
  - `rules` object（`providerType`、`baseUrl`、`apiKeyCreate`、`model`）
  - `handleOk` 提交逻辑
  - `requiresBaseUrl`、`apiKeyOptional` 条件渲染
  - `mode === "edit"` 时 apiKey 可留空
  - `probe: true` 保存逻辑

- [ ] **7.7** 运行 `cd frontend && pnpm lint:all`，修复全部错误

- [ ] **7.8** 提交
  - `git add frontend/src/features/settings-llm/LLMConfigForm.tsx`
  - commit message: `feat(settings-llm): replace Select with Radio.Group and add form section divider`

## 验证标准

- `<Select>` 不再出现于 Provider 类型字段
- 三个 `<Radio>` 按 plain 圆形样式渲染
- 选中不同 Radio 时 `help` 文字动态更新
- model 和 fallback 字段之间有 `<Divider>`
- 所有原有逻辑（校验、提交、reset、条件渲染）与重构前行为完全相同
- `data-testid="llm-config-form-modal"` 和 `data-testid="llm-config-form"` 保留
- `pnpm lint:all` 通过
