# Step 17 Settings AccountTab 重入 CTA 投放生产

## 任务目标

`AccountTab` 中既有的「重置 Onboarding」按钮从 dev-only 门控改为所有环境可见，文案调整为「重新走一遍引导」，复用已扩展的 `useResetOnboarding`（清 sessionStorage + localStorage + 失效多个 query）。Popconfirm 二次确认防误触。

## 涉及文件

修改：
- `frontend/src/pages/settings/tabs/AccountTab.tsx`
- `frontend/src/pages/settings/tabs/AccountTab.test.tsx`

## 设计依据

- PRD §3.10 AccountTab 改造
- TRD §6.5 AccountTab 重入 CTA（完整 Popconfirm + mutation + navigate 序列）
- PRD 附录 C Pass 3 主路径 D：Settings 重入 + dismissed 标记清理
- 本 CTA 是 nudge dismissed 后的第二条 regret 路径，保证非 dev 用户也能回到 onboarding

## 实施要点

- 去掉 `import.meta.env.DEV` 条件渲染，按钮在所有环境可见
- 按钮文案：「重新走一遍引导」
- 放在 AccountTab 的第一个或第二个可见区块（接近顶部，易找）
- `Popconfirm` 二次确认：
  - title: 「确认重新走引导吗？」
  - description: 「将清空当前引导状态并从头开始。当前持仓和决策卡不受影响。」
  - 「确认」按钮：调 `useResetOnboarding().mutateAsync()` → 成功后 `navigate("/onboarding/welcome")`
  - 「取消」按钮：关闭 popconfirm
- Loading 状态由 `useResetOnboarding.isPending` 驱动
- 失败时 antd message.error 提示
- 测试覆盖：
  - 按钮始终可见（dev + prod）
  - 点击弹 Popconfirm
  - 确认后调 mutation + navigate
  - mutation 失败显示 error toast

## 验证标准

1. `cd frontend && pnpm lint:all` 通过
2. `pnpm test -- --run src/pages/settings/tabs/AccountTab` 通过
3. 手动 smoke：
   - 生产构建 (`pnpm build`) 的 AccountTab 能看到按钮
   - 点击弹 Popconfirm
   - 确认 → 跳 welcome
   - 验证 localStorage 的 `richman_onboarding_nudge_dismissed` 被清
   - 验证 sessionStorage 的 `richman_onboarding_draft` 被清

## 依赖说明

前置：step05（`useResetOnboarding` 已扩展失效范围和 storage 清理）
