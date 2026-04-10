# Step 10: DecisionCardSummary 更新中状态

**设计依据：** TRD § 2.7（analysisStatus prop 和卡片状态变化规格）；PRD § 4.4（卡片状态变化）

**依赖：** Step 5（HoldingAnalysisStatus 类型已定义）

## 任务目标

为 `DecisionCardSummary` 新增 `analysisStatus` 可选 prop，当为 `"running"` 时渲染蓝色边框 + 右上角「更新中…」角标；当为 `"done"` 时渲染 2 秒绿色闪烁后恢复正常。

## 涉及文件

- 修改：`frontend/src/features/decision-card/components/DecisionCardSummary.tsx`

## 实施内容

**Prop 新增（TRD § 2.7）：**
- `analysisStatus?: HoldingAnalysisStatus`（可选，不传时行为与现在完全相同）

**`running` 状态：**
- `Card` 的 `style` 加上 `borderColor: "#91caff"` 和 `borderWidth: 1.5`（需确认 Ant Design 6 Card 的 style prop 支持 borderColor，若不支持则改用 `className` + CSS）
- 右上角渲染一个绝对定位的 badge 元素，显示 `t("analysisProgress.updating")`（蓝色背景、7px 字号）

**`done` 状态（TRD § 2.7）：**
- 新增本地 state `justUpdated: boolean`，初始 `false`
- `useEffect` 监听 `analysisStatus`：当值变为 `"done"` 时 `setJustUpdated(true)`，同时 `setTimeout(() => setJustUpdated(false), 2000)` — 注意在 cleanup 中 `clearTimeout`
- `justUpdated === true` 时：`Card` style 改为 `borderColor: "#b7eb8f"`、`background: "#f6ffed"`

**`running` 状态底部细进度条（PRD § 4.4）：**
- Card 底部渲染一个 2px 高的进度条容器，宽度 100%，背景 `#f0f0f0`
- 内层填充条宽度 = `analysisProgress * 100%`（通过额外 prop `analysisProgress?: number` 传入，0-1）
- `transition: width 0.5s ease`

**状态优先级**：`running` > `justUpdated（done）` > 正常

## 验证标准

- `pnpm lint:all` 无报错
- `analysisStatus` 为 `undefined` 时组件行为与改动前完全一致（不影响现有渲染）

## 提交

```
feat(frontend): add analysisStatus prop to DecisionCardSummary for in-progress card state
```
