# Step 07 OnboardingStateProvider 与导航 hook

## 任务目标

新建 `OnboardingStateProvider` 集中管理 4 个 onboarding 页面共享的草稿状态（categories、holdingDraft、reachedStep、analysisFired），用 sessionStorage 持久化保证刷新不丢失。新建 `useOnboardingNav` hook 提供 `prev / next / skip / jumpTo / canGoNext / registerCanGoNext` 统一 API。Provider 和 hook 暂不接入页面，只保证独立可用。

## 涉及文件

创建：
- `frontend/src/pages/onboarding/state.tsx`（Context + Provider + `useOnboardingState`）
- `frontend/src/pages/onboarding/use-onboarding-nav.ts`（`useOnboardingNav` hook）

## 设计依据

- PRD §3.1 OnboardingStateProvider / §3.2 useOnboardingNav hook
- TRD §4.1 OnboardingState 数据结构 + §4.2 Provider 初始化顺序 + §4.3 Context 导出 + §4.4 useOnboardingNav 契约
- PRD 附录 A 组合 #3：返回用户场景下 provider 初始化要从后端读 categories
- PRD 附录 D Pass 4 Pre-mortem bug 2：Provider mount 检查 completed/skipped 主动清 sessionStorage
- PRD 附录 A 组合 #3 级联清理：categories 收缩时同步清 holdingDraft 的 asset 字段

## 实施要点

- `OnboardingState` 类型定义参考 PRD §3.1，包括 categories / holdingDraft / reachedStep / analysisFired
- sessionStorage key 固定为 `richman_onboarding_draft`
- Provider 初始化顺序：
  1. 读 `useOnboardingStatus()`；如果 completed 或 skipped 任一为 true，清 sessionStorage 用默认 state
  2. 否则 try/catch 读 sessionStorage，失败降级为默认 state
  3. 读 `useUserSettings()` 的 categories，如果与 sessionStorage 不一致以后端为准并写回
- state 变更时 throttled（500ms 防抖）写回 sessionStorage
- 级联清理：Provider 监听 categories 变化，发现 `holdingDraft.assetType` 不在新 categories 里时自动清空 holdingDraft 的 asset 相关字段
- `useOnboardingNav`：
  - 内部用 `useNavigate` + `useLocation` 计算当前 step
  - `prev()` 用 `navigate(prevPath, { replace: true })`
  - `next()` 先校验所有已注册的 `canGoNext` 谓词，全部 true 才前进；否则触发全局 shake 事件（具体如何触发 shake 在 step10 OnboardingLayout 中实现）
  - `skip()` 调 `useSkipOnboarding().mutateAsync()` + navigate，失败 toast
  - `jumpTo(n)` 要求 `n <= reachedStep`，否则 no-op
  - `registerCanGoNext` 让页面注册校验函数，返回 cleanup 函数
- Provider 在 `OnboardingShell` 路由外层挂载（具体挂载位置 step10 处理）

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. 新增 provider / hook 的单元测试：
   - sessionStorage 读写、跨 render 保留
   - completed=true 时 mount 清空 sessionStorage
   - categories 收缩时级联清理 holdingDraft
   - `jumpTo(超过 reachedStep)` 不执行
   - try/catch 包 sessionStorage 调用，storage disabled 时降级为内存 state
3. `pnpm test -- --run` 全部通过

## 依赖说明

前置：step05（需要 `useSkipOnboarding`、`useOnboardingStatus`、`OnboardingStatus` 类型）
