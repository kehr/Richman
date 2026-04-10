# Step 1: 扩展 TaskStatus Model 类型

**设计依据：** TRD § 1.1（新增 model 类型）

**依赖：** 无

## 任务目标

在 `backend/internal/model/task_status.go` 中新增分析任务所需的结构体和常量，为后续 TaskStore 扩展和 API 序列化提供类型基础。

## 涉及文件

- 修改：`backend/internal/model/task_status.go`

## 实施内容

参照 TRD § 1.1 的类型定义，在该文件中新增：

1. `TaskStepStatus` 类型及常量（`StepPending` / `StepRunning` / `StepDone` / `StepFailed`）
2. `TaskStep` 结构体（`Key string`、`Status TaskStepStatus`、`DurationMs *int64`、`startedAt time.Time` 非导出）
3. `TaskLog` 结构体（`Ts time.Time`、`Level string`、`Msg string`）
4. `HoldingProgress` 结构体（见 TRD § 1.1 字段列表，`startedAt time.Time` 非导出）
5. 步骤 key 常量（`StepKeyFetchData` 等 5 个）
6. `DefaultSteps() []TaskStep` 辅助函数，按顺序返回 5 个 pending 步骤
7. 在现有 `TaskStatus` 结构体中新增字段：`CurrentHolding string`、`Holdings []HoldingProgress`、`Steps []TaskStep`、`Logs []TaskLog`

## 验证标准

- `cd backend && go build ./internal/model/...` 无报错
- `go vet ./internal/model/...` 无警告

## 提交

```
feat(backend): extend TaskStatus model with step/log/holding types
```
