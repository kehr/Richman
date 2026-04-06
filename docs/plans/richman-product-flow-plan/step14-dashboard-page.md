# Step 14 Dashboard 页面三区结构

## 任务目标

把现有的 DashboardPage 重写为 PRD §3.1 定义的三区结构：顶部瘦 strip（含重新分析按钮）+ 中部决策卡墙 + 底部变化锚点摘要。

## 涉及文件

修改：
- `frontend/src/pages/dashboard/DashboardPage.tsx`

创建：
- `frontend/src/pages/dashboard/components/DashboardTopStrip.tsx`
- `frontend/src/pages/dashboard/components/DashboardTopStrip.test.tsx`
- `frontend/src/pages/dashboard/components/DecisionCardWall.tsx`
- `frontend/src/pages/dashboard/components/ChangeAnchorList.tsx`
- `frontend/src/pages/dashboard/components/ChangeAnchorList.test.tsx`
- `frontend/src/pages/dashboard/components/EmptyHoldingsHero.tsx`
- `frontend/src/pages/dashboard/DashboardPage.test.tsx`

## 设计依据

- PRD §3.1 三区结构与字段
- PRD §3.2 决策卡组件
- PRD §3.3 执行计划摘要条
- TRD §7.1 §7.3 前端模块组织

## 实施要点

- DashboardPage 是组合页，本身不写业务逻辑，只串起 4 个子组件
- DashboardTopStrip：
  - 第一行：左侧标题 + 副标题、右侧"⟳ 重新分析"主按钮（调 useRerunAnalysis）
  - 第二行：4 列总览（持仓数 / 总资金 / 综合浮盈亏 / 已分配仓位）
  - 总资金未设置时该列改为"设置以查看"链接，点击跳 /settings → 账户 tab（用 location state 携带 highlight 参数）
  - 通过 useMoney 渲染金额，通过 useUserSettings 拿 totalCapitalCny
- DecisionCardWall：
  - 用 features/decision-card 的 useDecisionCards hook 拉数据
  - 响应式 grid：3 列 / 2 列 / 1 列
  - 渲染 DecisionCardSummary，onClick 跳 /decision-cards/:id
  - loading 时骨架屏，error 时错误提示
- ChangeAnchorList：
  - 过滤出 badge_state != none 的卡
  - 每条一行 "● [徽章颜色] [标的名] → [变化摘要]"
  - 点击 anchor 滚动到 DecisionCardWall 中对应卡，scrollIntoView + 1.5s 高亮（用 ref + classList 临时加 highlight 类）
  - 全部卡都无变化时整个组件不渲染
- EmptyHoldingsHero：
  - 持仓为 0 时整页只渲染这个组件，DashboardTopStrip 和 DecisionCardWall 不显示
  - 大号引导卡 "先添加一个持仓 →"，按钮跳 /portfolio
- 重新分析按钮处理 loading / disabled 状态，分析进行中时禁用并显示进度

## 验证标准

1. `pnpm test src/pages/dashboard` 通过
2. 三种状态下视觉正确：0 持仓 / 持仓但首次分析未完成 / 多卡含变化
3. 重新分析按钮触发后能看到加载态 + 分析完成后卡片自动刷新
4. ChangeAnchorList 点击能滚动并高亮对应卡
5. 总资金未设置时所有金额位置不显示 "¥ 0"，而是只显示百分比
6. `pnpm lint:all` 通过

## 依赖说明

- 前置：step12 决策卡组件库、step11 useMoney、step09 后端 DTO 已对齐

## 预估提交

- commit 1: `refactor(dashboard): build top strip with rerun action`
- commit 2: `feat(dashboard): add decision card wall and change anchor list`
- commit 3: `feat(dashboard): add empty holdings hero state`
