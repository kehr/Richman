# Step 09 OnboardingPageTransition 组件

## 任务目标

新建 `OnboardingPageTransition` 组件封装 framer-motion 的 `AnimatePresence` + `motion.div`，提供方向感知的前进/回退过渡动画。后续页面通过这个组件统一包装内容区实现滑动切换。

## 涉及文件

创建：
- `frontend/src/pages/onboarding/components/OnboardingPageTransition.tsx`

## 设计依据

- PRD §3.5 OnboardingPageTransition
- PRD §4.3 动效清单：page transition 前进 `x:40→0, opacity:0→1`，回退方向相反，duration 0.35s ease-out
- PRD §4.4 reduced motion：降级为 opacity-only

## 实施要点

- 组件 props：`direction: "forward" | "backward"` + `stepKey: string` + `children`
- 内部用 `<AnimatePresence mode="wait">` 包装，`motion.div` 的 `key` 绑定到 stepKey 保证页面切换触发 exit + enter
- `initial / animate / exit` variants 根据 direction 动态构造
- `useReducedMotion()` 为 true 时 variants 退化为 `{ opacity: 0 }` / `{ opacity: 1 }`，不做 x 位移
- 不依赖外部 state 库，direction 由父组件 OnboardingLayout 传入（step10 计算）
- 组件内导出 variants 常量方便单元测试断言

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 组件单元测试：
   - 传 direction="forward" 时 motion.div 的 initial prop 包含 `x: 40`
   - 传 direction="backward" 时 initial prop 包含 `x: -40`
   - `useReducedMotion` mock 返回 true 时 initial prop 只有 `opacity: 0`，没有 x
   - stepKey 变化时触发 AnimatePresence 的 exit 动画（通过 `mode="wait"` 验证 key 变化行为）
3. 构建无 TypeScript 错误

## 依赖说明

前置：step06（framer-motion 已安装）。可与 step08 并行（两个组件互不依赖）
