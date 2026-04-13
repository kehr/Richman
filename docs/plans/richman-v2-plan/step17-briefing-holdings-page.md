# Step 17: Research Briefing + Holdings Enhancements

> Phase 4 | 并行组 R9 (可与 Step 15, 16, 18 同时执行) | 前置: Step 14

## 任务目标

重构投研简报页（从 DashboardPage 到 ResearchBriefingPage）+ research-briefing feature 模块 + BriefingCard 组件（sparkline + 反馈按钮）+ 简洁/详细模式切换，以及持仓管理页增强：标记/快速/明细三模式切换、风险偏好选择弹窗、集中度 Alert、升级提示标签、LLM 配置引导，和 user-feedback feature 模块。

## 涉及文件

### 创建

**Feature 模块：**
- `frontend/src/features/research-briefing/api.ts`
- `frontend/src/features/research-briefing/types.ts`
- `frontend/src/features/research-briefing/use-briefing.ts`
- `frontend/src/features/research-briefing/index.ts`
- `frontend/src/features/user-feedback/api.ts`
- `frontend/src/features/user-feedback/types.ts`
- `frontend/src/features/user-feedback/use-submit-feedback.ts`
- `frontend/src/features/user-feedback/index.ts`

**简报页组件：**
- `frontend/src/pages/briefing/briefing-page.tsx` (重构自 dashboard)
- `frontend/src/pages/briefing/components/briefing-header.tsx`
- `frontend/src/pages/briefing/components/briefing-card-list.tsx`
- `frontend/src/pages/briefing/components/briefing-card.tsx`
- `frontend/src/pages/briefing/components/empty-briefing-state.tsx`

**持仓增强组件：**
- `frontend/src/pages/portfolio/components/mode-selector.tsx`
- `frontend/src/pages/portfolio/components/tag-mode-form.tsx`
- `frontend/src/pages/portfolio/components/risk-preference-modal.tsx`
- `frontend/src/pages/portfolio/components/holding-upgrade-tag.tsx`
- `frontend/src/pages/portfolio/components/concentration-alert.tsx`
- `frontend/src/pages/portfolio/components/llm-config-guide.tsx`

### 修改

- `frontend/src/pages/dashboard/` -- 改名/重构为 briefing（或保留目录但内容重写）
- `frontend/src/features/dashboard-summary/` -- 改名为 research-briefing + API 切换 v2
- `frontend/src/i18n/locales/zh/briefing.json` -- 新增
- `frontend/src/i18n/locales/en/briefing.json`
- `frontend/src/i18n/locales/zh/common.json` -- 新增 disclaimer.* key
- `frontend/src/i18n/locales/en/common.json`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| 简报页结构 | SS6 投研简报 | frontend SS6.1 |
| BriefingCard 内容 | SS6.2 卡片内容 | frontend SS6.2 |
| 简洁/详细模式 | SS6 简报 | frontend SS6.3 |
| 反馈按钮 (点赞/点踩) | SS6.3 反馈机制 | frontend SS6.2 |
| 持仓录入三模式 | SS7 持仓管理 | frontend SS7.1 |
| TagModeForm | SS7.1 标记模式 | frontend SS7.1 |
| RiskPreferenceModal | SS7.6 风险偏好 | frontend SS7.2 |
| 集中度 Alert | SS8.2 集中度 | frontend SS7.4 |
| LLM 配置引导 | SS9.1 首次体验 | frontend SS7.5 |
| dashboard-summary 改名 | - | frontend SS13.2 |
| briefing_view_mode localStorage | - | frontend SS6.3 |
| 免责声明 4 个位置 | SS13 免责声明 | frontend SS14 |

## 关键约束

- BriefingCard 点击跳转 `/market/:code`（执行 Tab）
- sparkline 使用最近 90 天评分数据（简易折线，不需完整图表库）
- 简洁/详细模式存储在 localStorage `richman_briefing_view_mode`
- 持仓录入三模式：标记（仅选标的）/ 快速（标的+仓位）/ 明细（全部字段）
- RiskPreferenceModal 三型：conservative / moderate / aggressive
- 集中度 Alert 三级：red(>30%) / orange(>20%) / blue(>10%)
- LLM 配置引导在用户首次添加持仓且无 LLM 配置时弹出
- dashboard-summary feature 目录改名为 research-briefing，API 切换 v2
- 免责声明在 4 个位置展示（简报页底部、执行计划底部、邮件底部、注册页）

## 验证标准

- [ ] `cd frontend && pnpm lint:all` 全部通过
- [ ] `pnpm build` 成功
- [ ] /briefing 页面正常渲染
- [ ] BriefingCard 展示持仓 + 评分 + sparkline + 反馈按钮
- [ ] 简洁/详细模式切换正常，刷新后保持选择
- [ ] 持仓页三模式切换正常
- [ ] 集中度 Alert 在超阈值时显示正确级别颜色
- [ ] 无持仓时显示 EmptyBriefingState

## 变更点清单覆盖

E3.3 (1), E1.4 (1), E1.5 (1), E6.1-E6.4 (4), E7.1-E7.6 (6), E11.3 (1), E11.9 (1), E12.5-E12.6 (2) = **17 项**
