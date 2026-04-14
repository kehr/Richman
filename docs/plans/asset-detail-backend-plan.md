# Asset Detail Backend 实施计划

## 目标

按 `docs/trds/asset-detail-backend-trd.md` 把 `GET /api/v2/market/:code` 的响应扩展为前端 `AssetDetailDto` 的 superset。

## 执行方式

- 调度：subagent-driven-development
- 隔离：单 worktree（所有 step 都改后端同一个 service 包，串行执行更安全）
- 验证：每 step 后跑 `cd backend && go vet ./... && go test ./internal/service/market/... && go build ./cmd/server/...`，全 step 完成后跑 `cd backend && make check`
- 合入：worktree → main，rebase → ff-merge → push

## Steps

### Step 1: Repo 层补 GetByID

设计依据：`docs/trds/asset-detail-backend-trd.md` 「Repo 层新增」

涉及文件：
- 修改 `backend/internal/repo/asset_analysis_read_repo.go`：新增 `GetByID(ctx, id) (*model.AssetAnalysis, error)`，复用 `assetAnalysisColumns` 与 `scanAssetAnalysisRow`

验证标准：
- `go build ./...` 通过
- 新方法 nil-row 走 `pgx.ErrNoRows` 返回 `(nil, nil)`，错误返回 `(nil, error)`

依赖：无

### Step 2: 新增 JSONB 内部解码类型

设计依据：TRD「JSONB 反序列化」

涉及文件：
- 创建 `backend/internal/service/market/jsonb.go`：定义 `rawDemoPlan` / `rawDemoPlanScenario` / `rawAnalysisMetadata` / `rawDrawdownReference` 与 `unmarshalDemoPlan` / `unmarshalAnalysisMetadata` 函数
- 解码失败统一返回 nil，service 层兜底

验证标准：
- 新文件不暴露 exported 类型（与其他 service 文件无符号冲突）
- `go vet ./...` 通过

依赖：Step 1 不阻塞，但同包文件以 PR 顺序为准

### Step 3: 扩展 AssetDetailDTO struct

设计依据：TRD「数据源映射」表 + 「DTO 类型新增」

涉及文件：
- 修改 `backend/internal/service/market/service.go`：
  - 新增 `ExecutionPlanDTO` / `ExecutionScenarioDTO` / `RiskFactorDTO` / `KeyPriceLevelDTO` / `DrawdownReferenceDTO` / `MajorChangeRecapDTO` / `DimensionDTO` / `DimensionSubIndicatorDTO`
  - 扩 `AssetDetailDTO` 字段（按 TRD 表）
  - 所有新字段用指针 + `omitempty`，json tag camelCase

验证标准：
- struct 字段顺序贴近前端 TS 接口（便于审查）
- `go build ./...` 通过

依赖：Step 1 / 2 完成（避免 import cycle）

### Step 4: 注入 richson client + 构造函数升级

设计依据：TRD「Service 层接口」

涉及文件：
- 修改 `backend/internal/service/market/service.go`：`Service` struct 加 `richsonClient *richson.Client`；`NewService` 参数追加 richsonClient；新增 `ohlcvCache` 字段（`map[string]ohlcvCacheEntry` + sync.Mutex）
- 修改 `backend/cmd/server/main.go`：调用 `NewService` 处补传 richsonClient
- 修改 `backend/internal/api/v2/router.go`（如调用方变了）：保持构造链路

验证标准：
- `go build ./...` 通过；DI 链路完整
- `go vet ./...` 通过
- 启动后 `/health` 不退化

依赖：Step 3

### Step 5: 实现 build* 辅助方法

设计依据：TRD「派生字段拼装」+ 「内部辅助方法」

涉及文件：
- 修改 `backend/internal/service/market/service.go`，新增私有方法：
  - `buildExecutionPlan(raw json.RawMessage) *ExecutionPlanDTO`
  - `buildDimensions(*model.AssetAnalysis, []model.AnalysisDimension) []DimensionDTO`
  - `buildKeyPriceLevels(supports, resistances []float64, currentPrice *float64, currency string) []KeyPriceLevelDTO`
  - `buildRiskFactors(raw json.RawMessage) []RiskFactorDTO`
  - `buildMajorChangeRecap(ctx, *model.AssetAnalysis) *MajorChangeRecapDTO`
  - `deriveDimensionSignal(*float64) string`

验证标准：
- 每个方法对 nil/空输入返回 nil 或空切片，不 panic
- `go build ./...` + `go vet ./...` 通过

依赖：Step 1 / 2 / 3 / 4

### Step 6: OHLCV 缓存层 + fetch helper

设计依据：TRD「缓存策略」+ 「内部辅助方法」

涉及文件：
- 修改 `backend/internal/service/market/service.go`：新增 `fetchOHLCVForDetail(ctx, code) *ohlcvSnapshot`，内部用 60s in-memory cache，richson 失败时 `s.logger.Warn(...)` 并返回 nil

验证标准：
- 缓存命中走内存；过期或缺失走 richson；失败响应不写缓存
- 所有日志带 `asset_code` 字段

依赖：Step 4

### Step 7: 重写 GetAssetDetail 主体

设计依据：TRD「Service 层接口」

涉及文件：
- 修改 `backend/internal/service/market/service.go`：`GetAssetDetail` 方法主干在原有 asset / analysis / dimensions / percentile 基础上：
  1. 调 `fetchOHLCVForDetail` 拿 currentPrice/sma200/supports/resistances/currency
  2. JSONB 解码 demo_plan / analysis_metadata / risk_factors
  3. 调用 buildDimensions / buildExecutionPlan / buildKeyPriceLevels / buildRiskFactors / buildMajorChangeRecap 填字段
  4. 设置 currency / usdExchangeRate / scoreBandLow/High / marketInterpretation / changeSummary / conflictType / conflictMessage / validDays
  5. drawdownReference 直接从 analysis_metadata 拼装

验证标准：
- 单元测试覆盖所有派生字段
- `make check` 通过

依赖：Step 1 / 2 / 3 / 4 / 5 / 6

### Step 8: 单元测试

设计依据：TRD「测试策略」

涉及文件：
- 创建 / 修改 `backend/internal/service/market/service_test.go`：表驱动单测覆盖 build* 方法 + deriveDimensionSignal 边界
- 创建 `backend/internal/repo/asset_analysis_read_repo_test.go`（如不存在）：GetByID 用 fakeRow / mock pgx 测试

验证标准：
- `go test ./internal/service/market/...` 通过
- 覆盖率不低于 build* 方法分支数

依赖：Step 7

### Step 9: 全栈构建验证 + 端到端 smoke

设计依据：项目验收闭环

涉及文件：无（验证步骤）

验证步骤：
- worktree 内 `cd backend && make check`（lint + test + build）
- 主仓库 `cd backend && go build ./...` 通过后启动 dev：`make dev`
- `curl http://localhost:8080/api/v2/market/SH600519` 返回完整 payload，包含 currentPrice / executionPlan / riskFactors / keyPriceLevels / drawdownReference 等新字段
- 前端 `pnpm dev` 打开 `/market/SH600519`，确认页面无空字段、无 Helmet 报错

依赖：Step 8

## 完成标准

- 9 个 step 全部 done
- backend `make check` 全绿
- 前端 asset-detail 页面所有 Tab 在生产数据上正常渲染
- 执行报告同步更新 Phase C 完成状态
