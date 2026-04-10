# 执行计划 V2 PRD

## 1 问题陈述

当前决策卡的执行计划存在两个核心缺陷：

1. **monitor 类型（hold 建议）只展示止损/止盈价格**，没有可操作的步骤。用户看到"止损: 2.85 / 止盈: 3.30"后不知道什么时候该行动、行动幅度多大。
2. **非 monitor 类型的步骤 rationale 是自由文本**，LLM 输出质量不稳定，内容缺乏可预测的结构。

## 2 目标

- 每张决策卡（含 hold/monitor）都展示 1-5 条可执行步骤，每条步骤有明确的触发条件、仓位变动和结构化解释
- 每条步骤附带参考手数（lotCount），让用户可以直接对照下单
- 结构化 rationale 保证前端渲染的可预测性和一致性

## 3 设计决策记录

| 编号 | 问题 | 选项 | 决定 | 理由 |
|------|------|------|------|------|
| D1 | 改动范围 | A 仅前端 / B 前端+后端 prompt / C 全栈含数据模型 | C | 根本问题在后端数据结构和 LLM prompt |
| D2 | lotCount 计算位置 | A 纯前端 / B 纯后端 / C 后端算+前端校验 | C | 后端有 currentPrice 和 totalCapitalCny，前端可用 totalCapitalCny 做显示校验 |
| D3 | monitor 类型步骤 | A 保持无步骤 / B 生成 1-2 条件步骤+保留止损止盈 | B | 用户需要知道什么条件下应该行动 |
| D4 | rationale 结构 | A 自由文本 / B 5 个固定语义字段 | B | 可预测的前端渲染，降低 LLM 输出波动影响 |
| D5 | 历史数据处理 | A 不迁移 / B 惰性迁移 / C 批量重新生成 | C | 全量替换保证一致性 |
| D6 | 实现方式 | A 扩展 / B 版本化 / C 同名替换 | C | 旧 string rationale 信息价值低，直接替换无损失 |

## 4 数据模型变更

### 4.1 StructuredRationale（新增）

5 个固定语义字段，每个字段值由 LLM 生成的一句自然语言填充：

| 字段 | 语义 | 示例 |
|------|------|------|
| triggerReason | 为什么选择这个触发条件 | "技术面显示 MACD 金叉确认，突破压力位" |
| positionReason | 为什么是这个仓位变动幅度 | "当前仓位偏低，加 5% 可接近目标仓位" |
| precondition | 执行前必须满足的前提 | "需要成交量放大至 5 日均量以上" |
| fallback | 如果触发条件未命中的备选方案 | "若 3 日内未突破则降低触发价至 2.80" |
| timeWindow | 预计触发时间窗口 | "1-3 个交易日内" |

### 4.2 Step 字段变更

| 字段 | 原值 | 新值 |
|------|------|------|
| rationale | `string` | `StructuredRationale`（5 字段对象） |
| lotCount | 不存在 | `number`（新增，0 = 未计算/隐藏） |

### 4.3 Execution 约束变更

| 类型 | 原约束 | 新约束 |
|------|--------|--------|
| monitor | steps = nil/[] | steps = [1-2 条件步骤]，stopLoss/takeProfit 保留 |
| one-shot | steps = [1] | 不变 |
| staged | steps = [2-5] | 不变 |

## 5 lotCount 计算规则

- 公式：`lotCount = floor(totalCapitalCNY * abs(deltaPct) / 100 / latestPrice)`
- A 股额外约束：`lotCount = floor(rawCount / 100) * 100`（整手，100 股/手）
- 其他资产类型：`lotCount = floor(rawCount)`
- 缺失 totalCapitalCNY 或 latestPrice 时：`lotCount = 0`，前端不显示
- lotCount 不参与 fingerprint 计算（与 rationale 相同理由：totalCapitalCNY 可独立变化）
- 前端标签："参考手数"，仅在 `lotCount > 0` 时渲染

