# Step 13 Onboarding 4 屏页面

## 任务目标

实现 PRD §2.3 的 4 步强制 onboarding 流程，从 /onboarding/welcome 一直到 /onboarding/first-analysis 自动跳 /dashboard。

## 涉及文件

创建：
- `frontend/src/pages/onboarding/WelcomePage.tsx`
- `frontend/src/pages/onboarding/CategoriesPage.tsx`
- `frontend/src/pages/onboarding/FirstHoldingPage.tsx`
- `frontend/src/pages/onboarding/FirstAnalysisPage.tsx`
- `frontend/src/pages/onboarding/components/OnboardingLayout.tsx`（共享的步骤指示器 / 居中容器）
- `frontend/src/pages/onboarding/components/StepIndicator.tsx`
- `frontend/src/features/onboarding/api.ts`
- `frontend/src/features/onboarding/use-mark-completed.ts`
- `frontend/src/features/onboarding/index.ts`

修改：
- `frontend/src/routes.tsx`（接入 4 个 onboarding 路由组件）

## 设计依据

- PRD §2.3 4 步强制流程
- TRD §6.1 OnboardingGuard 与 onboarding 完成标记
- PRD §1.5 4 个标的类型

## 实施要点

- WelcomePage：单列居中，三维简介卡片 + "开始设置 →"按钮 → 跳 /onboarding/categories
- CategoriesPage：2x2 网格多选，至少选 1 个；选择写入 user_settings.categories；下一步前调 PATCH /api/v1/user/settings
- FirstHoldingPage：复用 features/portfolio 的 AddHolding 表单组件（step16 实现的可复用部分）；底部小字"先录 1 个就行，后面随时可以加"和可选"设置总资金（可选）"链接
- FirstAnalysisPage：
  - 进入页面时 POST /api/v1/analysis/run 触发首次分析
  - 通过 TanStack Query 轮询 GET /api/v1/analysis/tasks/:taskId 显示 4 步进度
  - 完成时调 useMarkCompleted 标记 onboarding 完成
  - 标记完成后跳 /dashboard
  - 异常分支：分析失败时显示重试按钮和"暂时跳过先看 Dashboard"链接
- 所有页面的 antd 组件通过 ui-kit/eat barrel 导入
- 步骤指示器组件接受 currentStep + totalSteps 渲染 "第 N / M 步" 小字

## 验证标准

1. `pnpm lint:all` 通过
2. 注册一个新邀请码用户走完 4 步，最终到达 /dashboard 看到首张决策卡
3. 中途刷新页面不破坏流程（守卫保证仍在 onboarding）
4. CategoriesPage 不选任何类型时下一步按钮置灰
5. 已完成 onboarding 的用户访问任何 /onboarding/* 都被守卫跳走

## 依赖说明

- 前置：step10 路由 + 守卫、step11 user-settings hook、step12 决策卡组件库（FirstAnalysisPage 完成跳转后看到的卡）
- 注意：FirstHoldingPage 需要 AddHolding 表单组件，建议把组件抽出来由 step16 统一管理；本 step 临时用一个简化版本，由 step16 替换

## 预估提交

- commit 1: `feat(onboarding): add welcome and categories pages`
- commit 2: `feat(onboarding): add first holding page with quick mode`
- commit 3: `feat(onboarding): add first analysis progress page with auto redirect`
