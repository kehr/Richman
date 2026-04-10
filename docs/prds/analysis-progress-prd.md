# 分析进度与执行日志 PRD

## 1. 背景与目标

Richman 的分析流程是异步的：用户点击「最新分析」后后端返回 202，分析在后台 goroutine 中执行（可达数十秒）。当前前端仅弹一个 toast 通知，用户无法感知进度，也无法判断本次分析是 AI 生成还是规则模板降级生成。

目标：在不打断页面浏览的前提下，让用户随时感知分析进度、了解每张卡片的分析状态与结果来源，并在发生 LLM 降级时获得明确提示。

## 2. 用户故事

- 我触发重新分析后，想知道「现在在做什么、做到哪一步了」，不需要盯着页面等
- 我想知道分析完成后，是 AI 生成的还是规则模板生成的，两者对我的决策参考价值不同
- 如果 LLM 超时或失败，我希望看到明确的警告，而不是悄悄降级
- 分析完成后，我想立刻看到刷新后的卡片内容

## 3. 功能范围

### 3.1 在 scope

- 触发按钮状态变化（默认 / 分析中 / 完成 / 恢复）
- 右侧 Fixed Overlay Drawer（进度面板）
- 卡片「更新中」状态角标
- 执行日志实时追加
- LLM 降级可见性（橙色警告 + 日志）
- 前端轮询 task 状态接口
- 后端补充 task 步骤日志接口（返回结构化步骤列表）

### 3.2 不在 scope

- WebSocket / SSE 实时推送（MVP 用轮询）
- 分析历史归档页面
- 单卡片独立重跑的独立进度视图（与全量重跑共用同一套 Drawer）

## 4. 交互设计

### 4.1 触发按钮状态

| 状态 | 外观 | 触发条件 |
|------|------|---------|
| 默认 | 白底边框「最新分析」 | 无进行中任务 |
| 分析中 | 蓝底「分析中 N%」+ 脉冲白点 | taskId 存在且 status=running |
| 完成 | 绿底「分析完成」/ 橙底「分析完成（含降级）」 | status=done，Drawer 未关闭 |
| 恢复 | 回到默认状态 | 用户关闭 Drawer 后 |

分析中状态下点击按钮 = 打开 Drawer（而非重复触发分析）。

### 4.2 Fixed Overlay Drawer

- 定位：`position: fixed`，右侧贴边，覆盖在卡片网格上方，不压缩布局
- 宽度：280px
- 打开时机：触发分析后自动打开；分析中点击按钮也可打开
- 关闭时机：用户主动点「收起」或「关闭」按钮

**Drawer 内部结构（从上到下）：**

```
┌─────────────────────────────┐
│ 分析进度            [收起 ›] │  ← header
├─────────────────────────────┤
│ 总进度              1 / 2 张 │
│ ████████████░░░░░░░░  80%   │  ← overall section
│ ● Gold ETF       LLM ✓ 2.1s │
│ ● SSE 50 ETF     64%…       │
├─────────────────────────────┤
│ SSE 50 ETF · 当前步骤        │
│ ✓ 获取数据              0.3s │
│ ✓ 趋势/仓位/催化剂       0.8s │  ← current card steps
│ ✓ 推荐决策              0.2s │
│ → LLM 合成内容         3.2s… │  (active step highlighted)
│ ○ 保存结果                   │
├─────────────────────────────┤
│ 执行日志                     │
│ 00:53:01 [gold] fetch ok    │  ← scrollable log
│ 00:53:02 [gold] LLM ok      │
│ 00:53:03 [sse50] fetch ok   │
│ 00:53:04 [sse50] LLM call…  │
└─────────────────────────────┘
```

**完成状态（全部 LLM 成功）：**
- Header 变绿，显示「✓ 分析完成」
- 卡片列表显示每张「LLM ✓ Xs」
- 右上角出现绿色「关闭」按钮

**完成状态（含降级）：**
- Header 变橙，显示「⚠ 分析完成（含降级）」
- 降级卡片显示「⚠ template fallback」
- 橙色区块说明：「SSE 50 ETF 本次使用规则模板，不含 AI 深度解读」
- 日志中可见 `LLM timeout` 和 `fallback → template` 条目
- 关闭按钮为橙色

### 4.3 分析步骤定义

后端 11 个分析阶段映射为前端 5 个显示步骤（合并技术细节）：

| 前端显示步骤 | 对应后端阶段 | progress 区间 |
|------------|------------|-------------|
| 获取数据 | fetch data | 0 – 0.1 |
| 趋势 / 仓位 / 催化剂 | trend + position + valuation + catalyst + LLM enhance catalyst + weights | 0.1 – 0.5 |
| 推荐决策 | confidence + recommendation | 0.5 – 0.7 |
| LLM 合成内容 | synthesize card | 0.7 – 0.9 |
| 保存结果 | persist raw + persist card | 0.9 – 1.0 |

### 4.4 卡片状态变化

```
旧内容（正常）→ 蓝色边框 + 「更新中…」角标 → 绿色边框闪 2s → 正常
```

- 等待队列中的卡片：无角标，旧内容照常显示
- 当前分析中的卡片：`border: 1.5px solid #91caff` + 右上角 badge「更新中…」+ 底部细进度条
- 刚完成的卡片：短暂 `border: 1.5px solid #b7eb8f` + 背景 `#f6ffed`，2 秒后恢复

## 5. 数据流

### 5.1 前端轮询流程

```
用户点击「最新分析」
  → POST /api/v1/analysis/trigger
  → 返回 { taskId, message }
  → 本地存储 taskId，打开 Drawer，启动轮询

每 1.5s 轮询：
  GET /api/v1/analysis/tasks/{taskId}
  → { status, progress, steps[], logs[], currentCard }

  status=running → 更新进度条 + 步骤 + 日志
  status=done    → 停止轮询，切换完成状态，失效 TanStack Query 缓存
  status=failed  → 停止轮询，显示错误状态
```

