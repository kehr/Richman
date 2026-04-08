# Step 13 FirstHoldingPage 表单 stagger 与跳过按钮语义修正

## 任务目标

将 `FirstHoldingPage` 接入 `useOnboardingNav`，表单字段 stagger 进场，修复既有「跳过直接分析」按钮的语义契约破坏：原来直接调 `markCompleted` 跳过整个流程，改为 `nav.next()` 前进到 step 4 由 FirstAnalysisPage 统一处理。按钮文案改为「用已有持仓直接分析 →」与 header 的「跳过引导」区分。表单状态通过 state provider 持久化。

## 涉及文件

修改：
- `frontend/src/pages/onboarding/FirstHoldingPage.tsx`

## 设计依据

- PRD §3.6 页面改造 - FirstHoldingPage
- PRD 附录 B Pass 2 契约打破警报 #2：按钮语义从 markCompleted 改为 nav.next
- PRD 附录 C Pass 3 主路径 E：回退再前进的 state 持久化

## 实施要点

- 替换 `useNavigate` + 直接 mutation 调用，改为 `useOnboardingNav`
- 表单字段的 value 和 onChange 接入 `useOnboardingState` 的 `holdingDraft`
- 三 tab 状态（quick / detail / screenshot）存到 `holdingDraft.mode`，持久化到 sessionStorage
- 检测到 `holdings.length > 0` 时渲染 alert 条：
  - 文案：「检测到你已有 X 个持仓」
  - 按钮：「用已有持仓直接分析 →」（文案修正！）
  - 行为：`nav.next()` 前进到 step 4，**不再调用 markCompleted**
- 当前 quick / detail 模式的表单验证接入 `registerCanGoNext`：
  - quick mode: 校验 costPrice 和 positionRatio 有值
  - detail mode: 校验所有必填字段
  - screenshot mode: canGoNext 始终 false（screenshot 流程由自己的 Modal 处理）
- 主表单字段 stagger 进场（container staggerChildren 0.08）
- `useReducedMotion` 降级无位移
- 用户从 state provider 恢复 draft 时，form 字段应反映已保存值

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 单元测试更新：
   - 按钮文案断言改为「用已有持仓直接分析」
   - 按钮点击断言：navigate 到 first-analysis，不调 markCompleted
3. 手动走 onboarding：
   - 无持仓用户：看到 quick/detail/screenshot 三 tab
   - 有持仓用户：看到 alert + 「用已有持仓直接分析」按钮
   - 填 quick mode 字段，前进到 step 4；再回退，字段保留
   - 切换到 detail tab 也持久化
4. canGoNext 校验：quick mode 必填字段留空时「下一步」disabled

## 依赖说明

前置：step10（OnboardingLayout）+ step07（state provider）
