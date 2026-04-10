# Step 7: AnalysisStepTimeline 组件

**设计依据：** TRD § 2.5（AnalysisStepTimeline props 和步骤图标映射）；PRD § 4.3（步骤定义）

**依赖：** Step 5（AnalysisTaskStep 类型已定义）

## 任务目标

新建 `AnalysisStepTimeline` 组件，渲染当前持仓的 5 个分析步骤，每个步骤根据 status 显示对应图标和耗时。

## 涉及文件

- 新建：`frontend/src/features/decision-card/components/AnalysisStepTimeline.tsx`

## 实施内容

参照 TRD § 2.5 的 props 定义：
- `steps: AnalysisTaskStep[]`
- `currentHolding: string`

**步骤图标规格（见 TRD § 2.5）：**
- `pending`：灰色空心圆（`background: #f0f0f0`，宽高 10px）
- `running`：蓝色实心圆（`#1677ff`）+ 内嵌白色小圆心 + CSS pulse 动画（见 TRD § 2.8 动效规格）
- `done`：绿色实心圆（`#52c41a`）+ 白色 ✓ 字符
- `failed`：红色实心圆（`#ff4d4f`）+ 白色 ✗ 字符

**步骤名称**：通过 `useTranslation("app")` 读取 `analysisProgress.step.${step.key}` 键

**当前步骤高亮**：`status === "running"` 的行添加蓝色背景（`background: #e6f4ff`，`borderRadius: 4px`）

**耗时格式**：`done` 或 `failed` 状态显示 `${durationMs}ms`，`running` 状态显示计时中（无需实时，轮询刷新即可）

**标题行**：渲染 `t("analysisProgress.currentSteps", { name: currentHolding })`

## 验证标准

- `pnpm lint:all` 无报错
- 组件在 TypeScript 严格模式下无类型错误

## 提交

```
feat(frontend): add AnalysisStepTimeline component
```
