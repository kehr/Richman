# 执行计划 V2 TRD

对应 PRD: `docs/prds/execution-plan-v2-prd.md`

## 1 后端类型变更

### 1.1 recommendation/types.go

新增 `StructuredRationale` 类型，修改 `Step` 结构体：

```go
// StructuredRationale holds the structured explanation for a single step.
// Each field is a short sentence from the LLM. Empty string = not provided.
type StructuredRationale struct {
	TriggerReason  string `json:"triggerReason"`
	PositionReason string `json:"positionReason"`
	Precondition   string `json:"precondition"`
	Fallback       string `json:"fallback"`
	TimeWindow     string `json:"timeWindow"`
}

type Step struct {
	Order          int                 `json:"order"`
	TriggerType    TriggerType         `json:"triggerType"`
	TriggerValue   string              `json:"triggerValue"`
	TriggerPayload TriggerPayload      `json:"triggerPayload"`
	DeltaPct       float64             `json:"deltaPct"`
	LotCount       float64             `json:"lotCount"`
	Rationale      StructuredRationale `json:"rationale"`
}
```

JSONB 兼容性：旧卡的 `"rationale": "some string"` 在反序列化到 `StructuredRationale` 时会得到零值（所有字段为空字符串），不会报错。Go 的 `encoding/json` 对类型不匹配的字段静默跳过。

### 1.2 recommendation/fingerprint.go

当前 fingerprint 已排除 `Rationale`。新增的 `LotCount` 同样不参与 fingerprint 计算（totalCapitalCNY 可独立变化，不应触发 BadgePlanAdjust）。

当前 step 哈希行：`step|%d|%s|%s|%.6f` 对应 `order|triggerType|triggerValue|deltaPct`。保持不变。

唯一变更：更新注释，显式说明 LotCount 与 Rationale 一样被排除。

## 2 LLM Prompt 变更

### 2.1 recommendation_prompt.go - recommendationPromptSection()

将 step 的 rationale 字段从字符串改为对象：

```
"steps": [
  {
    "order": 1,
    "triggerType": "price|time|event",
    "triggerValue": "short condition text",
    "deltaPct": 5.0,
    "rationale": {
      "triggerReason": "why this trigger condition (1 sentence)",
      "positionReason": "why this delta size (1 sentence)",
      "precondition": "what must be true before acting (1 sentence)",
      "fallback": "what to do if trigger missed (1 sentence)",
      "timeWindow": "expected timeframe (1 sentence)"
    }
  }
]
```

新增 prompt 约束：
- `For hold recommendations: use type="monitor" with 1-2 conditional watch steps (triggerType="price" or "event"). These steps represent conditions to watch, not immediate actions.`
- `Monitor steps should have negative or zero deltaPct (reduce or observe, never add).`

### 2.2 recommendation_prompt.go - fallbackRecommendation()

**hold/monitor 分支变更：**

```go
case recommendation.ActionHold:
	rec.Execution.Type = recommendation.ExecutionMonitor
	rec.Execution.Steps = []recommendation.Step{
		{
			Order:       1,
			TriggerType: recommendation.TriggerPrice,
			TriggerValue: fmt.Sprintf("%.4f below", input.CostPrice*0.95),
			DeltaPct:    -5,
			Rationale: recommendation.StructuredRationale{
				TriggerReason:  "Reduce if price breaks below stop-loss to limit downside.",
				PositionReason: "Moderate trim to observe before further action.",
				Precondition:   "Price closes below stop-loss level on consecutive days.",
				Fallback:       "If price recovers above cost, continue holding.",
				TimeWindow:     "Continuous monitoring.",
			},
		},
	}
	if input.CostPrice > 0 {
		stop := input.CostPrice * 0.95
		take := input.CostPrice * 1.10
		rec.Execution.StopLoss = &stop
		rec.Execution.TakeProfit = &take
	}
```

**非 hold 分支变更：**

将现有 `Rationale: "Aggressive add per matrix decision."` 等字符串改为 `StructuredRationale{TriggerReason: "..."}`，其余 4 字段留空。

### 2.3 synthesizer.go - ensureRecommendation()

新增检查：如果 `type == monitor && len(steps) == 0`，注入 fallback monitor 步骤（调用 `fallbackMonitorSteps(input)` 辅助函数）。

## 3 lotCount 计算（service/analysis/service.go）

计算位置：`AnalyzeHolding` 方法中，synthesis 完成后、fingerprint 计算前。

```go
// Post-synthesis: compute lotCount per step (privacy: totalCapitalCNY
// never flows into SynthesisInput or LLM context).
totalCap, capErr := s.userRepo.GetTotalCapitalCNY(ctx, userID)
if capErr == nil && totalCap != nil && len(data.Prices) > 0 {
	latestPrice := data.Prices[len(data.Prices)-1].Close
	if latestPrice > 0 {
		for i := range synthOutput.Recommendation.Execution.Steps {
			step := &synthOutput.Recommendation.Execution.Steps[i]
			raw := *totalCap * math.Abs(step.DeltaPct) / 100.0 / latestPrice
			if holding.AssetType == "a_share" {
				step.LotCount = math.Floor(raw/100) * 100
			} else {
				step.LotCount = math.Floor(raw)
			}
		}
	}
}
```

隐私约束遵守：`totalCapitalCNY` 仅在 service 层读取，不注入 `SynthesisInput`，不传入 LLM context。符合 `privacy_guard.go` 的 "Analysis pipeline input types must not embed total_capital" 约定。

