# Step 11 金额换算 Hook 与 user-settings feature

## 任务目标

新建 `domain/money` 模块和 `features/user-settings` feature，为后续所有需要"百分比 + 金额"双显示的页面提供基础设施。

## 涉及文件

创建：
- `frontend/src/domain/money/useMoney.ts`
- `frontend/src/domain/money/format.ts`（纯格式化函数，便于单测）
- `frontend/src/domain/money/format.test.ts`
- `frontend/src/features/user-settings/api.ts`（GET / PATCH /api/v1/user/settings）
- `frontend/src/features/user-settings/use-user-settings.ts`（TanStack Query hook）
- `frontend/src/features/user-settings/use-onboarding-status.ts`（TanStack Query hook）
- `frontend/src/features/user-settings/index.ts`（barrel）
- `frontend/src/features/user-settings/types.ts`

修改：
- `frontend/src/domain/auth/onboarding-guard.tsx`（替换 step10 的占位 hook 为本 step 的正式 hook）

## 设计依据

- TRD §7.4 金额换算 hook 设计
- TRD §5.3 后端已附加 *Amount 字段，前端只负责展示
- PRD §8 总资金功能
- 工程规范 `docs/standards/frontend.md` features 隔离

## 实施要点

- useMoney 返回 3 个工具：
  - hasCapital: bool
  - format(pct, amount?): 设置了总资金且 amount 不为空 → "X% · ¥Y"；否则 → "X%"
  - formatAmountOnly(amount?): 仅金额，无金额时返回 null
- 数字格式化遵循 PRD §6.4 偏好（千分位分隔符等），从 user_settings 读偏好
- features/user-settings 严格遵守 features 不互相依赖原则
- api.ts 的请求 / 响应类型与 TRD §5.3 §6.1 对齐
- TanStack Query 配置：
  - queryKey: ['user-settings'] / ['onboarding-status']
  - staleTime 短（10s 内不重复请求）
  - 任何 PATCH 后 invalidate user-settings
- 单元测试覆盖 format 函数：
  - 总资金未设置 → 只返回百分比
  - 总资金已设置但 amount 为 null → 只返回百分比
  - 总资金已设置且 amount 有值 → 返回 "X% · ¥Y"
  - 大数字千分位正确

## 验证标准

1. `pnpm test src/domain/money` 通过
2. `pnpm lint:all` 通过
3. 浏览器中通过 React DevTools 查看 useMoney 返回值在切换 total_capital 后正确更新
4. dependency-cruiser 检查 features/user-settings 不依赖其他 feature

## 依赖说明

- 前置：step09（user_settings API 必须先存在）

## 预估提交

- commit 1: `feat(domain/money): add money formatting hook and pure utils`
- commit 2: `feat(features/user-settings): expose settings and onboarding hooks`
