# Step 8: Frontend API Types + Hooks

**依赖：** 无（独立，基于 TRD 合约）
**可与 Step 5/6/7 并行**
**设计依据：** TRD §前端 API Hooks、§API 设计

## 任务目标

新建 `features/schedule` 模块，包含 DTO 类型定义、API 函数和 TanStack Query hooks。

## 涉及文件

- 创建：`frontend/src/features/schedule/api.ts`
- 创建：`frontend/src/features/schedule/useSchedule.ts`
- 创建：`frontend/src/features/schedule/index.ts`（barrel）

## 执行步骤

- [ ] 查看 `features/settings-llm/api.ts` 和 `features/settings-llm/useSettingsLlm.ts` 作为同类 feature 的参考，理解 DTO 定义和 hook 模式
- [ ] 创建 `api.ts`，按 TRD §前端 API Hooks 定义：
  - `WindowDTO`、`MarketScheduleDTO`、`ScheduleSettingsDTO` 接口（对应后端 GET /settings/schedule 响应结构）
  - `HoldingScheduleDTO` 接口（对应 GET /holdings/:id/schedule 响应结构）
  - `fetchScheduleSettings()`、`updateScheduleSettings(data)`
  - `fetchHoldingSchedule(holdingId)`、`updateHoldingSchedule(holdingId, data)`
  - 所有函数通过 `request()` from `@/domain/http`
- [ ] 创建 `useSchedule.ts`，按 TRD 定义四个 hooks：
  - `useScheduleSettings()` — queryKey `["schedule-settings"]`
  - `useUpdateScheduleSettings()` — 成功后 invalidate `["schedule-settings"]`
  - `useHoldingSchedule(holdingId)` — queryKey `["holding-schedule", holdingId]`
  - `useUpdateHoldingSchedule()` — 成功后 invalidate `["holding-schedule", holdingId]`
- [ ] 创建 `index.ts` barrel，导出全部类型和 hooks
- [ ] 执行 `cd frontend && pnpm lint:all` 验证通过
- [ ] `git add frontend/src/features/schedule/ && git commit -m "feat(schedule): add schedule feature API and hooks"`

## 验证标准

- `pnpm lint:all` 通过（含 tsc 类型检查 + dependency-cruiser）
- DTO 字段名与 TRD §API 设计中的 JSON 字段名完全一致（camelCase）
- dependency-cruiser 无跨 feature 依赖违规
