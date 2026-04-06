# Step 04 权重管理器接受 risk_preference bias

## 任务目标

让 weight manager 在 LLM 微调权重的基础上，根据用户的 risk_preference 字段额外加 bias，实现 PRD §6 风险偏好对分析结果的影响。

## 涉及文件

修改：
- `backend/internal/analysis/weight/manager.go`
- `backend/internal/analysis/weight/manager_test.go`
- `backend/internal/service/analysis/service.go`（在调用 weight manager 时传入用户的 risk_preference）

可能涉及：
- `backend/internal/repo/user_repo.go`（增加 GetRiskPreference 方法或包含在 GetByID 返回值里）

## 设计依据

- TRD §5.4 risk_preference 影响权重
- PRD §6.2 Tab 1 账户的风险偏好字段说明
- PRD §3.4 现有权重微调机制（保留 LLM 在 ±10% 范围内调整，bias 是叠加而非替代）

## 实施要点

- weight.Manager 现有 Adjust 函数签名增加 RiskPreference 参数（或新增 AdjustWithBias 函数避免破坏现有调用）
- bias 规则按 TRD §5.4：
  - conservative：position +5%、catalyst -5%
  - neutral：不调整
  - aggressive：catalyst +5%、position -5%
- 应用 bias 后必须**重新归一化**保证三维之和等于 100%，且每个维度仍在 PRD §3.2.3 表格的允许范围内（不能超出 ±10% 上限）
- 如果叠加 bias 后超出范围，截断到最大允许值（不报错）
- 单元测试覆盖三种 risk_preference × 至少两种 LLM 微调结果，验证最终权重正确
- analysis service 在调 weight manager 前从 user repo 读 risk_preference，没读到默认 neutral

## 验证标准

1. `go test ./internal/analysis/weight/...` 通过
2. 测试覆盖三种 risk_preference 下的归一化逻辑
3. neutral 用户分析结果与改造前完全一致（回归保护）
4. `make check` 通过

## 依赖说明

- 前置：step01（users 表 risk_preference 字段必须存在）

## 预估提交

- 1 次 commit：`feat(weight): apply risk_preference bias on top of llm adjustment`
