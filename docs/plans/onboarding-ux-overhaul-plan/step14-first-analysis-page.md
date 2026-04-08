# Step 14 FirstAnalysisPage 使用 analysisFired 持久化

## 任务目标

将 `FirstAnalysisPage` 的 `startedRef` 迁移到 `OnboardingState.analysisFired`（sessionStorage 持久化），保证回退再前进时不会重复触发分析。保持现有的「mutation 不 gate」修复（fire-and-forget 语义）。每个步骤项的 checkmark 用 SVG pathLength 动画 draw-in，增强生动感。

## 涉及文件

修改：
- `frontend/src/pages/onboarding/FirstAnalysisPage.tsx`

## 设计依据

- PRD §3.6 页面改造 - FirstAnalysisPage
- PRD 附录 B Pass 2 契约打破警报 #1：startedRef 迁移到 sessionStorage
- PRD §4.3 动效清单：checkmark pathLength 0→1 draw-in、完成态 scale pulse

## 实施要点

- 删除本地 `startedRef: useRef<boolean>`，改为读 `state.analysisFired`
- `useEffect` mount 时：
  - 如果 `state.analysisFired === true` → 跳过触发分析，直接从 currentStep 0 开始 4 个步骤的动画演示
  - 否则触发 `rerunAnalysis.mutateAsync()` + `setState({ analysisFired: true })`
- 保留既有的「不 gate 在 isPending」的修复：4 步动画跑完后直接 markCompleted + navigate，不等待 mutation
- 4 个 step item 的 checkmark 用 framer-motion 的 `motion.svg` + `motion.path`，`initial={{ pathLength: 0 }}`, `animate={{ pathLength: 1 }}`, transition 0.4s ease-out
- 完成态整块 `animate={{ scale: [1, 1.05, 1] }}` pulse 一次
- `useReducedMotion` 降级：checkmark 直接显示（无 draw-in），无 pulse
- step 4 页面可以被 `nav.prev()` 回退到 step 3（回退时 sessionStorage 保留 analysisFired，不会重新触发）

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 手动测试：
   - 首次到达 step 4：触发分析 + 动画演示 + 自动跳 dashboard
   - 回退到 step 3，再前进到 step 4：只演示动画，不再调 `/analysis/trigger`（通过 network 面板确认）
3. 验证 sessionStorage 的 `richman_onboarding_draft` 包含 `analysisFired: true` 字段
4. 键盘 ← 回退到 step 3 正常工作
5. 跳过按钮触发 skip 流程

## 依赖说明

前置：step10（OnboardingLayout）+ step07（state provider 的 analysisFired 字段）
