# Step 3: Backend Integration

## 任务目标

把 Step 2 的 Resolver 集成到分析流水线和 API 层，包含 Synthesizer 接口升级、Analysis Service 改造、新的 REST handlers、main.go wire-up。完成后 backend 端到端工作，前端可以开始消费 API。

## 涉及文件

### 修改

- `backend/internal/analysis/synthesis/synthesizer.go`：接口改为 `Synthesize(ctx, input, userID) (*Output, *Meta, error)`
- `backend/internal/analysis/synthesis/synthesizer_test.go`：stubProvider 改为 stubResolver，全部现有测试迁移到新签名
- `backend/internal/service/analysis/service.go`：AnalyzeHolding 里调用新签名，写入 card 的新字段；新增 `TriggerReanalyzeAll`
- `backend/internal/repo/decision_card_repo.go`：CreateDecisionCard/Tx 的 INSERT 包含新列；Get/List 的 SELECT 包含新列
- `backend/internal/api/v1/decision_cards.go`：DTO 增加 `synthesisSource` + `providerUsed`，nullable 映射为 "unknown"
- `backend/internal/api/v1/dashboard.go`：summary 响应新增 `llmStatus` 子对象
- `backend/internal/api/v1/analysis.go`：新增 `POST /reanalyze-all` handler + rate limiting
- `backend/cmd/server/main.go`：初始化 Crypto、Resolver、新 repos，把 Resolver 注入 Synthesizer

### 新增

- `backend/internal/api/v1/settings_llm.go`：5 个 handler（GET/PUT/DELETE/probe/onboarding-consent）
- `backend/internal/api/v1/settings_llm_test.go`：handler 级别测试
- `backend/internal/api/v1/reanalyze_test.go`：reanalyze-all 测试

## 设计依据

- PRD "Fallback 链" + "数据模型/三态触发规则"：Synthesizer 的输出 meta 如何确定
- PRD "可观测性"：每次 resolver 调用输出一条结构化 log
- TRD "Synthesizer 接口演进"：新签名、SynthesisMeta 结构
- TRD "Service 层改造"：AnalyzeHolding 里的 meta 传递 + 写库
- TRD "API 层/settings_llm.go"：5 个 endpoint 的请求/响应 schema
- TRD "API 层/analysis.go 扩展"：reanalyze-all 的 rate limit
- TRD "API 层/decision_cards.go 扩展"：DTO 字段映射
- TRD "API 层/dashboard.go 扩展"：needsReanalysis SQL 逻辑
- TRD "Wire-up (main.go)"：初始化顺序

## 验证标准

### Synthesizer 测试全部通过

原有 5 个测试用例重写为新签名：
- LLMSuccess：stubResolver 返回 `Layer=user` + 完整响应，断言 `meta.Source=llm`, `meta.ProviderUsed=user`
- LLMFailure：stubResolver 返回 err，断言 `meta.Source=template`, `meta.ProviderUsed=none`
- MalformedJSON：stubResolver 返回无效 JSON，断言 `meta.Source=template`, `meta.ProviderUsed=user`（层级保留）
- MissingRecommendation：断言 `meta.Source=mixed`, `meta.ProviderUsed=user`
- NilResolver：Synthesizer 构造时传 nil，断言不 panic，返回 `meta.Source=template, meta.ProviderUsed=none`

新增 2 个用例：
- SystemDefaultFallback：stubResolver 返回 `Layer=system_default`，断言 `meta.ProviderUsed=system_default`
- AllLayersFailed：stubResolver 返回 `ErrAllLayersFailed`，断言走 template

### Service 测试

- AnalyzeHolding 调用后，持久化的 DecisionCard 包含非空的 `SynthesisSource` 和 `ProviderUsed`
- `TriggerReanalyzeAll` 行为与 `TriggerAnalysis` 等价或显式调用它

### Handler 测试

- `GET /api/v1/settings/llm`：未配置返回 `configured=false`，已配置返回 masked 响应
- `PUT /api/v1/settings/llm`：
  - 合法 claude 配置 + probe=true + stub probe 成功 → 200
  - openai_compatible + 合法 https base_url → 200
  - openai_compatible + http base_url → 400 (SSRF)
  - openai_compatible + private IP base_url → 400 (SSRF)
  - probe 失败 → 400 带详细 error
  - 未登录 → 401
- `DELETE /api/v1/settings/llm` → 204，再次 GET 返回 `configured=false`
- `POST /settings/llm/probe` → 返回健康状态 + latency
- `POST /analysis/reanalyze-all`：
  - 首次调用 → 200 带 taskId
  - 10 分钟内第二次 → 429
- `GET /decision-cards/:id`：响应包含 `synthesisSource` 和 `providerUsed` 字段
- `GET /dashboard/summary`：响应包含 `llmStatus` 子对象，`needsReanalysis` 按 SQL 计算

### 集成验证

- `make check`（backend）全绿
- `golangci-lint run ./...` 0 issues
- `go test ./...` 所有包 pass

## 依赖

- Step 1 + Step 2 已完成

## 偏差处理

- 如果 Synthesizer 接口变更影响了不在清单里的调用点（比如某个其它模块的直接调用），同步更新并在执行报告里说明
- Handler 的 rate limit 优先用现有中间件，如果现有中间件只支持全局限流，需要扩展成 per-user；该扩展也记入执行报告
- `DecisionCard` DTO 映射 null → "unknown" 可以在 model → DTO 的 mapper 函数里集中处理
- settings_llm.go 的 handler 数量较多，允许拆分成 settings_llm_get.go / settings_llm_put.go 等，按现有代码风格决定

## 预期产出

- Synthesizer 接口升级 + 测试迁移
- Service 层写入新字段
- 5 个 settings/llm handler + 1 个 reanalyze-all handler
- DecisionCard DTO 扩展
- Dashboard summary 扩展
- main.go wire-up
- 多个分主题 commit：
  - `refactor(synthesis): thread SynthesisMeta through Synthesize signature`
  - `feat(service): persist synthesis_source and provider_used on decision cards`
  - `feat(api): add llm settings endpoints and onboarding consent`
  - `feat(api): add reanalyze-all endpoint with per-user rate limit`
  - `feat(api): extend decision_cards and dashboard summary responses`
  - `feat(server): wire up llm crypto, resolver, and settings handlers`
