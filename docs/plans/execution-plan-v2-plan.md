# 执行计划 V2 实施 Plan

PRD: `docs/prds/execution-plan-v2-prd.md`
TRD: `docs/trds/execution-plan-v2-trd.md`

## Step 1: 后端类型和 fingerprint

目标: 修改 recommendation 包的类型定义，新增 StructuredRationale，更新 Step 结构体。

涉及文件:
- 修改 `backend/internal/analysis/recommendation/types.go`
- 修改 `backend/internal/analysis/recommendation/fingerprint.go`（更新注释）
- 修改 `backend/internal/analysis/recommendation/types_test.go`（如有 Step 相关断言）
- 修改 `backend/internal/analysis/recommendation/fingerprint_test.go`（如有 Step 相关断言）

设计依据:
- TRD 1.1 StructuredRationale 类型定义
- TRD 1.2 fingerprint 排除 LotCount

验证标准:
- `go build ./internal/analysis/recommendation/...` 通过
- `go test ./internal/analysis/recommendation/...` 通过
- Step 结构体包含 LotCount float64 和 Rationale StructuredRationale

依赖: 无

## Step 2: 后端 LLM prompt、fallback 和 synthesizer

目标: 更新 LLM prompt 输出 StructuredRationale；更新 fallback 为 monitor 生成步骤；更新 ensureRecommendation 兜底。

涉及文件:
- 修改 `backend/internal/analysis/synthesis/recommendation_prompt.go`
- 修改 `backend/internal/analysis/synthesis/synthesizer.go`
- 修改 `backend/internal/analysis/synthesis/synthesizer_test.go`（如有断言）

设计依据:
- TRD 2.1 prompt JSON schema
- TRD 2.2 fallbackRecommendation hold 分支
- TRD 2.3 ensureRecommendation monitor 空步骤兜底

验证标准:
- `go build ./internal/analysis/synthesis/...` 通过
- `go test ./internal/analysis/synthesis/...` 通过
- fallbackRecommendation(hold) 返回包含 1 个步骤的 monitor plan
- ensureRecommendation 对 monitor+空步骤注入 fallback

依赖: Step 1

## Step 3: 后端 service 层 lotCount 计算

目标: 在 AnalyzeHolding 的 synthesis 后、fingerprint 前插入 lotCount 计算逻辑。

涉及文件:
- 修改 `backend/internal/service/analysis/service.go`

设计依据:
- TRD 3 lotCount 计算位置和公式
- PRD 5 lotCount 计算规则

验证标准:
- `go build ./internal/service/analysis/...` 通过
- `go test ./internal/service/analysis/...` 通过
- service.go 在 synthesis 后调用 userRepo.GetTotalCapitalCNY 计算 lotCount

依赖: Step 1, Step 2

## Step 4: 前端类型和 i18n

目标: 更新 TypeScript 类型定义，新增 i18n key。

涉及文件:
- 修改 `frontend/src/features/decision-card/types.ts`
- 修改 `frontend/src/i18n/locales/zh/app.json`
- 修改 `frontend/src/i18n/locales/en/app.json`

设计依据:
- TRD 4.1 StructuredRationale interface
- TRD 4.2 向后兼容策略
- TRD 6 i18n key 列表

验证标准:
- `pnpm tsc --noEmit` 通过
- zh 和 en 同步包含 9 个新 key
- Step.rationale 类型为 `StructuredRationale | string`

依赖: 无（可与 Step 1-3 并行）

## Step 5: 前端渲染组件更新

目标: 更新 ExecutionPlanStrip 和 ExecutionPlanFull 渲染逻辑。

涉及文件:
- 修改 `frontend/src/features/decision-card/components/ExecutionPlanStrip.tsx`
- 修改 `frontend/src/pages/decision-cards/components/ExecutionPlanFull.tsx`

设计依据:
- TRD 5.1 ExecutionPlanStrip 变更
- TRD 5.2 ExecutionPlanFull 变更
- PRD 6 UI 渲染规格

验证标准:
- `pnpm lint:all` 通过
- monitor 类型有步骤时渲染步骤而非仅止损/止盈
- StructuredRationale 逐字段渲染（空字段隐藏）
- lotCount > 0 时显示参考手数
- 旧卡 string rationale 不 crash

依赖: Step 4

## Step 6: 全链路验证

目标: 前后端 lint + build 全链路通过。

涉及文件:
- 无新文件

验证标准:
- `cd frontend && pnpm lint:all` 通过
- `cd backend && make check` 通过
- 无 type error
- 无 lint error

依赖: Step 1-5
