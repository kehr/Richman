# Step 16: Asset Detail Page

> Phase 4 | 并行组 R9 (可与 Step 15, 17, 18 同时执行) | 前置: Step 14

## 任务目标

实现标的详情页（/market/:code）：StickyHeader（价格/评分/变化摘要/冲突警告/新鲜度）、三 Tab 结构（分析/风险/执行）、全部 19 个子组件、asset-detail feature 模块（6 hooks + 轮询），以及 portfolio feature 扩展（useHoldingByAssetCode + entryMode 字段）。

## 涉及文件

### 创建

**Feature 模块：**
- `frontend/src/features/asset-detail/api.ts`
- `frontend/src/features/asset-detail/types.ts`
- `frontend/src/features/asset-detail/use-asset-detail.ts`
- `frontend/src/features/asset-detail/use-asset-ohlcv.ts`
- `frontend/src/features/asset-detail/use-score-history.ts`
- `frontend/src/features/asset-detail/use-demo-plan.ts`
- `frontend/src/features/asset-detail/use-trigger-holding-analysis.ts`
- `frontend/src/features/asset-detail/use-analysis-job.ts` -- 轮询 hook (3s, max 60)
- `frontend/src/features/asset-detail/index.ts`

**页面 + 组件：**
- `frontend/src/pages/asset-detail/asset-detail-page.tsx`
- `frontend/src/pages/asset-detail/components/sticky-header.tsx`
- `frontend/src/pages/asset-detail/components/asset-identity.tsx`
- `frontend/src/pages/asset-detail/components/score-summary.tsx`
- `frontend/src/pages/asset-detail/components/change-summary.tsx`
- `frontend/src/pages/asset-detail/components/major-change-recap.tsx`
- `frontend/src/pages/asset-detail/components/conflict-warning.tsx`
- `frontend/src/pages/asset-detail/components/freshness-indicator.tsx`
- `frontend/src/pages/asset-detail/components/score-bar.tsx`
- `frontend/src/pages/asset-detail/components/analysis-tab.tsx`
- `frontend/src/pages/asset-detail/components/ohlcv-chart.tsx`
- `frontend/src/pages/asset-detail/components/interpretation-card.tsx`
- `frontend/src/pages/asset-detail/components/dimension-panel-list.tsx`
- `frontend/src/pages/asset-detail/components/score-trend-chart.tsx`
- `frontend/src/pages/asset-detail/components/risk-tab.tsx`
- `frontend/src/pages/asset-detail/components/risk-factor-list.tsx`
- `frontend/src/pages/asset-detail/components/key-price-levels.tsx`
- `frontend/src/pages/asset-detail/components/drawdown-reference.tsx`
- `frontend/src/pages/asset-detail/components/event-calendar.tsx`
- `frontend/src/pages/asset-detail/components/execution-tab.tsx`
- `frontend/src/pages/asset-detail/components/demo-plan-register-cta.tsx`
- `frontend/src/pages/asset-detail/components/demo-plan-add-holding-cta.tsx`
- `frontend/src/pages/asset-detail/components/full-execution-plan.tsx`

### 修改

- `frontend/src/features/portfolio/` -- 新增 use-holding-by-asset-code.ts + Holding 类型新增 entryMode
- `frontend/src/features/portfolio/index.ts` -- barrel 导出新 hook
- `frontend/src/i18n/locales/zh/asset-detail.json` -- 新增
- `frontend/src/i18n/locales/en/asset-detail.json`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| StickyHeader 结构 | SS5.2.1 Sticky Header | frontend SS5.2 |
| 分析 Tab | SS5.2.2 | frontend SS5.3 |
| 风险 Tab | SS5.2.3 | frontend SS5.4 |
| 执行 Tab (三态展示) | SS5.2.4 | frontend SS5.5 |
| Tab 加载策略 (懒加载) | - | frontend SS5.6 |
| OhlcvChart (lightweight-charts v4) | SS5.2.2 | frontend SS5.3 |
| ScoreTrendChart (echarts) | SS5.2.2 | frontend SS5.3 |
| DimensionPanelList (折叠面板) | SS5.2.2 | frontend SS5.3 |
| FreshnessIndicator (三级警告) | SS5.2.1 | frontend SS5.2 |
| 货币展示 + CNY 附注 USD | SS4.2.4 | frontend SS5.4 |
| Job 轮询 (3s, max 60) | - | frontend SS8.3 |
| SEO meta | - | frontend SS12.1 |
| v1/v2 决策卡片区分 | - | frontend SS16.3 |
| richson 503 降级 UI | - | frontend SS16.6 |

## 关键约束 + 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G3.3 v1/v2 决策卡片 | 展示区分（v2 有 scenarios/stopLoss 字段） | frontend SS16.3 |
| G3.6 richson 503 降级 | 分析数据不可用时 Tab 内显示 skeleton + retry | frontend SS16.6 |

- 公开页面，执行 Tab 内容根据登录/持仓状态展示不同内容
- OhlcvChart 使用 lightweight-charts v4 API（`createChart` + `addCandlestickSeries`）
- ScoreTrendChart 使用 echarts（折线图 + 竖线版本变更标记）
- Tab 懒加载：分析 Tab 默认展示，风险/执行 Tab 点击时加载
- destroyOnHidden: false（antd ^5.24 属性名，需 grep node_modules 确认）
- 条件展示规则：ChangeSummary (delta>=5), MajorChangeRecap (|delta|>20)
- FreshnessIndicator 三级：>2% 黄 / >5% 橙 / >10% 红
- CNY 标的价格位附注 USD 等价（`price * usdExchangeRate`）
- 维度面板 "?" 图标 hover 展示概念解释（i18n key）

## 验证标准

- [ ] `cd frontend && pnpm lint:all` 全部通过
- [ ] `pnpm build` 成功
- [ ] 浏览器访问 /market/GLD 页面正常渲染
- [ ] StickyHeader 滚动时固定顶部
- [ ] 三个 Tab 切换正常
- [ ] OhlcvChart 渲染 K 线 + SMA200 + 支撑/阻力位
- [ ] 未登录时执行 Tab 显示 DemoPlan + RegisterCTA
- [ ] 已登录无持仓时显示 DemoPlan + AddHoldingCTA
- [ ] 已登录有持仓时显示 FullExecutionPlan
- [ ] useAnalysisJob 轮询 3s 间隔正常工作

## 变更点清单覆盖

E3.2 (1), E1.2 (1), E2.1-E2.2 (2), E5.1-E5.19 (19), E11.2 (1), E12.3 (1), G3.3 (1), G3.6 (1) = **27 项**
