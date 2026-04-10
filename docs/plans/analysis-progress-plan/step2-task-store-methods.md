# Step 2: TaskStore 新增方法

**设计依据：** TRD § 1.1（TaskStore 方法签名）

**依赖：** Step 1（model 类型已定义）

## 任务目标

在 `backend/internal/service/analysis/task_store.go` 中新增步骤推进、日志追加、持仓状态更新方法，并确保所有新方法在读写 `TaskStatus` 的新字段时正确持有锁。

## 涉及文件

- 修改：`backend/internal/service/analysis/task_store.go`

## 实施内容

参照 TRD § 1.1，新增以下方法（签名见 TRD）：

1. **`InitHoldings(taskID string, holdings []model.HoldingProgress)`**
   - 写入 `task.Holdings`，同时将 `task.Steps` 初始化为 `model.DefaultSteps()`

2. **`SetCurrentHolding(taskID string, symbol string)`**
   - 更新 `task.CurrentHolding`
   - 将 `task.Steps` 重置为 `model.DefaultSteps()`（每张卡片重置步骤）

3. **`UpdateHoldingStatus(taskID, symbol, status string, source, provider *string, durationMs *int64)`**
   - 在 `task.Holdings` 中找到对应 symbol，更新其字段

4. **`StartStep(taskID string, key string)`**
   - 在 `task.Steps` 中找到对应 key，将 `Status` 改为 `StepRunning`，记录 `startedAt = time.Now()`

5. **`CompleteStep(taskID string, key string)`**
   - 找到对应 step，`Status` 改为 `StepDone`，计算 `DurationMs = time.Since(startedAt).Milliseconds()`

6. **`FailStep(taskID string, key string)`**
   - 找到对应 step，`Status` 改为 `StepFailed`，计算 `DurationMs`

7. **`AppendLog(taskID string, level string, msg string)`**
   - append `model.TaskLog{Ts: time.Now(), Level: level, Msg: msg}` 到 `task.Logs`

8. **`Get` 方法**（若现有实现返回指针）：改为返回值拷贝或对 slice 字段做 `append(nil, ...)` 防御性拷贝，避免调用方持有内部指针发生 race

## 验证标准

- `cd backend && go build ./internal/service/analysis/...` 无报错
- `go vet ./internal/service/analysis/...` 无警告
- 现有 `service_test.go` 全部通过：`go test ./internal/service/analysis/...`

## 提交

```
feat(backend): add step/log/holding methods to TaskStore
```
