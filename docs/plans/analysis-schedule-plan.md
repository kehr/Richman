# 自动分析调度策略 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为用户提供三层嵌套的自动分析调度配置（全局 → 市场 → 持仓），支持时间窗口开关、自定义时间和多级频率选项，替换后端硬编码调度逻辑。

**Architecture:** 后端新增两张配置表（user_schedule_settings / holding_schedule_overrides），调度器改为从数据库动态加载；前端新增 ScheduleTab 和持仓覆盖控件，下次分析时间改由后端计算返回。

**Tech Stack:** Go + Gin + sqlc + robfig/cron/v3（后端）；React 19 + TanStack Query v5 + Ant Design 6 + lucide-react（前端）

**设计依据：**
- PRD: `docs/prds/analysis-schedule-prd.md`
- TRD: `docs/trds/analysis-schedule-trd.md`

**Step 文件：**

| Step | 文件 | 依赖 | 可并行 |
|------|------|------|-------|
| 1 | `step1-db-migration.md` | 无 | 否 |
| 2 | `step2-sqlc-queries.md` | Step 1 | 否 |
| 3 | `step3-schedule-service.md` | Step 2 | 否 |
| 4 | `step4-dst-scheduler.md` | Step 3 | 否 |
| 5 | `step5-api-handlers.md` | Step 3 | 与 Step 6/7/8 并行 |
| 6 | `step6-premarket-delta.md` | Step 3 | 与 Step 5/7/8 并行 |
| 7 | `step7-frontend-i18n.md` | 无 | 与 Step 5/6/8 并行 |
| 8 | `step8-frontend-hooks.md` | 无 | 与 Step 5/6/7 并行 |
| 9 | `step9-frontend-schedule-tab.md` | Step 7/8 | 与 Step 10 并行 |
| 10 | `step10-frontend-holding-section.md` | Step 7/8 | 与 Step 9 并行 |
| 11 | `step11-integration.md` | Step 4/5/9/10 | 否 |

**并行执行建议：**
- 第一轮：Step 1 → 2 → 3 → 4（串行后端基础）
- 第二轮：Step 5 + 6 + 7 + 8 同时派发
- 第三轮：Step 9 + 10 同时派发
- 第四轮：Step 11（集成收尾）
