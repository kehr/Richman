# Step 11 WelcomePage 接入 nav hook 与 stagger 进场

## 任务目标

将 `WelcomePage` 从直接 `useNavigate` 迁移到 `useOnboardingNav`，标题、副标题、三张维度卡片用 framer-motion 做 stagger fade-up 进场动画，保持现有视觉结构，加强视觉生动感。

## 涉及文件

修改：
- `frontend/src/pages/onboarding/WelcomePage.tsx`
- `frontend/src/pages/onboarding/WelcomePage.test.tsx`

## 设计依据

- PRD §3.6 页面改造 - WelcomePage
- PRD §4.3 动效清单：stagger 80ms 间隔、fade-up 0.4s each

## 实施要点

- 替换 `useNavigate` 为 `useOnboardingNav`
- 「开始设置」按钮的点击改为 `nav.next()`
- 主内容区用 `motion.div` 的 `initial / animate / variants` 实现 stagger：
  - container variant 用 `staggerChildren: 0.08`
  - 子元素 variant 从 `opacity: 0, y: 20` 到 `opacity: 1, y: 0`，duration 0.4
- 三张维度卡片也是 stagger 的子元素
- `useReducedMotion` 为 true 时降级为无 y 位移、无 stagger（同时显示）
- 页面注册 `canGoNext` 始终返回 true（Welcome 页无校验）
- OnboardingBackground 的光环 hero 已在 step10 由 OnboardingLayout 根据 step 1 渲染，本 step 不重复
- 既有测试断言保留（title / 三张卡 / CTA 按钮存在），新增断言不挂掉

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. `pnpm test -- --run src/pages/onboarding/WelcomePage` 通过
3. 手动访问 `/onboarding/welcome`：
   - 标题、副标题、三张卡片依次进场（肉眼可见 stagger 效果）
   - 点「开始设置」前进到 categories
   - 开启系统 prefers-reduced-motion 后进场动画降级为同时显示
4. 构建无 TypeScript 错误

## 依赖说明

前置：step10（OnboardingLayout 已支持 nav hook）
