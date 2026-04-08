# Step 12 CategoriesPage 接入 state + stagger 选中动画

## 任务目标

将 `CategoriesPage` 改造为从 `OnboardingStateProvider` 读写 categories，4 张类型卡片 stagger 进场，点选时有 scale + 黑边闪烁动画。保持现有 `usePatchUserSettings({ categories })` 逐步 PATCH 后端的语义。注册 `canGoNext = categories.length >= 1`。

## 涉及文件

修改：
- `frontend/src/pages/onboarding/CategoriesPage.tsx`
- `frontend/src/pages/onboarding/CategoriesPage.test.tsx`

## 设计依据

- PRD §3.6 页面改造 - CategoriesPage
- PRD §4.3 动效清单：stagger 80ms、scale 1→1.02 点选动画

## 实施要点

- `useOnboardingState` 读写 categories，`useOnboardingNav` 提供 next + registerCanGoNext
- 替换本地 `useState<string[]>([])` 为 state provider 的 categories
- mount 时 `registerCanGoNext(() => state.categories.length >= 1)`，unmount 自动注销
- 「下一步」按钮保持调 `usePatchUserSettings({ categories })`，mutation 成功后再 `nav.next()`
  - 幂等：回退再前进时 categories 未变仍会重复 PATCH，无副作用
- 4 张卡片用 `motion.div` stagger 进场（同 WelcomePage）
- 点选时子卡片 `whileTap={{ scale: 1.02 }}` + `animate` 切换黑边
- `useReducedMotion` 降级：无 stagger、无 scale 动画，点选仍切换黑边
- 如果 state.categories 非空（例如用户从返回场景进入），页面初次渲染时卡片应反映已选状态

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. `pnpm test -- --run src/pages/onboarding/CategoriesPage` 通过
3. 手动访问 `/onboarding/categories`：
   - 4 张卡片 stagger 进场
   - 选中一张后「下一步」按钮从 disabled 变 enabled
   - 点「下一步」调 PATCH + 前进到 first-holding
   - 回退到 categories 页时，之前选过的卡片保持选中
4. 清空选择后「下一步」重新变 disabled

## 依赖说明

前置：step10（OnboardingLayout）+ step07（state provider 和 nav hook）
