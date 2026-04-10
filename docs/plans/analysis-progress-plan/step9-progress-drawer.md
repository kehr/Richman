# Step 9: AnalysisProgressDrawer 主组件

**设计依据：** TRD § 2.5（Drawer props 和区域划分）；PRD § 4.2（Drawer 完整结构）

**依赖：** Step 6（useAnalysisTask）、Step 7（AnalysisStepTimeline）、Step 8（AnalysisLogPanel）

## 任务目标

新建 `AnalysisProgressDrawer` 组件，组合 AnalysisStepTimeline 和 AnalysisLogPanel，实现进行中 / 完成（clean/degraded）/ 失败三种状态的完整 Drawer UI。

## 涉及文件

- 新建：`frontend/src/features/decision-card/components/AnalysisProgressDrawer.tsx`

## 实施内容

参照 TRD § 2.5 props：
- `taskId: string | null`
- `open: boolean`
- `onClose: () => void`

**Ant Design Drawer 配置（TRD § 2.5）：**
- `placement="right"`
- `mask={false}`（不阻断背景交互；在使用前先 grep `node_modules` 确认 Ant Design 6 的 `Drawer` 确实有该 prop）
- `width={280}`
- `closable={false}`
- `styles={{ body: { padding: 0, display: "flex", flexDirection: "column", height: "100%" } }}`

**内部数据**：通过 `useAnalysisTask(taskId)` 取得 `task`

**区域划分（TRD § 2.5 四个区）：**

1. **Header 区（`flex: 0 0 auto`）**
   - 进行中：`t("analysisProgress.title")` + 「收起 ›」按钮（调用 `onClose`）
   - 完成（clean）：绿色背景 + `t("analysisProgress.doneClean")` + 绿色「关闭」按钮
   - 完成（degraded）：橙色背景 + `t("analysisProgress.doneDegraded")` + 橙色「关闭」按钮
   - 失败：红色背景 + `t("analysisProgress.failed")` + 「关闭」按钮

2. **Overall 区（`flex: 0 0 auto`）**
   - 总进度条（progress 值来自 `task.progress`，`transition: width 0.5s ease`）
   - 持仓列表：每行显示圆点（颜色与 status 对应）+ 持仓名称 + 右侧状态文字
   - 降级卡片在 done 状态时额外显示橙色说明块（见 PRD § 4.2）

3. **Steps 区（`flex: 0 0 auto`）**
   - 仅在 `task.steps.length > 0` 时渲染
   - 渲染 `<AnalysisStepTimeline steps={task.steps} currentHolding={task.currentHolding} />`

4. **Log 区（`flex: 1 1 0`，`overflow: hidden`）**
   - 渲染 `<AnalysisLogPanel logs={task.logs ?? []} />`
   - 在 Log 区顶部加标签行 `t("analysisProgress.logs")`

**降级检测逻辑**：`task.holdings.some(h => h.synthesisSource === "template" || h.synthesisSource === "mixed")`

**CSS pulse 动画**（TRD § 2.8）：在组件顶部通过 `<style>` 或内联 `@keyframes` 注入，或放到全局 CSS，命名为 `analysis-pulse`，应用于 running 步骤圆点

## 验证标准

- `pnpm lint:all` 无报错
- 不传 taskId（null）时 Drawer 不显示（`open={false}` 由父控制）
- task 为 undefined（加载中）时不崩溃

## 提交

```
feat(frontend): add AnalysisProgressDrawer component
```
