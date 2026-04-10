# Step 7: Frontend i18n

**依赖：** 无（独立）
**可与 Step 5/6/8 并行**
**设计依据：** TRD §i18n 键命名空间

## 任务目标

在中英文 settings locale 文件中新增 `schedule` 命名空间的所有翻译键。

## 涉及文件

- 修改：`frontend/src/i18n/locales/zh/settings.json`
- 修改：`frontend/src/i18n/locales/en/settings.json`

## 执行步骤

- [ ] 阅读 TRD §i18n 键命名空间，获取所有需要的键
- [ ] 在 `zh/settings.json` 的 `tabs` 对象中添加 `"schedule": "调度策略"`
- [ ] 在 `zh/settings.json` 中新增顶层 `"schedule"` 对象，包含 TRD 中所有中文翻译键（globalFrequency、window、markets、holdingOverride 四个子命名空间）
- [ ] 在 `en/settings.json` 的 `tabs` 对象中添加 `"schedule": "Schedule"`
- [ ] 在 `en/settings.json` 中新增顶层 `"schedule"` 对象，包含对应英文翻译
- [ ] 执行 `cd frontend && pnpm lint:all` 验证通过（Biome 会检查 JSON 格式）
- [ ] `git add frontend/src/i18n/ && git commit -m "feat(i18n): add schedule namespace translations"`

## 验证标准

- `pnpm lint:all` 通过
- 中英文 `tabs.schedule` key 均存在
- TRD §i18n 中列出的所有 key 在两个 locale 文件中均有对应翻译，无遗漏
