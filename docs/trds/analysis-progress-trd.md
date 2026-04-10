# 分析进度与执行日志 TRD

设计依据：[analysis-progress-prd.md](../prds/analysis-progress-prd.md)

## 1. 后端架构

### 1.1 TaskStore 扩展

现有 `backend/internal/service/analysis/task_store.go` 是纯内存存储，`model.TaskStatus` 只有 `Status` / `Progress` / `Error`。需在内存层扩展，**不改 DB 表结构**（步骤和日志生命周期与任务等长，持久化收益低）。

**新增 model 类型（`backend/internal/model/task_status.go`）：**

```go
type TaskStepStatus string

const (
    StepPending TaskStepStatus = "pending"
    StepRunning TaskStepStatus = "running"
    StepDone    TaskStepStatus = "done"
    StepFailed  TaskStepStatus = "failed"
)

type TaskStep struct {
    Key        string         `json:"key"`
    Status     TaskStepStatus `json:"status"`
    DurationMs *int64         `json:"durationMs"`
    startedAt  time.Time      // 内部用，不序列化
}

type TaskLog struct {
    Ts    time.Time `json:"ts"`
    Level string    `json:"level"` // "info" | "warn" | "error"
    Msg   string    `json:"msg"`
}

type HoldingProgress struct {
    Symbol          string  `json:"symbol"`
    Name            string  `json:"name"`
    Status          string  `json:"status"` // pending|running|done|failed
    Progress        float64 `json:"progress"`
    SynthesisSource *string `json:"synthesisSource"`
    ProviderUsed    *string `json:"providerUsed"`
    DurationMs      *int64  `json:"durationMs"`
    startedAt       time.Time
}
```

**扩展 `TaskStatus`：**

```go
type TaskStatus struct {
    TaskID         string            `json:"taskId"`
    UserID         int64             `json:"userId"`
    Status         string            `json:"status"`
    Progress       float64           `json:"progress"`
    Error          string            `json:"error,omitempty"`
    StartedAt      time.Time         `json:"startedAt"`
    DoneAt         *time.Time        `json:"doneAt,omitempty"`
    CurrentHolding string            `json:"currentHolding"`
    Holdings       []HoldingProgress `json:"holdings"`
    Steps          []TaskStep        `json:"steps"`
    Logs           []TaskLog         `json:"logs"`
}
```

**新增 TaskStore 方法签名：**

```go
// 初始化 holdings 列表（分析开始前，symbol 顺序即执行顺序）
InitHoldings(taskID string, holdings []HoldingProgress)

// 切换当前分析持仓
SetCurrentHolding(taskID string, symbol string)

// 更新持仓分析状态
UpdateHoldingStatus(taskID string, symbol string, status string, source *string, provider *string, durationMs *int64)

// 步骤推进（按 key 找到对应 step，设为 running）
StartStep(taskID string, key string)

// 步骤完成（设为 done，计算 durationMs）
CompleteStep(taskID string, key string)

// 步骤失败
FailStep(taskID string, key string)

// 追加日志
AppendLog(taskID string, level string, msg string)
```

### 1.2 步骤 key 常量（`backend/internal/model/task_status.go`）

```go
const (
    StepKeyFetchData      = "fetch_data"
    StepKeyCalcIndicators = "calc_indicators"
    StepKeyRecommendation = "recommendation"
    StepKeyLLMSynthesis   = "llm_synthesis"
    StepKeyPersist        = "persist"
)

// DefaultSteps 初始化时写入 TaskStatus.Steps
func DefaultSteps() []TaskStep {
    keys := []string{
        StepKeyFetchData,
        StepKeyCalcIndicators,
        StepKeyRecommendation,
        StepKeyLLMSynthesis,
        StepKeyPersist,
    }
    steps := make([]TaskStep, len(keys))
    for i, k := range keys {
        steps[i] = TaskStep{Key: k, Status: StepPending}
    }
    return steps
}
```

### 1.3 Service 层插桩

`backend/internal/service/analysis/service.go` 的 goroutine 内在各阶段前后插入 TaskStore 调用：

