# Step 03 Synthesis 扩展与分析管线集成

## 任务目标

把 step02 的 recommendation 类型接入现有分析管线：

1. 扩展 `synthesis.Synthesizer` 让它在生成卡片文本的同时输出结构化 Recommendation
2. 在 LLM 失败时通过模板降级生成默认的执行计划，保证管线总能产出非空结构
3. 在 `service/analysis/Service` 写卡前调用 diff.Compute 计算 badge_state、confidence_delta、execution_fingerprint，并写入 prev_card_id

## 涉及文件

修改：
- `backend/internal/analysis/synthesis/synthesizer.go`
- `backend/internal/analysis/synthesis/synthesizer_test.go`（如不存在则创建）
- `backend/internal/analysis/synthesis/template_fallback.go`（如已有则修改）
- `backend/internal/service/analysis/service.go`
- `backend/internal/service/analysis/service_test.go`（如不存在则创建）
- `backend/internal/repo/decision_card_repo.go`（增加新字段读写）
- `backend/db/query/decision_card.sql`（如使用 sqlc）
- `backend/internal/model/decision_card.go`（如有 model 层）

创建：
- `backend/internal/analysis/synthesis/recommendation_prompt.go`（拆分 prompt 构造逻辑，避免 synthesizer.go 过大）

## 设计依据

- TRD §2.4 Synthesizer 变更要点
- TRD §2.5 API 响应字段（决定 repo / model 需要暴露什么）
- TRD §3.5 调用时机（写卡前事务性调 diff）
- PRD §3.5 Recommendation 数据模型

## 实施要点

- SynthesisOutput 增加结构化 recommendation 字段
- LLM Prompt 在原有"生成 trend/position/catalyst summary"基础上追加输出 recommendation 子对象的指令
- 解析 LLM 响应失败时，根据 matrix 推导出的 base recommendation + ActionLevel 构造一个 1 步 one-shot 默认计划作为降级
- 降级路径必须能通过单元测试断言（mock LLM 报错 → 输出非空 recommendation）
- analysis service 写卡流程改为：
  1. 查同 holding 最近一张卡
  2. 构造 diff.Input
  3. 计算 badge_state / confidence_delta / fingerprint
  4. 写新卡（含上一卡 ID）
- 整个写卡过程必须在同一个事务中完成，避免并发分析造成 prev 错位
- decision_card_repo 增加 GetLatestByHolding 方法（如不存在）
- prev_card_id 允许 NULL（首次分析）

## 验证标准

1. `go test ./internal/analysis/synthesis/... ./internal/service/analysis/...` 通过
2. 单测覆盖：LLM 成功 / LLM 失败降级 / 首次分析（无 prev）/ 有 prev 的几种 badge 状态
3. 集成跑一次实际分析（mock 数据源）能落卡，badge_state 字段非默认值
4. `make check` 通过

## 依赖说明

- 前置：step01 数据库迁移、step02 类型与算法

## 预估提交

- commit 1: `feat(synthesis): generate structured recommendation with template fallback`
- commit 2: `feat(analysis): integrate badge diff into card persistence pipeline`
- commit 3: `feat(repo): expose new decision_card structured fields`
