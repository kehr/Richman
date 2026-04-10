# 分析进度与执行日志 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为触发分析操作添加右侧进度 Drawer，实时展示步骤时间轴和执行日志，LLM 降级时显示橙色警告，完成后卡片自动刷新。

**Architecture:** 后端扩展内存 TaskStore（新增步骤/日志/持仓进度字段）并新增 GET 查询端点；前端以 1.5s 轮询驱动 Drawer，useRerunAnalysis/useReanalyzeAll 改为返回 taskId，缓存失效推迟到轮询检测到 done 状态时触发。

**Tech Stack:** Go (Gin + in-memory TaskStore)、React 19、Ant Design 6 Drawer、TanStack Query v5 refetchInterval、react-i18next

**设计文档：**
- PRD: `docs/prds/analysis-progress-prd.md`
- TRD: `docs/trds/analysis-progress-trd.md`

## 文件地图

### 后端（修改）
| 文件 | 变更类型 | 职责 |
|------|---------|------|
| `backend/internal/model/task_status.go` | 修改 | 新增 TaskStep、TaskLog、HoldingProgress 类型及常量 |
| `backend/internal/service/analysis/task_store.go` | 修改 | 新增步骤/日志/持仓方法 |
| `backend/internal/service/analysis/service.go` | 修改 | 在各分析阶段插入 TaskStore 调用 |
| `backend/internal/api/v1/analysis.go` | 修改 | 新增 GetTask handler |
| `backend/internal/api/router.go` | 修改 | 注册 GET /analysis/tasks/:taskId 路由 |

### 前端（新建）
| 文件 | 变更类型 | 职责 |
|------|---------|------|
| `frontend/src/features/decision-card/use-analysis-task.ts` | 新建 | 轮询 task 状态的 TanStack Query hook |
| `frontend/src/features/decision-card/components/AnalysisStepTimeline.tsx` | 新建 | 步骤时间轴子组件 |
| `frontend/src/features/decision-card/components/AnalysisLogPanel.tsx` | 新建 | 可滚动日志面板子组件 |
| `frontend/src/features/decision-card/components/AnalysisProgressDrawer.tsx` | 新建 | 进度 Drawer 主组件 |

### 前端（修改）
| 文件 | 变更类型 | 职责 |
|------|---------|------|
| `frontend/src/features/decision-card/types.ts` | 修改 | 新增 AnalysisTask、HoldingProgress 等类型 |
| `frontend/src/features/decision-card/api.ts` | 修改 | 新增 getAnalysisTask API 函数 |
| `frontend/src/features/decision-card/use-rerun-analysis.ts` | 修改 | onSuccess 改为回调返回 taskId |
| `frontend/src/features/decision-card/use-reanalyze-all.ts` | 修改 | 同上 |
| `frontend/src/features/decision-card/index.ts` | 修改 | 导出新组件和 hook |
| `frontend/src/features/decision-card/components/DecisionCardSummary.tsx` | 修改 | 接受 analysisStatus prop，渲染更新中状态 |
| `frontend/src/pages/dashboard/DashboardPage.tsx` | 修改 | 接入 taskId state 和 AnalysisProgressDrawer |
| `frontend/src/pages/dashboard/components/DashboardTopStrip.tsx` | 修改 | 按钮状态改为分析中/完成态 |
| `frontend/src/i18n/locales/zh/app.json` | 修改 | 新增 analysisProgress 节点 |
| `frontend/src/i18n/locales/en/app.json` | 修改 | 新增 analysisProgress 节点 |

## Step 索引

| # | 文件 | 内容 |
|---|------|------|
| 1 | [step1-backend-model.md](analysis-progress-plan/step1-backend-model.md) | 扩展 TaskStatus model 类型 |
| 2 | [step2-task-store-methods.md](analysis-progress-plan/step2-task-store-methods.md) | TaskStore 新增方法 |
| 3 | [step3-service-instrumentation.md](analysis-progress-plan/step3-service-instrumentation.md) | 分析服务插桩 |
| 4 | [step4-backend-endpoint.md](analysis-progress-plan/step4-backend-endpoint.md) | GET /tasks/:taskId 端点 + 后端验收 |
| 5 | [step5-frontend-types-api.md](analysis-progress-plan/step5-frontend-types-api.md) | 前端类型 + API 函数 + i18n 键 |
| 6 | [step6-polling-hook.md](analysis-progress-plan/step6-polling-hook.md) | useAnalysisTask + 修改 rerun/reanalyze mutations |
| 7 | [step7-step-timeline.md](analysis-progress-plan/step7-step-timeline.md) | AnalysisStepTimeline 组件 |
| 8 | [step8-log-panel.md](analysis-progress-plan/step8-log-panel.md) | AnalysisLogPanel 组件 |
| 9 | [step9-progress-drawer.md](analysis-progress-plan/step9-progress-drawer.md) | AnalysisProgressDrawer 主组件 |
| 10 | [step10-card-status.md](analysis-progress-plan/step10-card-status.md) | DecisionCardSummary 更新中状态 |
| 11 | [step11-page-wiring.md](analysis-progress-plan/step11-page-wiring.md) | DashboardPage + TopStrip 接入 + barrel + 最终 lint |
