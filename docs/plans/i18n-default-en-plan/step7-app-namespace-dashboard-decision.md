# Step 7: app namespace 迁移 -- dashboard + decision-card

## 任务目标

将 dashboard 页面组件和 decision-card 相关组件中的所有硬编码中文替换为 `t("app:...")` 调用。同时传入 locale 参数给已重构的 format helpers。

## 涉及文件

Dashboard 页面：
- 修改: `frontend/src/pages/dashboard/DashboardPage.tsx`
- 修改: `frontend/src/pages/dashboard/components/DashboardTopStrip.tsx`
- 修改: `frontend/src/pages/dashboard/components/EmptyHoldingsHero.tsx`
- 修改: `frontend/src/pages/dashboard/components/DecisionCardWall.tsx`
- 修改: `frontend/src/pages/dashboard/components/ChangeAnchorList.tsx`
- 修改: `frontend/src/pages/dashboard/components/OnboardingSkippedNudge.tsx`
- 修改: `frontend/src/features/dashboard-llm-status/LLMStatusBanner.tsx`

Decision Card 页面 + 组件：
- 修改: `frontend/src/pages/decision-cards/DecisionCardDetailPage.tsx`
- 修改: `frontend/src/pages/decision-cards/components/ConclusionBanner.tsx`
- 修改: `frontend/src/pages/decision-cards/components/MainRisks.tsx`
- 修改: `frontend/src/pages/decision-cards/components/ExecutionPlanFull.tsx`
- 修改: `frontend/src/pages/decision-cards/components/DimensionReasoning.tsx`
- 修改: `frontend/src/pages/decision-cards/components/MetaSidebar.tsx`
- 修改: `frontend/src/pages/decision-cards/components/CardHero.tsx`
- 修改: `frontend/src/features/decision-card/components/DecisionCardSummary.tsx`
- 修改: `frontend/src/features/decision-card/components/SourcePill.tsx`
- 修改: `frontend/src/features/decision-card/components/ChangeBadge.tsx`
- 修改: `frontend/src/features/decision-card/components/ExecutionPlanStrip.tsx`

## PRD/TRD 引用

- PRD §4.1（dashboard + decision-card 迁移范围）
- TRD §12（字符串迁移约定）
- TRD §6.3-6.4（format helpers 调用方传入 locale）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm test` 通过
- [ ] `rg '[\u4e00-\u9fff]' frontend/src/pages/dashboard frontend/src/pages/decision-cards frontend/src/features/dashboard-llm-status frontend/src/features/decision-card --type tsx` 结果为零（测试文件除外）
- [ ] `pnpm dev` 启动后 Dashboard 和 Decision Card 详情页默认英文
- [ ] 切中文后所有文案 + 数字格式同步切换
- [ ] Decision Card 的推荐标签（积极加仓/持有观望等）随语言切换

## 依赖

- Step 3（app namespace JSON 已就绪）
- Step 5（format helpers 已重构，调用方可传 locale）

## 实施注意

- DashboardTopStrip 有 9 处中文且使用 formatAmount，需要同时加 useTranslation + 传 locale 给 format
- DecisionCardDetailPage 有 12 处中文，是 decision-card 最密集的文件
- SourcePill / ChangeBadge 中的中文是标签文案（如「利好」「利空」），key 格式用 `app:decisionCard.sourcePill.bullish` 等
- DimensionReasoning 有 9 处，包含趋势/仓位/催化剂维度标签
- MetaSidebar 中 Intl.DateTimeFormat 硬编码 "en-GB"，本 step 改为读 locale（通过 TRD §6 的模式）
