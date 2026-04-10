# Step 5: 前端类型 + API 函数 + i18n 键

**设计依据：** TRD § 2.1（类型定义）、TRD § 2.2（API 函数）、TRD § 2.9（i18n 键）

**依赖：** Step 4（后端 API 结构已确定）

## 任务目标

在前端 `features/decision-card` 中新增 TypeScript 类型、getAnalysisTask API 函数，并在两个 locale 文件中同步添加 `analysisProgress` 节点。

## 涉及文件

- 修改：`frontend/src/features/decision-card/types.ts`
- 修改：`frontend/src/features/decision-card/api.ts`
- 修改：`frontend/src/i18n/locales/zh/app.json`
- 修改：`frontend/src/i18n/locales/en/app.json`

## 实施内容

**`types.ts`：** 参照 TRD § 2.1，新增以下类型（均为 export）：
- `TaskStepKey`（union literal）
- `TaskStepStatus`（union literal）
- `AnalysisTaskStep` 接口
- `HoldingAnalysisStatus`（union literal）
- `HoldingProgress` 接口
- `AnalysisTaskLog` 接口
- `AnalysisTaskStatus`（union literal）
- `AnalysisTask` 接口
- 同时修改现有 `RerunAnalysisResponse` 和 `ReanalyzeAllResponse` 确保含 `taskId: string`（若当前无该字段则添加）

**`api.ts`：** 参照 TRD § 2.2，新增 `getAnalysisTask(taskId: string)` 函数，请求 `GET /analysis/tasks/${taskId}`，返回 `ApiResponse<AnalysisTask>`

**i18n（两个 locale 同步）：** 参照 TRD § 2.9，在 `app.json` 中新增完整的 `analysisProgress` 节点（zh 为中文值，en 为英文值）

## 验证标准

- `cd frontend && pnpm lint:all` 无报错（重点：tsc 无类型错误）
- zh/en locale 文件 `analysisProgress` 节点的 key 集合完全一致

## 提交

```
feat(frontend): add AnalysisTask types, API function, and i18n keys
```
