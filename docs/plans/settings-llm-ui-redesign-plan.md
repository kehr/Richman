# settings-llm UI 重设计实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 settings-llm feature 的 4 个 UI 组件按行业最佳实践（三段式 Card、Badge 状态点、Radio.Group 枚举选择、相对时间）重构，提升视觉层次和专业调性。

**Architecture:** 新增 `ProviderCardLayout` 内部共享组件承载 HealthyCard / FailingCard 的三段式骨架和 Dropdown 删除逻辑；新增 `formatRelativeTime` 工具函数处理相对时间；eat barrel 补充 `EllipsisOutlined` 导出；i18n 按 TRD 增删 key。

**Tech Stack:** React 19, Ant Design 6, lucide-react (Bot icon), react-i18next, Intl.RelativeTimeFormat

**设计依据：**
- PRD: `docs/prds/settings-llm-ui-redesign-prd.md`
- TRD: `docs/trds/settings-llm-ui-redesign-trd.md`

**执行策略（三阶段并行）：**

| 阶段 | 并行 Steps | 前置条件 |
|------|-----------|---------|
| Phase 1 | Step 1 + Step 2 | 无 |
| Phase 2 | Step 3 + Step 6 + Step 7 | Phase 1 完成 |
| Phase 3 | Step 4 + Step 5 | Step 3 完成（Phase 2） |

**Steps 索引：**

- [ ] [Step 1: 基础设施 — eat barrel + formatRelativeTime](settings-llm-ui-redesign-plan/step1-infrastructure.md)
- [ ] [Step 2: i18n 变更](settings-llm-ui-redesign-plan/step2-i18n.md)
- [ ] [Step 3: ProviderCardLayout 内部共享组件](settings-llm-ui-redesign-plan/step3-provider-card-layout.md)
- [ ] [Step 4: LLMHealthyCard 重构](settings-llm-ui-redesign-plan/step4-llm-healthy-card.md)
- [ ] [Step 5: LLMFailingCard 重构](settings-llm-ui-redesign-plan/step5-llm-failing-card.md)
- [ ] [Step 6: LLMEmptyState 重构](settings-llm-ui-redesign-plan/step6-llm-empty-state.md)
- [ ] [Step 7: LLMConfigForm 重构](settings-llm-ui-redesign-plan/step7-llm-config-form.md)
