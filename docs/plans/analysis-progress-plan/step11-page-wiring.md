# Step 11: DashboardPage + DashboardTopStrip 接入 + Barrel 导出 + 最终 lint

**设计依据：** TRD § 2.6（BriefingPage 改动）；PRD § 4.1（按钮状态）

**依赖：** Step 6（mutations 已可返回 taskId）、Step 9（AnalysisProgressDrawer 已就绪）、Step 10（DecisionCardSummary 已支持 analysisStatus）

## 任务目标

在 `DashboardPage`（路由 `/briefing`）中接入 taskId state、AnalysisProgressDrawer 和按钮状态逻辑；更新 `DashboardTopStrip` 渲染分析中/完成态按钮；将新组件和 hook 加入 barrel；最后跑完整 lint 通过。

## 涉及文件

- 修改：`frontend/src/pages/dashboard/DashboardPage.tsx`
- 修改：`frontend/src/pages/dashboard/components/DashboardTopStrip.tsx`
- 修改：`frontend/src/features/decision-card/index.ts`

## 实施内容

**`DashboardPage.tsx`（TRD § 2.6）：**

新增 state：
- `const [taskId, setTaskId] = useState<string | null>(null)`
- `const [drawerOpen, setDrawerOpen] = useState(false)`

改造 mutation 调用：
- `useRerunAnalysis((id) => { setTaskId(id); setDrawerOpen(true); })`
- `useReanalyzeAll((id) => { setTaskId(id); setDrawerOpen(true); })`
- `handleRerun` 和 `handleReanalyzeAll` 中移除手动 `message.success`（完成状态由 Drawer 展示）；保留 `message.error` 用于触发失败

获取 task 数据：`const { task } = useAnalysisTask(taskId)`

计算按钮状态（TRD § 2.6 按钮逻辑）：
- `isRunning`: `task?.status === "running"`
- `isDone`: `task?.status === "done"`
- `hasDegraded`: `isDone && task.holdings.some(h => h.synthesisSource === "template" || h.synthesisSource === "mixed")`

将状态信息和 `onOpenDrawer` 回调传给 `DashboardTopStrip`；将 `AnalysisProgressDrawer` 放在 JSX 末尾：
```tsx
<AnalysisProgressDrawer
  taskId={taskId}
  open={drawerOpen}
  onClose={() => { setDrawerOpen(false); setTaskId(null); }}
/>
```

将 task holdings 传给 `DecisionCardWall`，在渲染 `DecisionCardSummary` 时匹配 `card.symbol`（或等价字段）传入 `analysisStatus` 和 `analysisProgress`（holding 的 `progress` 字段，0-1）。

**`DashboardTopStrip.tsx`：**

新增 props（替换现有 `rerunLoading: boolean`）：
- `isRunning: boolean`
- `isDone: boolean`
- `hasDegraded: boolean`
- `taskProgress: number`（用于显示 `分析中 N%`）
- `onOpenDrawer: () => void`

按钮渲染逻辑（TRD § 2.6 按钮 4 种状态）：
- `isRunning`：蓝底按钮，`t("analysisProgress.buttonRunning", { pct: Math.round(taskProgress * 100) })`，`onClick = onOpenDrawer`，禁用 loading 状态改为文字
- `isDone && drawerOpen`：绿/橙底，`t("analysisProgress.doneClean" / "doneDegraded")`，`onClick = onOpenDrawer`
- 默认：保持现有「最新分析」按钮样式，`onClick = onRerun`

**`index.ts` barrel 导出：**

新增：
- `export { useAnalysisTask } from "./use-analysis-task"`
- `export { AnalysisProgressDrawer } from "./components/AnalysisProgressDrawer"`
- `export type { AnalysisTask, HoldingProgress, AnalysisTaskStep, AnalysisTaskLog, HoldingAnalysisStatus, AnalysisTaskStatus, TaskStepKey, TaskStepStatus } from "./types"`

## 验证标准

- `cd frontend && pnpm lint:all` 全部通过（Biome + tsc strict + depcruiser，零报错）
- 无新增 hardcoded 中文/英文字符串（全部走 i18n）
- `cd backend && make check` 全部通过

## 提交

```
feat(frontend): wire analysis progress drawer into DashboardPage
```

---

## 最终验收清单（全部 step 完成后）

- [ ] 触发「最新分析」后按钮变为蓝色「分析中 N%」，Drawer 自动打开
- [ ] Drawer 总进度、持仓列表、步骤时间轴随轮询实时更新（~1.5s 间隔可见变化）
- [ ] 执行日志逐条追加，新条目自动滚到底部
- [ ] 全 LLM 成功：Drawer header 绿色「分析完成」，持仓列表显示 `LLM ✓ Xs`
- [ ] LLM 降级：Drawer header 橙色「分析完成（含降级）」，降级持仓显示 `⚠ template fallback`，日志可见 timeout 条目
- [ ] 正在分析的卡片：蓝色边框 + 「更新中…」角标
- [ ] 分析完成：卡片绿色闪 2s 后恢复正常
- [ ] 用户点「关闭」后按钮恢复默认，Drawer 关闭
- [ ] `pnpm lint:all` 零报错，`make check` 零报错
