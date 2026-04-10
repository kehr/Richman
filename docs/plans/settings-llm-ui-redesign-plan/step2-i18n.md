# Step 2: i18n 变更

**依赖：** 无（Phase 1，可与 Step 1 并行）

## 任务目标

按 TRD i18n 完整变更章节，同步更新中英文 `settings.json`：
- 在 `llm.healthyCard` 新增 `lastProbedAt`、`fallbackOn`、`fallbackOff`、`deleteMenuLabel`
- 在 `llm.emptyState` 新增 `description`；更新 `callout.systemConsentGiven` 和 `callout.systemNoConsent` 文案；删除 `callout.noSystem`
- 在 `llm.configForm` 新增 `providerHelp.{claude,openai,openai_compatible}`
- 删除 `llm.healthyCard.lastProbed`、`llm.failingCard.lastProbed`

## 涉及文件

- 修改：`frontend/src/i18n/locales/zh/settings.json`
- 修改：`frontend/src/i18n/locales/en/settings.json`

## 设计依据

- PRD §i18n 变更：key 清单
- TRD §i18n 完整变更：精确 key 路径、中英文文案、移除 key 清单

## 实施步骤

- [ ] **2.1** 修改 `zh/settings.json`：
  - `llm.healthyCard`：新增 `lastProbedAt: "测试于 {{time}}"`、`fallbackOn: "已开启"`、`fallbackOff: "已关闭"`、`deleteMenuLabel: "删除 Provider 配置"`
  - 删除 `llm.healthyCard.lastProbed`
  - `llm.emptyState`：新增 `description` 字段；将 `callout.systemConsentGiven` 和 `callout.systemNoConsent` 替换为 TRD 中的精简版文案；删除 `callout.noSystem`
  - `llm.configForm`：新增 `providerHelp` 对象（claude / openai / openai_compatible 三个子键）
  - 删除 `llm.failingCard.lastProbed`

- [ ] **2.2** 修改 `en/settings.json`：与 zh 完全对称，使用 TRD §新增 key（en/settings.json）中的英文文案
  - 同样删除 `llm.healthyCard.lastProbed` 和 `llm.failingCard.lastProbed`
  - 同样删除 `llm.emptyState.callout.noSystem`

- [ ] **2.3** 运行 `cd frontend && pnpm lint:all`，修复全部错误

- [ ] **2.4** 提交
  - `git add frontend/src/i18n/locales/`
  - commit message: `feat(settings-llm): update i18n keys for ui redesign`

## 验证标准

- 中英文两个 JSON 结构完全对称（新增 key 在两个文件中均存在）
- 已移除的 key（`lastProbed` x2、`noSystem`）在两个文件中均不存在
- JSON 格式合法，`pnpm lint:all` 通过
