# Step 8: AnalysisLogPanel 组件

**设计依据：** TRD § 2.5（AnalysisLogPanel props 和自动滚动规格）

**依赖：** Step 5（AnalysisTaskLog 类型已定义）

## 任务目标

新建 `AnalysisLogPanel` 组件，渲染可滚动的 monospace 执行日志列表，新日志追加时自动滚到底部。

## 涉及文件

- 新建：`frontend/src/features/decision-card/components/AnalysisLogPanel.tsx`

## 实施内容

参照 TRD § 2.5 props 定义：
- `logs: AnalysisTaskLog[]`

**样式规格：**
- 容器：`flex: 1 1 0`、`overflow-y: auto`、`padding: 8px 14px`
- 字体：`font-family: monospace`、`font-size: 12px`、`line-height: 1.7`

**行颜色（TRD § 2.5）：**
- `level === "info"` → `color: #555`
- `level === "warn"` → `color: #fa8c16`
- `level === "error"` → `color: #ff4d4f`

**时间格式**：从 `log.ts`（ISO 字符串）解析出 `HH:mm:ss` 显示，时间部分颜色 `#bbb`

**自动滚动**：`useRef` 指向容器 div；`useEffect` 在 `logs` 变化时执行 `containerRef.current.scrollTop = containerRef.current.scrollHeight`

**标题行**：渲染 `t("analysisProgress.logs")`（标题在 Drawer 内 header 区显示，Panel 本身不含标题，由父组件决定是否显示标签）

## 验证标准

- `pnpm lint:all` 无报错
- 日志为空时组件渲染正常（空列表不报错）

## 提交

```
feat(frontend): add AnalysisLogPanel component with auto-scroll
```