```
InitHoldings → 写入持仓列表

for each holding:
  SetCurrentHolding(symbol)
  UpdateHoldingStatus(symbol, "running", ...)
  AppendLog("info", "[symbol] fetch data")

  StartStep(StepKeyFetchData)
  → fetch data
  CompleteStep(StepKeyFetchData)
  AppendLog("info", "[symbol] fetch ok · trend=X")

  StartStep(StepKeyCalcIndicators)
  → trend + position + catalyst + weights
  CompleteStep(StepKeyCalcIndicators)

  StartStep(StepKeyRecommendation)
  → confidence + recommendation matrix
  CompleteStep(StepKeyRecommendation)

  StartStep(StepKeyLLMSynthesis)
  AppendLog("info", "[symbol] LLM call [provider]")
  → synthesize (LLM or fallback)
  if fallback:
    AppendLog("warn", "[symbol] LLM timeout/error, fallback → template")
  CompleteStep(StepKeyLLMSynthesis)

  StartStep(StepKeyPersist)
  → persist raw + persist card
  CompleteStep(StepKeyPersist)

  UpdateHoldingStatus(symbol, "done", source, provider, durationMs)
  AppendLog("info", "[symbol] done · source=X provider=Y")

taskStore.Complete(taskID)
```

注意：每张持仓分析时 `Steps` 重置（`DefaultSteps()`），因为步骤是「当前卡片」粒度的。

### 1.4 新增 API 端点

**`GET /api/v1/analysis/tasks/:taskId`**

Handler（`backend/internal/api/v1/analysis.go`）：

```go
func (h *AnalysisHandler) GetTask(c *gin.Context) {
    taskID := c.Param("taskId")
    userID := middleware.GetUserID(c)
    task := h.taskStore.Get(taskID)
    if task == nil || task.UserID != userID {
        c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": task})
}
```

路由注册（`backend/internal/api/router.go`）：

```go
analysis.GET("/tasks/:taskId", analysisHandler.GetTask)
```

TaskStore 的 `Get` 方法需加读锁（现有实现若已加锁则无需改动）。

## 2. 前端架构

### 2.1 新增类型（`frontend/src/features/decision-card/types.ts`）

```typescript
export type TaskStepKey =
    | "fetch_data"
    | "calc_indicators"
    | "recommendation"
    | "llm_synthesis"
    | "persist";

export type TaskStepStatus = "pending" | "running" | "done" | "failed";

export interface AnalysisTaskStep {
    key: TaskStepKey;
    status: TaskStepStatus;
    durationMs: number | null;
}

export type HoldingAnalysisStatus = "pending" | "running" | "done" | "failed";

export interface HoldingProgress {
    symbol: string;
    name: string;
    status: HoldingAnalysisStatus;
    progress: number;
    synthesisSource: "llm" | "template" | "mixed" | null;
    providerUsed: "user" | "system_default" | "none" | null;
    durationMs: number | null;
}

export interface AnalysisTaskLog {
    ts: string;
    level: "info" | "warn" | "error";
    msg: string;
}

export type AnalysisTaskStatus = "running" | "done" | "failed";

export interface AnalysisTask {
    taskId: string;
    status: AnalysisTaskStatus;
    progress: number;
    currentHolding: string;
    holdings: HoldingProgress[];
    steps: AnalysisTaskStep[];
    logs: AnalysisTaskLog[];
    error?: string;
}

export interface RerunAnalysisResponse {
    taskId: string;
    message: string;
}

export interface ReanalyzeAllResponse {
    taskId: string;
    message: string;
}
```

### 2.2 API 函数（`frontend/src/features/decision-card/api.ts`）

新增：

```typescript
export function getAnalysisTask(taskId: string): Promise<ApiResponse<AnalysisTask>> {
    return request<ApiResponse<AnalysisTask>>(`/analysis/tasks/${taskId}`);
}
```

### 2.3 `useAnalysisTask` hook（`frontend/src/features/decision-card/use-analysis-task.ts`）

```typescript
// 契约
export function useAnalysisTask(taskId: string | null): {
    task: AnalysisTask | undefined;
    isPolling: boolean;
}
```

