# Step 10 OnboardingLayout 三段式重写

## 任务目标

重写 `OnboardingLayout` 为三段式结构（header bar + 标题区 + AnimatePresence 内容区），接入 `OnboardingBackground`、`OnboardingPageTransition`、`useOnboardingNav`，实现「← 上一步」按钮、「跳过引导」按钮、可点击 step indicator、全局键盘监听、skip 确认 Modal。在 OnboardingShell 层挂载 `OnboardingStateProvider`。

## 涉及文件

修改：
- `frontend/src/pages/onboarding/components/OnboardingLayout.tsx`
- `frontend/src/pages/onboarding/components/StepIndicator.tsx`
- `frontend/src/routes.tsx`（在 OnboardingShell 挂载 Provider）

## 设计依据

- PRD §3.3 OnboardingLayout 改造
- PRD §4.3 动效清单：圆点 pulse、shake 触发、Modal 触发延迟
- PRD 附录 D Pass 4 Pre-mortem bug 1：skip Modal 与 framer-motion focus trap 冲突 → `setTimeout(0)` 缓解
- PRD 附录 C Pass 3：键盘监听过滤 input / textarea / select 焦点

## 实施要点

- 顶部 header bar 三段：
  - 左：「← 上一步」按钮，`nav.prev` 回调；step 1 时隐藏（通过当前路径判断）
  - 中：StepIndicator（已完成圆点可点击触发 `nav.jumpTo`）
  - 右：「跳过引导」文字链，触发 `nav.skip`
- 标题区：保持现有 title + description props 渲染风格
- 内容区：用 `OnboardingPageTransition` 包装 `children`，`direction` 由 `nav` 推导（对比上一次 step 和当前 step）
- 挂载 `OnboardingBackground`，传 currentStep
- 全局 `useEffect` 监听 `window.keydown`：
  - `e.target` 是 INPUT / TEXTAREA / SELECT 时直接 return
  - ArrowLeft → `nav.prev()`
  - ArrowRight → `nav.next()`，失败时触发 `shake` ref（传给子组件）
  - Escape → `nav.skip()`
- skip 触发逻辑：`setTimeout(0)` 清空 framer-motion 动画队列后调 `Modal.confirm`，`onOk` 返回 mutation promise 让 antd 管理 loading，失败 re-throw 阻止 Modal 关闭
- StepIndicator 组件改造：
  - 圆点支持 `onClick` prop
  - 当前圆点添加 `animation: pulse` CSS keyframe
  - `useReducedMotion` 为 true 时 pulse 关闭
- routes.tsx 的 `OnboardingShell` 组件外层包一层 `<OnboardingStateProvider>`

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 手动访问 `/onboarding/categories`：
   - 左上角「← 上一步」可见，点击回到 welcome
   - 右上角「跳过引导」可见，点击弹 Modal
   - Modal 「确定」触发 skip mutation，成功后跳 dashboard
   - Modal 「取消」关闭 Modal，停留在当前 step
3. 键盘 ← 回到上一步
4. 键盘 → 在校验通过时前进
5. 键盘 Esc 弹出 skip Modal
6. 在 input 里按 ← / → 不触发导航（光标移动）
7. 当前 step 的圆点有 pulse 动画
8. 点击已完成步骤的圆点可回退
9. 单元测试覆盖键盘事件过滤、skip Modal 打开关闭、direction 推导

## 依赖说明

前置：step07（StateProvider + nav hook）、step08（Background）、step09（PageTransition）
