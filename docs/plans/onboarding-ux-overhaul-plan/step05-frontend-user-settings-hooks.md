# Step 05 前端 user-settings hooks 同步后端契约

## 任务目标

同步前端 `OnboardingStatus` 类型和 `User` 类型以反映后端新增的 `skippedAt` 字段，新增 `useSkipOnboarding` mutation hook，扩展 `useResetOnboarding` 的缓存失效范围和 localStorage / sessionStorage 清理。

## 涉及文件

修改：
- `frontend/src/features/user-settings/types.ts`
- `frontend/src/features/user-settings/api.ts`
- `frontend/src/features/user-settings/index.ts`
- `frontend/src/features/user-settings/use-reset-onboarding.ts`
- `frontend/src/domain/auth/types.ts`

创建：
- `frontend/src/features/user-settings/use-skip-onboarding.ts`

## 设计依据

- PRD §3.11 useResetOnboarding 扩展 / §3.12 useSkipOnboarding / §7 实施顺序
- TRD §6.1 前端 user-settings hook 契约（OnboardingStatus 类型 + useSkipOnboarding / useResetOnboarding 的完整 onSuccess 逻辑）
- 附录 B Pass 2：useResetOnboarding 失效范围不完整是已识别的契约 gap

## 实施要点

- `OnboardingStatus` 接口追加 `skipped: boolean` + `skippedAt?: string | null`，字段顺序紧贴 `completed` / `completedAt`
- `domain/auth/types.ts` 的 `User` 接口追加 `onboardingSkippedAt?: string | null`，与后端 model.User JSON 输出对齐
- `api.ts` 新增 `skipOnboarding(): Promise<ApiResponse<OnboardingStatus>>` 方法
- `use-skip-onboarding.ts` 新建 `useSkipOnboarding` hook，`onSuccess` 按顺序执行：
  1. `sessionStorage.removeItem("richman_onboarding_draft")`
  2. `invalidateQueries(ONBOARDING_STATUS_QUERY_KEY)`
  3. `invalidateQueries(["auth", "me"])`
  4. `refetchQueries(ONBOARDING_STATUS_QUERY_KEY)` 强制等新数据
- `use-reset-onboarding.ts` 扩展 `onSuccess`：同时 invalidate `["auth", "me"]`，清 `localStorage.removeItem("richman_onboarding_nudge_dismissed")` 和 `sessionStorage.removeItem("richman_onboarding_draft")`
- `index.ts` 导出新 hook

## 验证标准

1. `cd frontend && pnpm lint:all` 通过（Biome + tsc + dep-cruiser）
2. 既有测试全部通过
3. 手动调用 `useSkipOnboarding().mutateAsync()` 后，开发者工具 react-query devtools 看到 `onboarding-status` 和 `["auth","me"]` 都被 invalidate 并 refetch
4. 手动调用 `useResetOnboarding().mutateAsync()` 后，`localStorage.getItem("richman_onboarding_nudge_dismissed")` 返回 null，sessionStorage 同理

## 依赖说明

前置：step04（后端 `POST /onboarding/skip` 已实现，否则 FE 测试无法跑通）
