# Step 6: Pre-market Information Delta

**依赖：** Step 3（service 层）
**可与 Step 5/7/8 并行**
**设计依据：** PRD §盘前分析内容、TRD §盘前信息增量

## 任务目标

在盘前分析触发时，额外收集并注入自上次分析以来的价格变动和新闻摘要到分析 prompt context。

## 涉及文件

- 修改：`backend/internal/service/analysis/scheduler.go`（新增 `runPreWindowJob` 函数或在现有 `runJob` 中分支处理）
- 修改：`backend/internal/service/analysis/`（调研现有 prompt 构建逻辑，找到 context 注入点）

## 执行步骤

- [ ] 阅读 `backend/internal/service/analysis/` 下所有文件，找到：prompt 构建函数、数据拉取函数（OHLCV、新闻）、分析任务的入参结构
- [ ] 在 scheduler 的触发判断中，区分 `isPreWindow bool` 参数（盘前/盘后），传入 `processUserJob`
- [ ] 在 `processUserJob` 中，若 `isPreWindow=true`：
  - 查询该持仓的 `last_analyzed_at`（已有字段或从 decision_cards 最新记录获取）
  - 拉取 last_analyzed_at 至 now 的区间 OHLCV 数据（复用现有数据管道函数）
  - 将价格变动（区间涨跌幅、高低点）格式化为文本，prepend 到 prompt context
  - 若新闻数据管道已有相关函数则一并注入；若无则跳过（不阻塞分析流程）
- [ ] 确保盘后分析（`isPreWindow=false`）行为与现有完全一致，无副作用
- [ ] `cd backend && make check` 通过
- [ ] `git add backend/internal/service/analysis/ && git commit -m "feat(analysis): inject price delta context in pre-window analysis"`

## 验证标准

- `make check` 通过
- 盘后分析逻辑无变化（走读确认）
- 盘前分析路径中有区间 OHLCV 数据注入（走读确认）
- 若数据拉取失败，分析任务仍继续（不因信息增量缺失而中断）
