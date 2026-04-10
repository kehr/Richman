# settings-llm UI 重设计执行报告

## 执行方式

- 分支：`feat/settings-llm-redesign`
- Worktree：`.claude/worktrees/settings-llm-redesign/`（已清理）
- 执行模式：subagent-driven-development（三阶段并行）
- 基准 commit：`bf23631`（plan 提交）
- 最终 HEAD：`801b873`

## 全局规则

- 零 AI 痕迹：commit message / 注释 / 文件内容中不出现 AI/Claude/Anthropic 相关字样
- 代码和注释使用英文，文档使用中文
- 所有 AntD 运行时组件导入必须通过 `@/ui-kit/eat` barrel（纯类型导入例外）
- 每次修改文件后必须运行 `cd frontend && pnpm lint:all` 并修复全部错误
- 前端禁止 `.test.tsx` UI 测试，只允许纯函数 `.test.ts`

## 并行执行计划

| 阶段 | Steps | 状态 |
|------|-------|------|
| Phase 1 | Step 1（eat barrel + formatRelativeTime）+ Step 2（i18n）| 完成 |
| Phase 2 | Step 3（ProviderCardLayout）+ Step 6（LLMEmptyState）+ Step 7（LLMConfigForm）| 完成 |
| Phase 3 | Step 4（LLMHealthyCard）+ Step 5（LLMFailingCard）| 完成 |

## Step 执行记录

### Step 1: eat barrel + formatRelativeTime

- 状态：完成
- 涉及文件：`frontend/src/ui-kit/eat/index.ts`、`frontend/src/features/settings-llm/utils/formatRelativeTime.ts`（新建）、`formatRelativeTime.test.ts`（新建）
- Commit SHA：`7dc4753`（后续 fix：`af6d581`）
- 关键决策：
  - `formatRelativeTime(date, lang)` 使用原生 `Intl.RelativeTimeFormat`，无需新增依赖
  - null/undefined 参数返回 `"—"`（长破折号）
  - 7 个时间阈值：second / minute / hour / day / week / month / year
- 偏差说明：初版缺少 week / month 单元测试，code quality review 发现后补充

### Step 2: i18n 变更

- 状态：完成
- 涉及文件：`frontend/src/i18n/locales/zh/settings.json`、`frontend/src/i18n/locales/en/settings.json`
- Commit SHA：`d6d3f06`（后续 fix：`801b873`）
- 关键决策：
  - 新增 key：`llm.healthyCard.{lastProbedAt,fallbackOn,fallbackOff,deleteMenuLabel,fallbackLabel}`
  - 新增 key：`llm.emptyState.description`
  - 更新 key：`llm.emptyState.callout.{systemConsentGiven,systemNoConsent}`（更简短文案）
  - 新增 key：`llm.configForm.providerHelp.{claude,openai,openai_compatible}`
  - 新增 key：`llm.failingCard.deleteError`
- 偏差说明：初版残留 10 个废弃 key，final code review 发现后在 `801b873` 中清除

### Step 3: ProviderCardLayout

- 状态：完成
- 涉及文件：`frontend/src/features/settings-llm/ProviderCardLayout.tsx`（新建）
- Commit SHA：`42973d2`（后续 fix：`8180f4f`）
- 关键决策：
  - 组件不通过 feature barrel 导出，仅 HealthyCard / FailingCard 内部使用
  - Props 设计：`onEdit` 不出现在 ProviderCardLayout（edit 按钮在 footerContent 中传入）
  - `PROVIDER_BADGE_STYLE` 颜色 map：claude=蓝，openai=绿，openai_compatible=紫；failing 状态覆盖为红色
  - Header border 使用 `theme.useToken()` 的 `token.colorBorderSecondary`，颜色跟随主题
  - Dropdown 仅含一项：删除（带 Popconfirm 确认）
  - 本地定义 `BadgeStatus` union type，不从 antd 导入（规避 Biome noRestrictedImports）
- 偏差说明：初版 `dropdownItems` 缺少 `MenuProps["items"]` 类型标注导致 lint 报错，`8180f4f` 修复（使用 eat barrel 导出的 `MenuProps`）

### Step 4: LLMHealthyCard

- 状态：完成
- 涉及文件：`frontend/src/features/settings-llm/LLMHealthyCard.tsx`
- Commit SHA：`d0421b6`
- 关键决策：
  - 通过 `<ProviderCardLayout healthStatus="healthy">` 构建三段式卡片
  - bodyContent：4 格 info grid（model / apiKeyHint / baseUrl 条件渲染 / fallbackLabel+fallbackText）+ Divider + toggle 行
  - footerContent：`<Space><LLMProbeButton /><Button>编辑</Button></Space>`
  - 删除操作从 footer 移入 ProviderCardLayout 的 Dropdown（由 `onDelete` prop 传入）
- 偏差说明：无

### Step 5: LLMFailingCard

