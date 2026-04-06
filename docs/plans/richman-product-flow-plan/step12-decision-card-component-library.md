# Step 12 决策卡组件库

## 任务目标

新建 `features/decision-card`，提供 4 个核心组件 + api / hook，作为 Dashboard 和决策卡详情页的共享底座。

## 涉及文件

创建：
- `frontend/src/features/decision-card/api.ts`（GET 列表 + GET 详情 + POST 重新分析）
- `frontend/src/features/decision-card/types.ts`（与后端 DTO 对齐）
- `frontend/src/features/decision-card/use-decision-cards.ts`
- `frontend/src/features/decision-card/use-decision-card-detail.ts`
- `frontend/src/features/decision-card/use-rerun-analysis.ts`
- `frontend/src/features/decision-card/components/ChangeBadge.tsx`
- `frontend/src/features/decision-card/components/ChangeBadge.test.tsx`
- `frontend/src/features/decision-card/components/DimensionBadges.tsx`
- `frontend/src/features/decision-card/components/DimensionBadges.test.tsx`
- `frontend/src/features/decision-card/components/ExecutionPlanStrip.tsx`
- `frontend/src/features/decision-card/components/ExecutionPlanStrip.test.tsx`
- `frontend/src/features/decision-card/components/DecisionCardSummary.tsx`
- `frontend/src/features/decision-card/components/DecisionCardSummary.test.tsx`
- `frontend/src/features/decision-card/index.ts`（barrel）

## 设计依据

- PRD §3.2 决策卡组件 6 区域
- PRD §3.3 执行计划摘要条
- PRD §3.4 8 种徽章状态机
- TRD §2.5 API DTO 字段
- TRD §7.3 features/decision-card 拆分

## 实施要点

- ChangeBadge：接受 badge_state prop，按 PRD §3.4 表格映射颜色和文案。无徽章时返回 null。每种状态都要可以单独 import 文案常量（便于 Help 页复用）
- DimensionBadges：接受三个 dimension 的 current 值 + 可选的 previous 值。翻转时显示绿/红底 + 箭头 + 划掉旧值
- ExecutionPlanStrip：
  - 接受 Recommendation.execution
  - type = staged：渲染前 3 步，超过 3 步显示 "+ 还有 N 批，查看详情 →"
  - type = monitor：渲染止损 / 止盈两条
  - type = one-shot：单步显示
  - 每步触发条件按 trigger_type 映射不同的小图标 / 文案样式
- DecisionCardSummary：
  - 整张 Dashboard 摘要卡，组合上面三个子组件 + 头部 + 建议框 + 今日要点 + 信心度
  - 接受 onClick 回调（Dashboard 用来跳详情页）
  - 内部调用 useMoney 显示金额
- 所有 antd 组件经 ui-kit/eat barrel 导入
- 测试：
  - ChangeBadge 8 种状态各渲染一次断言文案
  - DimensionBadges 翻转 / 不翻转两种快照
  - ExecutionPlanStrip 三种 type × 步骤数 1/3/5 × 持有止损止盈
  - DecisionCardSummary 含 / 不含金额两种渲染

## 验证标准

1. `pnpm test src/features/decision-card` 通过
2. `pnpm lint:all` 通过
3. 在 Dashboard 之外用一个 storybook-like 测试页加载这些组件确认视觉无错位（可选，如果项目已配 storybook）
4. dependency-cruiser 检查 features/decision-card 不依赖其他 feature

## 依赖说明

- 前置：step11 useMoney 必须先存在；step09 后端 DTO 必须已对齐

## 预估提交

- commit 1: `feat(decision-card): add api and hooks`
- commit 2: `feat(decision-card): add change badge and dimension badges`
- commit 3: `feat(decision-card): add execution plan strip`
- commit 4: `feat(decision-card): add summary card composing all sub components`