## 6 UI 渲染规格

### 6.1 ExecutionPlanStrip（卡片墙缩略）

- 所有类型统一显示步骤数和首步触发条件
- monitor 类型：首步触发条件 + 止损/止盈辅助信息
- 非 monitor 类型：首步触发条件 + deltaPct

### 6.2 ExecutionPlanFull（详情页完整）

每个步骤渲染为卡片，包含：
- 步骤编号 + triggerType 图标 + triggerValue + deltaPct（标题行）
- lotCount（参考手数，仅 > 0 时展示）
- StructuredRationale 各字段（仅非空字段渲染，空字段隐藏）
- monitor 类型步骤加 "(条件监控)" 标签，区别于 action 步骤

### 6.3 向后兼容

- 旧卡（string rationale）：前端 type guard `typeof rationale === 'string'`，将整个字符串渲染为 triggerReason 的位置
- 旧卡（monitor 无 steps）：保持原有 止损/止盈 文本展示
- 新旧格式同时存在不会导致 crash

## 7 LLM Prompt 变更

- 告知 LLM rationale 字段不再是字符串而是 5 字段对象
- monitor 类型必须生成 1-2 个条件步骤（触发类型为 price 或 event）
- 条件步骤的 deltaPct 通常为负值（减仓观察）或 0（纯监控信号）
- 每个 rationale 字段限制在一句话内

## 8 模板降级（template fallback）

- 非 hold 动作：保持原有单步骤 "execute immediately"，rationale 改为 StructuredRationale（仅 triggerReason 填充固定文案）
- hold 动作：生成 1 条条件步骤（价格跌破止损时减仓），rationale 填充固定中文文案
- lotCount = 0（template 路径不读 totalCapitalCNY）

## 9 迁移方案

- 利用已有 `POST /analysis/reanalyze-all` 端点触发全量重新分析
- 重新分析自动使用新 prompt，生成新格式数据
- 迁移失败的卡保持旧格式，下次定时任务自动重试
- 无需数据库 schema 变更（recommendation 存储在 JSONB 列，新字段自动兼容）

## 10 状态空间表

| # | type | steps | rationale | lotCount | 分类 | 说明 |
|---|------|-------|-----------|----------|------|------|
| S1 | monitor | [1-2] | Structured | >0 | Valid target | 完整体验（有 totalCapital） |
| S2 | monitor | [1-2] | Structured | 0 | Valid target | 无 totalCapital，隐藏手数 |
| S3 | monitor | [1-2] | 空 Structured | 0 | Valid target | template 降级 |
| S4 | monitor | nil/[] | string | 0 | Transient | 旧卡，兼容渲染 |
| S5 | one-shot/staged | [1-5] | Structured | >0 | Valid target | 完整体验 |
| S6 | one-shot/staged | [1-5] | Structured | 0 | Valid target | 无 totalCapital |
| S7 | one-shot/staged | nil/[] | string | 0 | Transient | 旧卡 |
| S8 | monitor | [] | Structured | 0 | Forbidden | 新格式 monitor 必须有步骤 |

## 11 Pre-mortem Bug 表

| # | 现象 | 根因 | 防御 |
|---|------|------|------|
| 1 | 旧卡页面崩溃 Cannot read property 'triggerReason' | string rationale 访问对象字段 | 前端 type guard |
| 2 | lotCount 显示 "0 手" | totalCapitalCNY 未设置 | `lotCount > 0` 时才渲染 |
| 3 | monitor 卡仍只显示止损/止盈 | LLM 忽略新 prompt 返回 steps:[] | ensureRecommendation 兜底 |
| 4 | 每次分析都触发 BadgePlanAdjust | fingerprint 包含 lotCount | fingerprint 排除 lotCount |
| 5 | template 降级卡片看起来空白 | 5 个 rationale 字段全空 | 前端仅渲染非空字段，全空时隐藏 rationale 区域 |
