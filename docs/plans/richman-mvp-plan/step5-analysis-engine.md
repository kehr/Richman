# Step 5: Three-Dimension Analysis Engine

## 任务目标

实现三维量化分析引擎的核心计算逻辑：趋势维度（MA/RSI/MACD）、位置维度（PE/PB 分位/CAPE）、催化剂维度（Polymarket 概率量化底座）。实现权重管理和信心度计算。

## 涉及文件路径

### 创建

- `backend/internal/analysis/types.go` -- 分析结果类型（维度结果、决策矩阵、信心度）
- `backend/internal/analysis/trend/calculator.go` -- 趋势维度量化计算
- `backend/internal/analysis/trend/calculator_test.go`
- `backend/internal/analysis/position/calculator.go` -- 位置维度量化计算
- `backend/internal/analysis/position/calculator_test.go`
- `backend/internal/analysis/catalyst/calculator.go` -- 催化剂维度量化底座（Polymarket）
- `backend/internal/analysis/catalyst/calculator_test.go`
- `backend/internal/analysis/weight/manager.go` -- 权重管理（预设 + 微调范围）
- `backend/internal/analysis/weight/manager_test.go`
- `backend/internal/analysis/confidence/calculator.go` -- 信心度计算（0-100%）
- `backend/internal/analysis/confidence/calculator_test.go`
- `backend/internal/analysis/matrix.go` -- 决策矩阵（三维 -> 五级建议）
- `backend/internal/analysis/matrix_test.go`

## PRD/TRD 章节引用

- PRD 3.2.2 三个维度（量化信号、输出格式）
- PRD 3.2.3 权重机制（四类标的权重表、+/-10% 微调）
- PRD 3.2.4 决策矩阵（五级建议）
- PRD 3.2.5 信心度（0-100%，计算逻辑）
- PRD 3.2.7 降级策略

## 验证标准

- [ ] 趋势计算：给定价格序列，输出正确的方向和强度评分
- [ ] 趋势计算：MA 交叉信号、RSI、MACD 计算结果与预期一致
- [ ] 位置计算：给定 PE 历史序列和当前值，输出正确的分位数
- [ ] 位置计算：黄金用实际利率/美元指数，美股用 CAPE，逻辑分支正确
- [ ] 催化剂量化底座：给定 Polymarket 概率，输出正确的催化剂方向
- [ ] 权重管理：四类标的的预设权重正确
- [ ] 权重管理：微调范围约束生效（超出 +/-10% 被裁剪）
- [ ] 权重管理：三维权重之和始终为 100%
- [ ] 信心度：三维一致时 80-100%，两维一致 50-70%，各异 20-40%
- [ ] 信心度：数据缺失时扣减正确
- [ ] 决策矩阵：三维输入 -> 五级建议输出正确
- [ ] `go test ./internal/analysis/...` 全部通过
- [ ] `golangci-lint run ./...` 零错误
- [ ] `go vet ./...` 零警告

## 依赖说明

- Step 4 完成（数据源客户端可提供行情和估值数据）
