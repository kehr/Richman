# Step 11: 测试文件迁移

## 任务目标

更新所有包含硬编码中文断言的测试文件，使其匹配 i18n 后的英文默认文案。测试 wrapper 在 Step 2 已更新（lng="en"），所以测试中的 assert 应改为英文翻译值。

## 涉及文件

测试文件（含中文断言）：
- 修改: `frontend/src/pages/auth/LoginPage.test.tsx`
- 修改: `frontend/src/pages/dashboard/DashboardPage.test.tsx`
- 修改: `frontend/src/pages/dashboard/components/DashboardTopStrip.test.tsx`
- 修改: `frontend/src/pages/dashboard/components/ChangeAnchorList.test.tsx`
- 修改: `frontend/src/pages/dashboard/components/OnboardingSkippedNudge.test.tsx`
- 修改: `frontend/src/pages/decision-cards/DecisionCardDetailPage.test.tsx`
- 修改: `frontend/src/pages/decision-cards/components/MetaSidebar.test.tsx`
- 修改: `frontend/src/features/decision-card/components/DecisionCardSummary.test.tsx`
- 修改: `frontend/src/features/decision-card/components/ExecutionPlanStrip.test.tsx`
- 修改: `frontend/src/pages/portfolio/PortfolioListPage.test.tsx`
- 修改: `frontend/src/pages/portfolio/PortfolioTransactionsPage.test.tsx`
- 修改: `frontend/src/pages/portfolio/components/ScreenshotImportModal.test.tsx`
- 修改: `frontend/src/pages/portfolio/components/AddHoldingDrawer.test.tsx`
- 修改: `frontend/src/pages/portfolio/components/HoldingTable.test.tsx`
- 修改: `frontend/src/pages/settings/tabs/AccountTab.test.tsx`
- 修改: `frontend/src/pages/help/HelpPage.test.tsx`
- 修改: `frontend/src/pages/onboarding/WelcomePage.test.tsx`
- 修改: `frontend/src/pages/onboarding/FirstHoldingPage.test.tsx`
- 修改: `frontend/src/pages/onboarding/components/OnboardingLayout.test.tsx`
- 修改: `frontend/src/domain/auth/onboarding-guard.test.tsx`
- 修改: `frontend/src/domain/ui/use-typewriter.test.tsx`

## PRD/TRD 引用

- TRD §10（测试策略：隔离 i18n 实例 + 两层策略）
- TRD §10.2（单元测试用真实翻译 + 英文断言，或 mock t()）

## 验证标准

- [ ] `pnpm test` 全部通过
- [ ] `pnpm lint:all` 通过
- [ ] `rg '[\u4e00-\u9fff]' frontend/src --type tsx -g '*test*'` 结果为零或仅剩注释
- [ ] 无新增 `vi.mock("react-i18next")` 除非组件测试确实只关心结构不关心文案

## 依赖

- Step 6-10（被测组件已迁移完毕）
- Step 2（test/utils.tsx 的 i18n wrapper 已就绪）

## 实施注意

- 大多数测试的改动是机械的：把 `getByText("中文文案")` 改为 `getByText("English text")`，对应的英文值从 en namespace JSON 查
- HelpPage.test.tsx 是旧 useLocale 的最后消费者，删除 `import { I18nProvider }` + `<I18nProvider>` 包裹（test/utils.tsx 已提供 react-i18next wrapper）
- use-typewriter.test.tsx 有 8 处中文（typewriter 效果的测试字符串），这些可能是测试数据而非 UI 文案，需甄别是否需要 i18n 或保留为测试固定值
- onboarding-guard.test.tsx 的中文可能是 mock 数据或 console 输出，同样甄别
- 如果某些测试的断言改为英文后可读性下降（如复杂的 Form 校验消息匹配），可考虑用 `getByRole` / `getByTestId` 替代文案匹配