实现要点：
- 使用 `useQuery`，`queryKey: ["analysis-task", taskId]`
- `enabled: taskId !== null`
- `refetchInterval: (query) => query.state.data?.status === "running" ? 1500 : false`
- `staleTime: 0`（每次轮询都要新鲜数据）
- 当 `status === "done"` 时调用 `queryClient.invalidateQueries(DECISION_CARDS_QUERY_KEY)`，触发卡片刷新（通过 `useEffect` 监听 status 变化）
- `isPolling`: `taskId !== null && task?.status === "running"`

### 2.4 修改 `useRerunAnalysis` / `useReanalyzeAll`

两个 hook 均改为：
- `onSuccess` 不再直接 `invalidateQueries`（改由 `useAnalysisTask` 在完成时触发）
- 通过 `onSuccess` 回调将 `taskId` 传出，由调用方存入本地 state

```typescript
// useRerunAnalysis 改动后的 onSuccess
onSuccess: (data) => {
    onTaskStarted?.(data.data.taskId);
}
// 新增可选 prop
export function useRerunAnalysis(onTaskStarted?: (taskId: string) => void)
```

`useReanalyzeAll` 同理。

### 2.5 组件设计

**`AnalysisProgressDrawer`**（`features/decision-card/components/AnalysisProgressDrawer.tsx`）

```typescript
interface AnalysisProgressDrawerProps {
    taskId: string | null;
    open: boolean;
    onClose: () => void;
}
```

使用 Ant Design `Drawer`：
- `placement="right"`
- `mask={false}` — 不遮挡背景，用户可继续操作页面
- `width={280}`
- `styles={{ body: { padding: 0, display: "flex", flexDirection: "column" } }}`
- `closable={false}` — 用自定义关闭按钮（完成态显示绿/橙色「关闭」）

Drawer 内部状态由 `useAnalysisTask(taskId)` 驱动，无本地 state。

内部区域划分：
1. Header（`flex: 0 0 auto`）：标题 + 收起/关闭按钮
2. Overall section（`flex: 0 0 auto`）：总进度条 + 持仓列表
3. Steps section（`flex: 0 0 auto`）：当前持仓步骤时间轴
4. Log section（`flex: 1 1 0`，`overflow-y: auto`）：执行日志

**`AnalysisStepTimeline`**（`features/decision-card/components/AnalysisStepTimeline.tsx`）

```typescript
interface AnalysisStepTimelineProps {
    steps: AnalysisTaskStep[];
    currentHolding: string;
}
```

步骤图标映射：
- `pending`：灰色空心圆（`background: #f0f0f0`）
- `running`：蓝色实心圆 + 白色小圆心（pulse 动画）
- `done`：绿色实心圆 + 白色 ✓
- `failed`：红色实心圆 + 白色 ✗

**`AnalysisLogPanel`**（`features/decision-card/components/AnalysisLogPanel.tsx`）

```typescript
interface AnalysisLogPanelProps {
    logs: AnalysisTaskLog[];
}
```

- `overflow-y: auto`，`flex: 1`
- 新日志追加时自动滚动到底部（`useEffect` + `scrollTop = scrollHeight`）
- 日志行颜色：`info` → `#555`，`warn` → `#fa8c16`，`error` → `#ff4d4f`
- 时间格式：`HH:mm:ss`（从 `ts` 解析）

### 2.6 Briefing 页面改动（`frontend/src/pages/briefing/BriefingPage.tsx`）

新增本地 state：

```typescript
const [taskId, setTaskId] = useState<string | null>(null);
const [drawerOpen, setDrawerOpen] = useState(false);
```

按钮渲染逻辑（伪代码）：

```typescript
const { task } = useAnalysisTask(taskId);
const isRunning = task?.status === "running";
const isDone = task?.status === "done";
const hasDegraded = isDone && task.holdings.some(h => h.synthesisSource === "template");

// 按钮
if (isRunning) → 蓝底，显示 `分析中 ${Math.round(task.progress * 100)}%`，onClick = setDrawerOpen(true)
if (isDone && drawerOpen) → 绿/橙底，显示 `分析完成` / `分析完成（含降级）`，onClick = setDrawerOpen(true)
else → 默认按钮，onClick = triggerAnalysis()
```

