# Step 6: LLM Integration + Decision Card Generation

## 任务目标

实现多模型 LLM 抽象层、催化剂维度的 LLM 增强（联网搜索 + 语义解读）、LLM 综合层（三维结果 -> 结构化决策卡）。实现分析编排 service 和决策卡存储/查询 API。

## 涉及文件路径

### 创建

- `backend/internal/llm/provider.go` -- LLMProvider 接口定义
- `backend/internal/llm/claude/client.go` -- Claude API 实现
- `backend/internal/llm/openai/client.go` -- OpenAI API 实现
- `backend/internal/llm/factory.go` -- Provider 工厂（按配置选择）
- `backend/internal/analysis/catalyst/llm_enhancer.go` -- 催化剂 LLM 增强（搜索 + 解读）
- `backend/internal/analysis/catalyst/llm_enhancer_test.go`
- `backend/internal/analysis/synthesis/synthesizer.go` -- LLM 综合层（三维 -> 决策卡）
- `backend/internal/analysis/synthesis/synthesizer_test.go`
- `backend/internal/model/analysis_result.go` -- 分析结果模型
- `backend/internal/model/decision_card.go` -- 决策卡模型
- `backend/internal/service/analysis/service.go` -- 分析编排 service
- `backend/internal/service/analysis/service_test.go`
- `backend/internal/service/decision_card/service.go` -- 决策卡查询 service
- `backend/internal/api/v1/analysis.go` -- 分析触发路由
- `backend/internal/api/v1/decision_card.go` -- 决策卡查询路由
- `backend/db/migration/003_analysis.up.sql` -- analysis_results + decision_cards 表
- `backend/db/migration/003_analysis.down.sql`
- `backend/db/query/analysis_result.sql`
- `backend/db/query/decision_card.sql`

## PRD/TRD 章节引用

- PRD 3.2.1 双层架构（量化底座 + LLM 增强）
- PRD 3.2.2 催化剂 LLM 增强描述
- PRD 3.3 决策卡内容和格式
- PRD 3.2.7 降级策略（LLM 不可用时）
- `docs/standards/backend.md` service 层模式
- `docs/standards/api.md` 异步任务模式、决策卡端点

## 验证标准

- [ ] LLM Provider 接口可切换 Claude / OpenAI
- [ ] 催化剂 LLM 增强：调用 LLM 搜索后输出事件摘要
- [ ] 催化剂降级：LLM 不可用时仅使用 Polymarket 量化底座
- [ ] LLM 综合层：接收三维量化结果，输出结构化决策卡 JSON
- [ ] 决策卡包含全部必要字段（三维摘要、信心度、操作建议、两层操作点、风险提示、今日要点）
- [ ] `POST /api/v1/analysis/trigger` 返回 202 + taskId
- [ ] `GET /api/v1/tasks/:taskId` 返回任务进度
- [ ] `GET /api/v1/decision-cards` 返回最新一轮决策卡
- [ ] `GET /api/v1/decision-cards/:id` 返回单张决策卡详情
- [ ] `GET /api/v1/decision-cards/history` 返回历史决策卡
- [ ] 分析结果和决策卡正确持久化到数据库
- [ ] `go test ./internal/analysis/synthesis/...` 通过（使用 mock LLM）
- [ ] `go test ./internal/service/analysis/...` 通过
- [ ] `golangci-lint run ./...` 零错误
- [ ] `go vet ./...` 零警告

## 依赖说明

- Step 5 完成（三维量化计算引擎就绪）
- Step 3 完成（持仓数据可用）