- 状态：完成
- 涉及文件：`frontend/src/features/settings-llm/LLMFailingCard.tsx`
- Commit SHA：`ea7d9a7`（后续 fix：`fe520c5`、`801b873`）
- 关键决策：
  - 通过 `<ProviderCardLayout healthStatus="failing">` 自动触发红色 badge 覆盖
  - bodyContent：Alert type="error" showIcon + 2 格 info grid（model / baseUrl）+ Divider + fallbackCopy 文字
  - footerContent：`<Space><LLMProbeButton label={retestButton} /><Button>编辑</Button></Space>`
- 偏差说明：
  - 初版 footerContent 缺少 `<Space>` 包裹导致按钮无间距，`fe520c5` 修复
  - 初版 `handleDelete` 引用了跨命名空间 key `healthyCard.deleteError`，`801b873` 添加专属 `failingCard.deleteError` key 并修正引用

### Step 6: LLMEmptyState

- 状态：完成
- 涉及文件：`frontend/src/features/settings-llm/LLMEmptyState.tsx`
- Commit SHA：`2252bc0`（后续 fix：`500f252`）
- 关键决策：
  - 移除 AntD `<Empty>` 组件
  - 使用 `lucide-react` 的 `Bot` 图标（size=32，直接导入，不经 eat barrel）
  - icon 颜色使用 `theme.useToken()` 的 `token.colorTextQuaternary`，跟随主题
  - callout 三分支 IIFE：`!systemDefaultAvailable` → null；有同意 → Alert info；未同意 → Alert warning
  - callout Alert 使用 `borderTop: "none", borderRadius: "0 0 8px 8px"` 贴合 Card 底边
- 偏差说明：初版 callout 逻辑丢失了 `!systemDefaultAvailable → null` 分支，`500f252` 修复

### Step 7: LLMConfigForm

- 状态：完成
- 涉及文件：`frontend/src/features/settings-llm/LLMConfigForm.tsx`
- Commit SHA：`0b97ccc`
- 关键决策：
  - 移除 `<Select>` 和 `providerOptions` useMemo，改用 `<Radio.Group>` 3 个纯 `<Radio>`（不加 optionType="button"）
  - `providerHelpText` 类型用 `Partial<Record<LLMProviderType, string>>`（TRD 写的是 `Record`，因 `Form.useWatch` 初始值为 undefined 故用 Partial）
  - `Form.Item` 的 `help` prop 展示 per-provider 描述文字
  - 添加 `<Divider style={{ margin: "8px 0 16px" }} />` 分隔凭证区和行为设置区
- 偏差说明：providerHelpText 类型从 `Record` 改为 `Partial<Record>` 属于有意偏差（防 undefined 类型错误）

## 已修复问题

| 问题 | 修复方式 | 修复 commit |
|------|----------|-------------|
| formatRelativeTime 缺少 week/month 单元测试 | 补充 2 个 test case | `af6d581` |
| LLMEmptyState callout null 分支丢失 | 恢复三分支 IIFE | `500f252` |
| ProviderCardLayout `dropdownItems` 缺类型标注导致 lint 报错 | 改用 `MenuProps["items"]` 类型（from eat barrel）| `8180f4f` |
| LLMFailingCard footerContent 按钮无间距 | 添加 `<Space>` 包裹 | `fe520c5` |
| i18n 残留 10 个废弃 key | grep 逐一确认后删除 7 个真正废弃的 key | `801b873` |
| LLMFailingCard 跨命名空间引用 `healthyCard.deleteError` | 添加 `failingCard.deleteError` 专属 key | `801b873` |

## 已记录但未修复的观察项

1. **Popconfirm UX**：Dropdown 内的删除项点击后触发 Popconfirm，但 Popconfirm 挂载在 Dropdown 内部可能导致弹层层级问题（实际表现需人眼验收）。若有异常可考虑改为 Modal.confirm。
2. **Card padding 与 callout 的视觉过渡**：LLMEmptyState 的 callout Alert 使用 `borderTop: none` 贴合 Card 底边，但如果 AntD 版本 Alert 内边距变化可能出现缝隙，需人眼验收。
3. **"unknown" healthStatus 未在实际数据中出现**：TRD 定义了 `unknown` 状态对应灰色 Badge，但 API 目前未返回此值，UI 路径未实际测试到。

## 无法决策项

无。

## Review 结果

### Spec Compliance Review

所有 7 个 step 均通过 spec compliance review，各组件的结构、颜色语义、i18n key、props 接口均与 PRD/TRD 规格对齐。ProviderCardLayout `onEdit` prop 被识别为不必要而移除（edit 按钮在 footerContent 传入），spec reviewer 确认为合理简化。

### Code Quality Review

所有 step 均通过 code quality review（经修复循环）。主要问题均已修复（见"已修复问题"一节）。

### Final Code Review

最终整体 review 发现并修复：10 个废弃 i18n key 中 7 个被清除，LLMFailingCard 跨命名空间 key 被修正。最终 lint 全部通过。
