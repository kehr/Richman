# Step 3: 分析服务插桩

**设计依据：** TRD § 1.3（Service 层插桩调用序列）

**依赖：** Step 2（TaskStore 新方法已就绪）

## 任务目标

在 `backend/internal/service/analysis/service.go` 的 goroutine 中，按照 TRD § 1.3 的插桩序列，在各分析阶段前后调用 TaskStore 方法，向 TaskStatus 写入步骤状态、日志和持仓进度。

## 涉及文件

- 修改：`backend/internal/service/analysis/service.go`

## 实施内容

在 `TriggerAnalysis` goroutine 中，按 TRD § 1.3 的顺序插入调用：

1. **分析前**：`InitHoldings`，传入所有待分析持仓（symbol + name，status=pending）

2. **每张持仓循环开始**：
   - `SetCurrentHolding(taskID, symbol)`（同时重置 Steps）
   - `UpdateHoldingStatus(symbol, "running", ...)`
   - `AppendLog("info", "[symbol] start")`

3. **获取数据阶段**：
   - `StartStep(StepKeyFetchData)` → fetch → `CompleteStep` / `FailStep`
   - `AppendLog("info", "[symbol] fetch ok · trend=X")`

4. **计算指标阶段**（trend+position+catalyst+weights 合并为一步）：
   - `StartStep(StepKeyCalcIndicators)` → 计算 → `CompleteStep` / `FailStep`

5. **推荐决策阶段**：
   - `StartStep(StepKeyRecommendation)` → matrix → `CompleteStep` / `FailStep`

6. **LLM 合成阶段**：
   - `StartStep(StepKeyLLMSynthesis)`
   - `AppendLog("info", "[symbol] LLM call [provider]")`
   - → synthesize → 若降级：`AppendLog("warn", "[symbol] LLM timeout, fallback → template")`
   - `CompleteStep` / `FailStep`

7. **持久化阶段**：
   - `StartStep(StepKeyPersist)` → persist → `CompleteStep` / `FailStep`

8. **每张持仓完成**：
   - `UpdateHoldingStatus(symbol, "done", source, provider, durationMs)`
   - `AppendLog("info", "[symbol] done · source=X provider=Y")`

注意：若某阶段 error 不终止整体流程（当前代码有 continue 逻辑），FailStep 后继续下一张持仓；若整体失败，`taskStore.Fail(taskID, err)` 保持现有逻辑不变。

## 验证标准

- `go build ./internal/service/analysis/...` 无报错
- `go test ./internal/service/analysis/...` 现有测试全部通过（插桩调用不影响现有测试逻辑）

## 提交

```
feat(backend): instrument analysis service with task step/log tracking
```
