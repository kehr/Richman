# 事件雷达数据源真实化 实施 Plan

依据：
- PRD: `docs/prds/event-radar-data-source-prd.md`
- TRD: `docs/trds/event-radar-data-source-trd.md`

## 全局规则

- Worktree: `.claude/worktrees/event-radar-data-source/`
- 分支: `event-radar-data-source`
- 每完成一个文件改动立即跑对应 lint，全绿才进入下一个 step
- Commit 主题与 step 一一对应，单 step 内可拆 2-3 个 commit（按 commit-hygiene 单主题原则）
- i18n 双语 keys 必须同 step 完成
- 三端 DTO 改动按 contract-drift checklist 在 Step 5 串行验证

## 派发策略

```
                     ┌─────────────────┐
                     │  Step 0 (准备)  │
                     │  (空跑、确认)   │
                     └────────┬────────┘
                              ↓
       ┌──────────────────────┼──────────────────────┐
       ↓                      ↓                      ↓
  ┌─────────┐           ┌──────────┐           ┌──────────┐
  │ Step 1  │           │ Step 2   │           │ Step 3   │
  │ richson │           │ backend  │           │ frontend │
  │ 全部改动│           │ types    │           │ 全部改动 │
  └────┬────┘           └─────┬────┘           └─────┬────┘
       │                      │                      │
       └──────────────────────┼──────────────────────┘
                              ↓
                    ┌──────────────────┐
                    │ Step 4 (联调验证)│
                    └──────────────────┘
```

- Step 1 / Step 2 / Step 3 完全独立，**必须同时派发并行执行**
- Step 4 等三个 lane 全部完成后串行验证

## Step 列表

| Step | Lane | 目标 | 子文件 |
|------|------|------|--------|
| Step 1 | richson | FRED Releases 接入 + metadata 表 + events.py 重写 + schema 扩展 | `event-radar-data-source-plan/step1-richson.md` |
| Step 2 | backend | EventItem 三个新指针字段 | `event-radar-data-source-plan/step2-backend.md` |
| Step 3 | frontend | 类型扩展 + 组件迁移 + asset-detail 集成 + i18n 双语 | `event-radar-data-source-plan/step3-frontend.md` |
| Step 4 | 全栈 | 联调 + lint + 视觉验证 | `event-radar-data-source-plan/step4-verification.md` |