### 5.2 需要后端补充的接口

**GET /api/v1/analysis/tasks/{taskId}**

当前 `analysis_tasks` 表已有 `status` / `progress` / `error`，需补充：
- `current_holding`：当前分析的持仓 symbol
- `steps`：结构化步骤列表，每项含 `key`（前端用于 i18n 查询 `analysisProgress.step.<key>`）、`status`、`durationMs`，不含 `label`
- `logs`：结构化日志条目列表
- `holdings[].status` 取值：`pending`（排队）/ `running`（进行中）/ `done`（完成）/ `failed`（失败）

响应结构：

```json
{
  "data": {
    "taskId": "uuid",
    "status": "running | done | failed",
    "progress": 0.64,
    "currentHolding": "SSE 50 ETF",
    "holdings": [
      {
        "symbol": "gold_etf",
        "name": "Gold ETF",
        "status": "done",
        "synthesisSource": "llm",
        "providerUsed": "user",
        "durationMs": 2100
      },
      {
        "symbol": "a_share_broad",
        "name": "SSE 50 ETF",
        "status": "running",  // pending | running | done | failed
        "progress": 0.64,
        "synthesisSource": null,
        "providerUsed": null,
        "durationMs": null
      }
    ],
    "steps": [
      { "key": "fetch_data", "status": "done", "durationMs": 312 },
      { "key": "calc_indicators", "status": "done", "durationMs": 821 },
      { "key": "recommendation", "status": "done", "durationMs": 203 },
      { "key": "llm_synthesis", "status": "running", "durationMs": null },
      { "key": "persist", "status": "pending", "durationMs": null }
    ],
    "logs": [
      { "ts": "2026-04-11T00:53:01Z", "level": "info", "msg": "[gold] fetch ok · trend=up" },
      { "ts": "2026-04-11T00:53:02Z", "level": "info", "msg": "[gold] LLM ok · provider=user" },
      { "ts": "2026-04-11T00:53:03Z", "level": "info", "msg": "[sse50] fetch ok · trend=up" },
      { "ts": "2026-04-11T00:53:04Z", "level": "warn", "msg": "[sse50] LLM call [user]…" }
    ]
  }
}
```

### 5.3 前端新增模块

| 模块 | 位置 | 职责 |
|------|------|------|
| `useAnalysisTask` hook | `features/decision-card/` | 封装轮询逻辑，返回 task 状态 |
| `AnalysisProgressDrawer` | `features/decision-card/components/` | Drawer UI 组件 |
| `AnalysisStepTimeline` | `features/decision-card/components/` | 步骤时间轴子组件 |
| `AnalysisLogPanel` | `features/decision-card/components/` | 可滚动日志面板子组件 |

`useRerunAnalysis` / `useReanalyzeAll` 改为在成功后返回 `taskId` 而非立即失效缓存；缓存失效改为由 `useAnalysisTask` 在 `status=done` 时触发。

## 6. 状态机

```
IDLE
  → [用户点击「最新分析」] → RUNNING
RUNNING
  → [轮询 status=done，无降级] → DONE_CLEAN
  → [轮询 status=done，有降级] → DONE_DEGRADED
  → [轮询 status=failed] → FAILED
DONE_CLEAN / DONE_DEGRADED / FAILED
  → [用户点「关闭」] → IDLE
```

## 7. 错误处理

| 场景 | 处理方式 |
|------|---------|
| LLM 超时 → template fallback | Drawer 橙色警告，日志显示 timeout + fallback 条目，`synthesisSource=template` |
| 整体分析失败（status=failed） | Drawer 显示红色错误态，error 字段内容展示，按钮恢复默认 |
| 轮询请求失败（网络） | 静默重试，超过 5 次后停止轮询并展示「无法获取进度」提示 |
| 页面刷新丢失 taskId | 按钮恢复默认，Drawer 不显示（不持久化 taskId 到 localStorage） |

## 8. i18n 键

所有新增字符串均需同步更新 `zh/app.json` 和 `en/app.json`：

```
analysisProgress.title
analysisProgress.overall
analysisProgress.cardCount          （「1 / 2 张」）
analysisProgress.collapse
analysisProgress.close
analysisProgress.currentSteps
analysisProgress.logs
analysisProgress.doneClean          （「✓ 分析完成」）
analysisProgress.doneDegraded       （「⚠ 分析完成（含降级）」）
analysisProgress.failed             （「分析失败」）
analysisProgress.degradedWarning    （「本次使用规则模板，不含 AI 深度解读」）
analysisProgress.updating           （卡片角标「更新中…」）
analysisProgress.pollError          （「无法获取进度」）
analysisProgress.step.fetch_data
analysisProgress.step.calc_indicators
analysisProgress.step.recommendation
analysisProgress.step.llm_synthesis
analysisProgress.step.persist
analysisProgress.source.llm
analysisProgress.source.template
analysisProgress.source.mixed
```

## 9. 验收标准

1. 触发分析后按钮立即变为「分析中 N%」，Drawer 自动打开
2. Drawer 总进度、卡片列表、当前步骤时间轴随轮询实时更新
3. 执行日志逐条追加，新日志自动滚动到底部
4. 分析完成后：全 LLM 成功显示绿色，有降级显示橙色 + 说明文字
5. 正在分析的卡片有蓝色边框 + 「更新中…」角标，完成后绿色闪 2s
6. 用户点「关闭」后按钮恢复默认，Drawer 关闭
7. `pnpm lint:all` 无报错，无新增 hardcoded 字符串
