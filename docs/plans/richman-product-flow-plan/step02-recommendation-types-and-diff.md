# Step 02 Recommendation 类型与徽章 Diff 算法

## 任务目标

新增两个独立 Go 包：

1. `backend/internal/analysis/recommendation` — 结构化建议数据类型（Action / Execution / Step / Recommendation 等），含 ActionLevel 计算、Execution 指纹算法
2. `backend/internal/analysis/diff` — 徽章 diff 算法 Compute 函数 + BadgeState 枚举

两个包均要有完整的单元测试覆盖核心路径。

## 涉及文件

创建：
- `backend/internal/analysis/recommendation/types.go`
- `backend/internal/analysis/recommendation/types_test.go`
- `backend/internal/analysis/recommendation/fingerprint.go`
- `backend/internal/analysis/recommendation/fingerprint_test.go`
- `backend/internal/analysis/diff/badge.go`
- `backend/internal/analysis/diff/badge_test.go`

## 设计依据

- TRD §2.2 Recommendation 数据模型
- TRD §3.1 - §3.4 徽章 diff 算法、执行计划指纹、阈值常量
- PRD §3.4 8 种徽章状态及优先级规则
- PRD §3.5 Recommendation 数据模型字段定义

## 实施要点

- recommendation 包内只放纯类型 + 纯函数，不引入 db、http、llm 依赖
- ActionLevel 映射函数实现 PRD §3.4"建议积极度等级"表
- Fingerprint 函数对 (Type, TargetPositionPct, StopLoss, TakeProfit, 每个 Step 的 TriggerType + TriggerValue + DeltaPct) 计算 SHA-1 hex
- diff.Compute 严格按 PRD §3.4 优先级 1-7 实现，第一个命中即返回
- ConfidenceDelta = current.Confidence - previous.Confidence；首次分析时 delta = 0
- 单元测试覆盖 8 种 BadgeState 各至少一组用例，加上"同时命中多状态时按优先级取最高"的用例至少 3 组
- 测试文件命名遵守 `docs/standards/testing.md`

## 验证标准

1. `cd backend && go test ./internal/analysis/recommendation/... ./internal/analysis/diff/... -v` 全部通过
2. 测试覆盖率 >= 85%（用 `go test -cover` 验证）
3. `make check` lint + vet 通过
4. 不引入新的第三方依赖（仅使用 stdlib + 项目现有 zap）

## 依赖说明

- 前置：step01（虽然类型不直接依赖 schema，但保持与 DB 字段同步的命名）

## 预估提交

- commit 1: `feat(analysis): add recommendation types and execution fingerprint`
- commit 2: `feat(analysis): add badge diff algorithm with state machine`
