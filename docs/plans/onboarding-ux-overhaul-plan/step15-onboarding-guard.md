# Step 15 OnboardingGuard 语义扩展

## 任务目标

扩展 `OnboardingGuard` 支持跳过和重入两种新场景：跳过用户 (`skipped=true`) 可以放行 app shell，也可以重新访问 onboarding 路由（Dashboard nudge 重入路径）；只有 `completed=true` 的用户访问 onboarding 路由会被弹回 dashboard。

## 涉及文件

修改：
- `frontend/src/domain/auth/onboarding-guard.tsx`
- `frontend/src/domain/auth/onboarding-guard.test.tsx`

## 设计依据

- PRD §3.8 OnboardingGuard 语义扩展
- TRD §6.2 OnboardingGuard 三态放行（完整 useEffect 判断分支 + 三种状态的 redirect 规则）
- PRD 附录 C Pass 3 主路径 C：从 nudge 重入 + 中途退出不被反弹
- PRD 附录 A 组合 #5/#7：skipped 用户访问 app shell 合法
- 关键决策：重入时**不清 skipped_at**，guard 必须放行 skipped 用户访问 onboarding 路由

## 实施要点

- 读取 `useOnboardingStatus()` 的 `data.completed` 和 `data.skipped`
- 放行规则：
  - `completed && isOnboardingRoute` → 弹回 dashboard
  - `!completed && !skipped && !isOnboardingRoute` → 弹回 welcome（新用户强制引导）
  - `skipped && !completed && isOnboardingRoute` → **允许**（nudge 重入）
  - 其他情况 → 正常渲染 children
- 其余代码（loading state、isOnboardingRoute 计算）保持
- 测试覆盖 6 种组合：
  - 新用户访问 onboarding：通过
  - 新用户访问 dashboard：弹回 welcome
  - completed 用户访问 dashboard：通过
  - completed 用户访问 onboarding：弹回 dashboard
  - skipped 用户访问 dashboard：通过
  - skipped 用户访问 onboarding：**通过**（新行为）

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. `pnpm test -- --run src/domain/auth/onboarding-guard` 全部 6 个用例通过
3. 手动 smoke：
   - 新用户登录 → 自动到 welcome
   - 走完 onboarding → 跳 dashboard
   - 手动访问 `/onboarding/welcome` → 弹回 dashboard（completed 反向）
   - 跳过后访问 dashboard → 通过 + 看到 nudge（nudge 来自 step16）
   - 跳过后手动访问 `/onboarding/welcome` → 通过（不被反弹）

## 依赖说明

前置：step05（`OnboardingStatus` 类型已包含 `skipped` 字段）
