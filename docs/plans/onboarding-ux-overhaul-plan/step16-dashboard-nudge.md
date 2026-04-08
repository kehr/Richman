# Step 16 Dashboard nudge 组件与页面集成

## 任务目标

新建 `OnboardingSkippedNudge` 组件，在 `DashboardPage` 顶部渲染（skipped 且未 dismiss 时）。`DashboardPage` 原有的 `holdings.length === 0` 早退逻辑重构为支持 nudge + EmptyHoldingsHero 共存的 flex column 布局。`EmptyHoldingsHero` 新增次级文字链「或者先走一遍引导」作为 dismissed + empty 孤岛的 regret 路径。

## 涉及文件

创建：
- `frontend/src/pages/dashboard/components/OnboardingSkippedNudge.tsx`

修改：
- `frontend/src/pages/dashboard/DashboardPage.tsx`
- `frontend/src/pages/dashboard/components/EmptyHoldingsHero.tsx`

## 设计依据

- PRD §3.7 Dashboard nudge 组件 / §3.9 DashboardPage 改造
- TRD §6.3 DashboardPage flex 容纳 nudge（完整 JSX 结构）+ §6.4 OnboardingSkippedNudge 组件（完整 dismiss 状态管理 + handleRestart 纯前端 navigate 决策）
- PRD 附录 B Pass 2 契约打破警报 #3：DashboardPage early return 必须改造让 nudge 能渲染
- PRD 附录 D Pass 4 Pre-mortem bug 4：nudge + hero 滚动穿透 → flex column + hero flex:1

## 实施要点

- `OnboardingSkippedNudge` 组件：
  - 读 `useOnboardingStatus()` + 读 `localStorage` dismissal 标记
  - 条件渲染：`status.skipped && !dismissed` 才返回 JSX，否则返回 null
  - 样式：inline alert bar 非 sticky，黑底白字与现有主题一致
  - 文案：「你跳过了引导流程，走一遍可以更好理解决策卡」
  - 主 CTA「开始引导」：纯前端 `navigate("/onboarding/welcome")`，**不调 mutation**，依赖 step15 扩展后的 guard 放行 skipped 用户
  - 次要按钮「不再提示」：`localStorage.setItem("richman_onboarding_nudge_dismissed", "1")` + 本地 state 立即隐藏
- `DashboardPage` 重构：
  - 外层改为 `<Flex vertical gap={16}>` 或等价 CSS
  - 顶部挂 `<OnboardingSkippedNudge />`
  - 原本 `if (holdings.length === 0) return <EmptyHero />` 的 early return 改为：主内容区内用条件 `{holdings.length === 0 ? <EmptyHero /> : <ThreeRegionLayout />}`
  - EmptyHoldingsHero 容器设 `flex: 1` 自适应剩余空间
- `EmptyHoldingsHero` 组件：
  - 底部新增次级文字链「或者先走一遍引导」
  - 点击：`navigate("/onboarding/welcome")`
  - 样式：淡灰小字，点击区域明显

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 组件单元测试：
   - `OnboardingSkippedNudge`：skipped=true 渲染、skipped=false 不渲染、dismissed 时不渲染、点击 dismiss 后立即隐藏
   - `DashboardPage`：skipped + 0 holdings 时同时显示 nudge 和 hero、completed + 0 holdings 只显示 hero
3. 手动 smoke：
   - 跳过 onboarding → dashboard 看到 nudge + EmptyHoldingsHero
   - 点 nudge 主 CTA → 跳 welcome
   - 再次跳过 → 回 dashboard → 点「不再提示」→ nudge 消失 + 只剩 EmptyHero 带次级文字链
   - 点 EmptyHero 的「或者先走一遍引导」→ 跳 welcome
4. 页面无滚动穿透，nudge + hero 布局正常

## 依赖说明

前置：step05（`useOnboardingStatus` 支持 skipped 字段）、step15（guard 允许 skipped 用户访问 onboarding 路由）