触发分析：

```typescript
const rerun = useRerunAnalysis((id) => {
    setTaskId(id);
    setDrawerOpen(true);
});
const reanalyzeAll = useReanalyzeAll((id) => {
    setTaskId(id);
    setDrawerOpen(true);
});
```

Drawer 关闭时：`setDrawerOpen(false)` + `setTaskId(null)`（重置状态，按钮恢复默认）。

### 2.7 `DecisionCardSummary` 改动

接受新 prop：

```typescript
interface DecisionCardSummaryProps {
    card: DecisionCardDTO;
    onClick?: () => void;
    analysisStatus?: HoldingAnalysisStatus; // 新增，来自 task.holdings 匹配 card.symbol
}
```

根据 `analysisStatus`：
- `"running"`：`Card` 添加 `style={{ borderColor: "#91caff", borderWidth: 1.5 }}`，右上角渲染 `更新中…` Badge
- `"done"`：`Card` 短暂添加 `style={{ borderColor: "#b7eb8f", background: "#f6ffed" }}`，2 秒后通过 setTimeout 清除（本地 state `justUpdated`）

### 2.8 CSS 动效规格

**脉冲动画**（running 步骤圆点）：

```css
@keyframes analysis-pulse {
    0%   { box-shadow: 0 0 0 0 rgba(22, 119, 255, 0.4); }
    70%  { box-shadow: 0 0 0 6px rgba(22, 119, 255, 0); }
    100% { box-shadow: 0 0 0 0 rgba(22, 119, 255, 0); }
}
/* animation: analysis-pulse 1.5s infinite */
```

**进度条过渡**：`transition: width 0.5s ease`

**卡片绿色闪烁**：`setTimeout(() => setJustUpdated(false), 2000)`

### 2.9 i18n 键（前端）

在 `src/i18n/locales/zh/app.json` 和 `en/app.json` 中，新增 `analysisProgress` 节点：

```json
{
  "analysisProgress": {
    "title": "分析进度",
    "overall": "总进度",
    "cardCount": "{{done}} / {{total}} 张",
    "collapse": "收起",
    "close": "关闭",
    "currentSteps": "{{name}} · 当前步骤",
    "logs": "执行日志",
    "doneClean": "分析完成",
    "doneDegraded": "分析完成（含降级）",
    "failed": "分析失败",
    "degradedWarning": "{{name}} 本次使用规则模板，不含 AI 深度解读",
    "updating": "更新中…",
    "pollError": "无法获取进度",
    "buttonRunning": "分析中 {{pct}}%",
    "step": {
      "fetch_data": "获取数据",
      "calc_indicators": "趋势 / 仓位 / 催化剂",
      "recommendation": "推荐决策",
      "llm_synthesis": "LLM 合成内容",
      "persist": "保存结果"
    },
    "source": {
      "llm": "LLM",
      "template": "规则模板",
      "mixed": "混合"
    }
  }
}
```

## 3. 关键约束与注意点

1. `TaskStore.Get` 需要对 `Holdings` / `Steps` / `Logs` slice 做防御性拷贝再返回，避免并发读写 race
2. `Logs` slice 在内存中无上限，若分析时间极长可能膨胀；MVP 阶段不做 cap，后续可加 `maxLogs=200` 截断
3. `Steps` 在每张持仓分析开始时通过 `DefaultSteps()` 重置，前端轮询时不做本地 diff，直接替换渲染
4. `Drawer` 的 `mask={false}` 模式下，Ant Design 仍会渲染一个透明遮罩 DOM 节点；如需完全无遮罩，需确认 Ant Design 6 的 `mask` prop 行为（在 node_modules 中 grep 验证）
5. 前端 `useAnalysisTask` 只在 `taskId !== null` 时启动查询，避免无效请求
6. 卡片 `justUpdated` 的 `setTimeout` 需在组件卸载时 `clearTimeout`，防止内存泄漏
