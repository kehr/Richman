# Step 6: useAnalysisTask hook + 修改 rerun/reanalyze mutations

**设计依据：** TRD § 2.3（useAnalysisTask 契约）、TRD § 2.4（修改 mutations）

**依赖：** Step 5（AnalysisTask 类型和 getAnalysisTask 已就绪）

## 任务目标

新建 `useAnalysisTask` hook 封装轮询逻辑；修改 `useRerunAnalysis` 和 `useReanalyzeAll`，使其在成功后通过回调将 `taskId` 传出，不再直接失效缓存。

## 涉及文件

- 新建：`frontend/src/features/decision-card/use-analysis-task.ts`
- 修改：`frontend/src/features/decision-card/use-rerun-analysis.ts`
- 修改：`frontend/src/features/decision-card/use-reanalyze-all.ts`

## 实施内容

**`use-analysis-task.ts`：**

参照 TRD § 2.3 契约，使用 TanStack Query `useQuery`：
- `queryKey: ["analysis-task", taskId]`
- `enabled: taskId !== null`
- `staleTime: 0`
- `refetchInterval`: 当 `query.state.data?.status === "running"` 时返回 `1500`，否则 `false`
- 通过 `useEffect` 监听 `task?.status`，当值变为 `"done"` 时调用 `queryClient.invalidateQueries({ queryKey: DECISION_CARDS_QUERY_KEY })`（触发卡片数据刷新）
- 返回 `{ task, isPolling }` 两个值（见 TRD § 2.3 定义）

**`use-rerun-analysis.ts`：**

参照 TRD § 2.4，接受可选参数 `onTaskStarted?: (taskId: string) => void`：
- `onSuccess` 改为从 `data.data.taskId` 取出 taskId 后调用 `onTaskStarted?.(taskId)`
- 移除 `queryClient.invalidateQueries` 调用（改由 useAnalysisTask 驱动）

**`use-reanalyze-all.ts`：**

同 useRerunAnalysis 的改动逻辑，接受相同可选参数。

## 验证标准

- `pnpm lint:all` 无报错（tsc 严格模式下 `taskId: string | null` 的 null check 要正确处理）
- 两个 mutation hook 的现有调用方（DashboardPage）能正常编译（暂时传 undefined 即可，Step 11 再接入）

## 提交

```
feat(frontend): add useAnalysisTask polling hook, update rerun mutations to expose taskId
```
