# 跨层 DTO 契约纪律（MANDATORY）

> 本规范适用于跨越多个语言/服务描述同一 JSON payload 的场景。
> 典型链路：richson（Python/pydantic）→ backend（Go struct）→ frontend（TypeScript interface）。
> 这个链路上任何一个节点的字段命名、可空性、类型语义发生偏差，都会在运行时表现为「编译通过但显示 undefined / NaN / 0 / 空白」，无法被任何单侧的 lint 拦截。

## 触发条件

以下任一情况属于本规范约束范围，改动前必须走字段对齐检查：

1. 修改 richson 的 `schemas/*.py`、任何 `@router.*` 返回 dict 的字段
2. 修改 backend 的 `internal/richson/types.go`、`internal/api/v*/` 里直接 `c.JSON` 的结构体
3. 修改 frontend 的 `features/*/types.ts` / `features/*/api.ts` 返回类型
4. 新增跨层透传端点（backend 代理 richson、frontend 消费 backend）

## 三类禁止模式

### 1. 命名漂移

每层各自按自己的语义命名相同字段，链路上任何一段走到 JSON 边界都会字面穿透。

错误示例：
- richson 返回 `impact` / `probability` / `probabilityChange24h`
- backend `EventItem` 透传 `impact` / `probability` / `probabilityChange24h`
- frontend `EventDto` 声明 `impactLevel` / `polymarketProbability` / `polymarketChange24h`

结果：frontend 读到的 `impactLevel === undefined`，`probability * 100 === NaN`。TS 编译不报错（类型正确，但运行时 key miss）。

### 2. Null 语义塌陷

Go `float64` / `string` 对 JSON `null` 默认 unmarshal 为零值（0 / ""），再 marshal 出去就是 `0` / `""`，null 信息永久丢失。

错误示例：
- richson schema 声明 `probability: float | None = None`
- backend 透传结构体声明 `Probability float64 json:"probability"`
- frontend 按 `number | null` 渲染

结果：richson 返回 `null` → backend 序列化为 `0` → frontend 显示「概率 0%」。视觉上没有报错，但语义完全错。

### 3. 静默默认值

pydantic `Field(default=...)` / Go zero value / TS 可选字段 `?:`：如果有一层用默认值填充了缺失字段，后续层无法区分「没这个字段」和「这个值就是默认值」。

## 对齐规则（必须满足）

| 维度 | Python (pydantic) | Go | TypeScript |
|---|---|---|---|
| 必填 string | `field: str` | `Field string` | `field: string` |
| 可空 string | `field: str \| None` | `Field *string` | `field: string \| null` |
| 必填 float | `field: float` | `Field float64` | `field: number` |
| 可空 float | `field: float \| None` | `Field *float64` | `field: number \| null` |
| 枚举 | `Literal["a", "b"]` | `Field string`（注释列举） | `"a" \| "b"` |

**关键点**：Python 的 `T | None` 必须映射为 Go 指针类型，绝对不能用值类型 + 约定「零值表示缺失」。

## 三层改动的强制步骤

新增或修改任何跨层字段时，必须**同一个 PR** 同步改三处：

1. richson：`schemas/<domain>.py` 中 `BaseModel`
2. backend：`internal/richson/types.go` 中对应 struct
3. frontend：`features/<domain>/types.ts` 中 interface

推迟任意一层都算契约漂移。如果 PR 只改一层，reviewer 必须拒绝合并。

## 审查清单（PR 自查和 reviewer 强制）

| 检查项 | 方法 |
|---|---|
| 字段名三端完全一致 | `grep` 各层定义文件，逐字段核对 |
| 可空字段三端都表达了 nullable | Python `\| None`、Go `*T`、TS `\| null` 同时存在 |
| 枚举取值三端同源 | frontend `Literal` / Go 注释 / Python `Literal` 值集一致 |
| 没有用 Go 值类型 + 注释暗示 nullable | 搜 `json:".*"` 看有无可空字段用了值类型 |
| 新增字段 backend 代理层透传了 | `backend/internal/richson/*.go` 中 struct 包含了新字段 |
| frontend 防御 `null \| undefined` | 判空用 `!= null` 或 `typeof x === "number"`，不用 `!== null` |

## 验证检验点

任何跨层契约改动合并前，必须至少人工验证一次**端到端实际响应**：

1. 本地起 richson + backend + frontend
2. 打开 DevTools Network，访问相关页面
3. 在 backend 返回里检查字段名和类型
4. 在 frontend 组件中 `console.log` 或 DevTools debug 关键字段值

不允许只靠三层 TypeScript/Go/Python 各自的类型检查通过就 merge。类型系统不会跨 JSON 边界校验。

## 历史教训

- 2026-04-15：事件雷达 Event Radar 面板上线时，richson 返回 `impact/probability/probabilityChange24h`，frontend 声明 `impactLevel/polymarketProbability/polymarketChange24h`。前端渲染出 `NaN%`、`NaNpp`、`impactLevel.undefined`。三端类型检查都通过，漂移到生产前才发现。修复时同步把 backend `Probability float64` 升级为 `*float64`，保住 null 语义。