计算顺序：lotCount 计算 -> fingerprint 计算（fingerprint 已排除 lotCount，顺序不影响结果，但逻辑上先算 lotCount 再算 fingerprint 更自然）。

## 4 前端类型变更

### 4.1 features/decision-card/types.ts

```typescript
export interface StructuredRationale {
	triggerReason: string;
	positionReason: string;
	precondition: string;
	fallback: string;
	timeWindow: string;
}

// Type guard for backward compat with pre-v2 string rationale
export function isStructuredRationale(
	r: unknown,
): r is StructuredRationale {
	return typeof r === "object" && r !== null && "triggerReason" in r;
}

export interface Step {
	order: number;
	triggerType: TriggerType;
	triggerValue: string;
	triggerPayload?: TriggerPayload;
	deltaPct: number;
	lotCount?: number;
	rationale: StructuredRationale | string;  // string for legacy cards
}
```

`Execution.steps` 保持 `steps?: Step[]`，前端逻辑用 `steps?.length` 判断是否有步骤。

### 4.2 向后兼容策略

渲染组件内部使用 `isStructuredRationale(step.rationale)` 分支：
- 是 StructuredRationale：逐字段渲染（跳过空字符串字段）
- 是 string：整体渲染为 triggerReason 位置（旧行为）

## 5 前端渲染变更

### 5.1 ExecutionPlanStrip（卡片墙缩略视图）

当前 monitor 分支只渲染 止损/止盈 文本。变更：

- 如果 `steps?.length > 0`：渲染首步的 triggerValue + deltaPct（与 non-monitor 一致），同时保留止损/止盈辅助行
- 如果 `steps` 为空/nil（旧卡）：保持原有止损/止盈文本渲染
- 显示步骤数量：`"+ 还有 N 步"` 提示

### 5.2 ExecutionPlanFull（详情页完整视图）

每个步骤卡片结构：

```
[步骤 N] [trigger icon] triggerValue | deltaPct%  [条件监控]  <- monitor 类型才有标签
参考手数: XX 手                                                <- lotCount > 0 时
--- rationale 区域（只渲染非空字段）---
触发原因: triggerReason
仓位依据: positionReason
前提条件: precondition
备选方案: fallback
时间窗口: timeWindow
```

monitor 类型的步骤在标题行右侧显示 Badge `(条件监控)`，与 action 步骤视觉区分。

rationale 区域：如果 5 个字段全部为空字符串，整个区域不渲染（template 降级场景）。如果部分字段有值，只渲染有值的字段。

## 6 i18n 新增 key

命名空间 `app`，路径 `decisionCard.executionPlan.*`：

| Key | zh | en |
|-----|----|----|
| rationale.triggerReason | 触发原因 | Trigger Reason |
| rationale.positionReason | 仓位依据 | Position Rationale |
| rationale.precondition | 前提条件 | Precondition |
| rationale.fallback | 备选方案 | Fallback |
| rationale.timeWindow | 时间窗口 | Time Window |
| lotCount | 参考手数 | Ref. Lot Count |
| lotUnit | 手 | lots |
| lotUnitShare | 股 | shares |
| monitorStepLabel | 条件监控 | Watch |

## 7 文件契约影响表

| 文件 | 现有契约 | 改动影响 |
|------|---------|---------|
| recommendation/types.go | Step.Rationale string | 替换为 StructuredRationale；新增 LotCount |
| recommendation/fingerprint.go | 排除 Rationale；哈希 order/triggerType/triggerValue/deltaPct | 保持不变，LotCount 自动排除（不在哈希行中）；更新注释 |
| synthesis/recommendation_prompt.go | monitor fallback Steps=nil；string rationale | monitor fallback 生成 1 步；rationale 改为 StructuredRationale |
| synthesis/synthesizer.go | ensureRecommendation 规范化 LLM 输出 | 新增 monitor 空步骤兜底逻辑 |
| service/analysis/service.go | AnalyzeHolding 调用 synthesizer 后直接算 fingerprint | 在 synthesizer 后插入 lotCount 计算，读 userRepo.GetTotalCapitalCNY |
| model/decision_card.go | Recommendation JSONB | 无变更，JSONB 自动兼容 |
| api/v1/decision_card.go | DTO 嵌入 recommendation.Recommendation | 无变更，类型变更自动传播 |
| features/decision-card/types.ts | Step.rationale: string | 改为 StructuredRationale 并 string 联合类型；新增 lotCount |
| ExecutionPlanStrip.tsx | monitor 只渲染止损/止盈 | monitor 有步骤时优先渲染步骤 |
| ExecutionPlanFull.tsx | rationale 作为 pre-wrap 文本渲染 | 逐字段渲染 StructuredRationale |
| i18n zh/en app.json | 无 rationale 字段 key | 新增 9 个 key |

## 8 替代路径验证

| 路径 | 设计处理 |
|------|---------|
| 旧卡查看（pre-migration） | 前端 type guard 分支，string rationale 渲染为单行文本 |
| LLM 不可用（template fallback） | fallbackRecommendation 生成固定步骤和空 StructuredRationale |
| totalCapitalCNY 未设置 | lotCount = 0，前端隐藏参考手数行 |
| LLM 返回 monitor + 空 steps | ensureRecommendation 兜底注入 fallbackMonitorSteps |
| 重新分析期间用户查看卡片 | 旧卡不受影响，新卡 CreateDecisionCard 新增行 |
| 旧卡 JSONB 反序列化新 Go struct | Go json.Unmarshal 对类型不匹配静默跳过，StructuredRationale 为零值 |
