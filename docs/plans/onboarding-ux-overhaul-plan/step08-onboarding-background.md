# Step 08 OnboardingBackground 装饰层组件

## 任务目标

新建 `OnboardingBackground` 组件提供三层装饰：细网格、漂移 radial glow、仅 Welcome 页显示的自转光环 hero。组件响应 `useReducedMotion` 降级静态呈现。

## 涉及文件

创建：
- `frontend/src/pages/onboarding/components/OnboardingBackground.tsx`

可能修改：
- 无（CSS 全部 inline 或 `<style>` 组件内）

## 设计依据

- PRD §3.4 OnboardingBackground 组件 / §4.1 色彩规范 / §4.4 reduced motion 降级
- TRD §5.2 OnboardingBackground（完整 Grid + Glow + RingHero 三层 CSS 与 framer-motion variants）
- PRD 附录 D Pass 4 Pre-mortem bug 3：光环 CSS `conic-gradient + mask-composite` 的 iPad 合成层问题 → `will-change: transform` 仅 mount 时启用

## 实施要点

- 组件接受 prop `currentStep: 1 | 2 | 3 | 4`，仅 `currentStep === 1` 时渲染光环 hero
- 三层 DOM 结构：外层 fixed 覆盖整 viewport，内部从下到上：
  - 层 1 grid：CSS `background-image: linear-gradient(to right, #0000000a 1px, transparent 1px)` 横纵叠加，间距 64px
  - 层 2 glow：`position: absolute inset: 0` + 大 radial-gradient `#00000008` 中心 → 透明，`animation: glow-drift 90s ease-in-out infinite`
  - 层 3 ring hero：仅 step 1 时渲染，120px 圆环，conic-gradient stroke + mask-composite 实现发光弧段，`framer-motion motion.div animate={{ rotate: 360 }} transition={{ duration: 30, repeat: Infinity, ease: "linear" }}`
- 中心 Richman R logo 用 `<img src="/logo.svg" />` 静态居中
- `useReducedMotion()` hook 返回 true 时：glow 不漂移、ring 不自转；grid 不变
- `will-change: transform` 只加在 ring 上，避免全组件 GPU 层
- 样式全部 inline 或用 `<style>` 组件内注入（避免全局 CSS 污染）

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 组件单元测试：
   - 渲染后 DOM 包含 grid / glow / ring 三层
   - `currentStep !== 1` 时 ring 不渲染
   - `useReducedMotion` mock 返回 true 时，motion.div 的 animate prop 退化
3. 手动在 dev server 访问 onboarding，观察：
   - 背景细网格清晰但不干扰
   - radial glow 极淡但可感（眯眼能看到位移）
   - step 1 看到光环 + R logo 匀速自转
4. 在 Chrome DevTools 的 Rendering 面板启用「Paint flashing」，确认只有 ring 组件有频繁重绘

## 依赖说明

前置：step06（framer-motion 已安装）
