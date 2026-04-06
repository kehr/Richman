# Step 15 决策卡详情页

## 任务目标

实现 PRD §5 决策卡详情页：左主内容（5 区块）+ 右侧固定 meta 栏，完整呈现一张卡的推理。

## 涉及文件

修改：
- `frontend/src/pages/decision-cards/DecisionCardDetailPage.tsx`

创建：
- `frontend/src/pages/decision-cards/components/CardHero.tsx`
- `frontend/src/pages/decision-cards/components/ConclusionBanner.tsx`
- `frontend/src/pages/decision-cards/components/ExecutionPlanFull.tsx`
- `frontend/src/pages/decision-cards/components/DimensionReasoning.tsx`
- `frontend/src/pages/decision-cards/components/MainRisks.tsx`
- `frontend/src/pages/decision-cards/components/MetaSidebar.tsx`
- `frontend/src/pages/decision-cards/components/MetaSidebar.test.tsx`
- `frontend/src/pages/decision-cards/DecisionCardDetailPage.test.tsx`

## 设计依据

- PRD §5 决策卡详情页 5 区块 + meta 栏
- TRD §2 Recommendation 数据模型（详情页消费完整 step.rationale）
- TRD §7.3 features/decision-card 复用 ChangeBadge / DimensionBadges

## 实施要点

- DecisionCardDetailPage：
  - 通过 useDecisionCardDetail hook 拉数据（含 prev_card_id，用于"此标的历史分析"）
  - 顶部面包屑 ← Dashboard / 决策卡 / 标的名 · 时间
  - 主内容区域 + 右侧 sidebar 用 grid 布局，1024px 以下 sidebar 折叠到顶部
- CardHero：复用 features/decision-card 的 ChangeBadge；显示成本/现价/仓位/浮盈亏（使用 useMoney）
- ConclusionBanner：
  - 大字建议 + 目标仓位叙述 + 与上次对比小字
  - 右侧大字信心度 + delta
  - 左侧 4px 色边按 badge_state 着色
- ExecutionPlanFull：
  - 接受 Recommendation.execution
  - 渲染所有 steps（不限于 3）
  - 每步独立卡片：编号 + 触发条件 + 变动仓位 + rationale 文本（rationale 是后端解析过的 markdown 字符串，用 react-markdown 渲染）
  - 持有场景显示 stop_loss / take_profit 两条
  - 底部黄色提示条显示 valid_days
- DimensionReasoning：
  - 三个维度独立卡片
  - 翻转的维度加彩色边框 + 浅底 + 📌 注释
  - 每个维度展示 quantitative signals（来自 raw_data JSONB）+ 权重微调轨迹 + 文字结论
  - 使用 features/decision-card 的 DimensionBadges 组件
- MainRisks：黄色告警底，列出 risk_warnings JSONB 数组每条
- MetaSidebar：
  - 分析时间 + 时区
  - 数据源逐条状态（降级时红色）
  - 下次自动分析时间
  - 此标的历史分析列表（最近 3-5 次，链接到对应 card detail）
  - 风险声明（统一一次）
- 所有组件遵守 ui-kit/eat barrel 导入

## 验证标准

1. `pnpm test src/pages/decision-cards` 通过
2. 浏览器中点 Dashboard 卡能跳到详情页，所有 5 区块 + meta 栏正确渲染
3. 分批 / 一次性 / 持有 三种 execution type 都正确渲染
4. 数据源降级的卡显示红色提示
5. 推送链接 /decision-cards/:id 在未登录情况下跳到 /login 后能登录回流到原页
6. `pnpm lint:all` 通过

## 依赖说明

- 前置：step12 决策卡组件库、step11 useMoney、step09 后端 DTO

## 预估提交

- commit 1: `feat(card-detail): add hero and conclusion banner`
- commit 2: `feat(card-detail): add full execution plan with rationale`
- commit 3: `feat(card-detail): add dimension reasoning and risks`
- commit 4: `feat(card-detail): add meta sidebar with history`
