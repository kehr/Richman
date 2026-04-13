# Richman v2 实施计划

> **For agentic workers:** 使用 superpowers:subagent-driven-development 执行此计划。每个 step 分配独立 subagent + worktree，按依赖拓扑和并行组调度。Steps 使用 checkbox 语法追踪进度。

**Goal:** 实现 Richman v2 全部功能 -- richson 量化服务、Market Overview 首页、标的详情页、投研简报重构、邀请系统、邮件推送、风险偏好体系

**Architecture:** richson (Python/FastAPI) 作为量化 + LLM 编排侧车，与 richman (Go/Gin) 通过 HTTP REST 通信，共享 PostgreSQL（rm_* richman 表 + rs_* richson 表）。前端 React SPA 消费 richman v2 API，Market Overview 和标的详情页公开访问，执行计划需持仓登录。

**Tech Stack:** Go (Gin + pgxpool), Python (FastAPI + Google ADK + SQLAlchemy 2.0 + asyncpg), React 19 (Vite 6 + antd ^5.24 + TanStack Query v5), PostgreSQL

## 输入文档

| 文档 | 路径 | 用途 |
|------|------|------|
| PRD v2 | docs/prds/richman-prd-v2.md | 产品需求 |
| richson TRD | docs/trds/richson-service-trd.md | richson 服务架构 |
| richman 后端 TRD | docs/trds/richman-backend-v2-trd.md | richman 后端变更 |
| 前端 TRD | docs/trds/frontend-v2-trd.md | 前端重构 |
| 邀请系统 TRD | docs/trds/invite-system-trd.md | 邀请裂变 |
| 变更点清单 | docs/trds/v2-change-inventory.md | 303 项覆盖基线 |

## 依赖拓扑与并行组

```
Phase 1 (Foundation)          Phase 2 (richson)       Phase 3 (richman backend)              Phase 4 (Frontend)           Phase 5
===========================    ====================    ====================================    ==========================    ========
[Step 1: DB Migrations    ]    [Step 4: Data+Quant]    [Step 7: richson Client]                [Step 14: FE Foundation ]    [Step 19]
[Step 2: richson Scaffold ] -> [Step 5: ADK Agents] -> [Step 8: Core Services  ]               [Step 15: Market Ovrvw  ]    Deploy
[Step 3: Model/Repo       ]    parallel: 4,5           [Step 9: Invite System  ] parallel      [Step 16: Asset Detail  ]
parallel: 1,2,3                       |                [Step 10: Email Push    ] 8,9,10        [Step 17: Briefing+Hold ]
                               [Step 6: API+Pipeline]         |                               [Step 18: Settings+Auth ]
                               depends: 4,5            [Step 11: v2 Handlers   ]               parallel: 15,16,17,18
                                                       [Step 12: Cron Tasks    ] parallel               |
                                                       parallel: 11,12                         [Step 19: Deploy+Integ  ]
                                                              |
                                                       [Step 13: Config+Startup]
                                                       depends: 11,12
```

## 并行组调度表

| 调度轮次 | 并行组 | Steps | 前置条件 |
|----------|--------|-------|----------|
| R1 | Phase 1 | 1, 2, 3 | 无 |
| R2 | richson Core | 4, 5 | Step 2 完成 |
| R3 | richson API | 6 | Steps 4, 5 完成 |
| R4 | richman Client | 7 | Step 6 完成 + Steps 1, 3 完成 |
| R5 | richman Services | 8, 9, 10 | Step 7 完成 |
| R6 | richman Handlers+Cron | 11, 12 | Steps 8, 9, 10 完成 |
| R7 | richman Config | 13 | Steps 11, 12 完成 |
| R8 | Frontend Foundation | 14 | Steps 1, 3 完成（API 契约稳定即可开始） |
| R9 | Frontend Pages | 15, 16, 17, 18 | Step 14 完成 |
| R10 | Deployment | 19 | 全部完成 |

注：R8 (前端 Foundation) 可与 R4-R7 重叠执行 -- 前端只依赖 API 契约（TRD 已定义），不依赖后端实现完成。实际调度时，R1 完成后可同时启动 R2 和 R8。

## Step 总览

| # | Step | Phase | 涉及服务 | 清单条目 | Step 文件 |
|---|------|-------|----------|----------|-----------|
| 1 | richman DB Migrations + Seed | 1 | richman | A1-A4 (38) + G2.12-G2.13 | step1-richman-db-migrations.md |
| 2 | richson Scaffold + DB Layer | 1 | richson | C1, B1-B6, C6 (15) | step2-richson-scaffold.md |
| 3 | richman Model + Repo Layer | 1 | richman | D5-D7 (25) | step3-richman-model-repo.md |
| 4 | richson Data Sources + Quant Engine | 2 | richson | C3.2-C3.14, C5 (21) | step4-richson-data-quant.md |
| 5 | richson ADK Agents + Degradation | 2 | richson | C4, C7 (8) | step5-richson-adk-agents.md |
| 6 | richson Pipeline + API + Middleware + CLI | 2 | richson | C2, C3.1, C8, C9, G1 (29) | step6-richson-api-pipeline.md |
| 7 | richman richson HTTP Client | 3 | richman | D1 (16) | step7-richman-richson-client.md |
| 8 | richman v2 Core Services | 3 | richman | D3.1-D3.5, D3.10-D3.11, D3.18, D4.1-D4.3 (11) | step8-richman-core-services.md |
| 9 | richman Invite System | 3 | richman | D3.12-D3.17, D4.4-D4.6, G2.8, G2.10, G4 (14) | step9-richman-invite-system.md |
| 10 | richman Email Push System | 3 | richman | D3.6-D3.9, D9 (14) | step10-richman-email-push.md |
| 11 | richman v2 Handlers + Routing | 3 | richman | D2, D11, G2.15 (29) | step11-richman-v2-handlers.md |
| 12 | richman Cron Tasks | 3 | richman | D8, G2.14 (13) | step12-richman-cron-tasks.md |
| 13 | richman Config + Startup + Deprecation | 3 | richman | D10, D12, G2 (remaining) (21) | step13-richman-config-startup.md |
| 14 | Frontend Foundation | 4 | frontend | E9, E10, E13, G3.1-G3.2, G3.7-G3.8 (19) | step14-frontend-foundation.md |
| 15 | Market Overview Page | 4 | frontend | E3.1, E1.1, E1.3, E4, E11.1, E11.8, E12.1-E12.2, E12.4, G3.5, G3.9 (16) | step15-market-overview-page.md |
| 16 | Asset Detail Page | 4 | frontend | E3.2, E1.2, E2.1-E2.2, E5, E11.2, E12.3, G3.3, G3.6 (27) | step16-asset-detail-page.md |
| 17 | Briefing + Holdings Pages | 4 | frontend | E3.3, E1.4, E1.5, E6, E7, E11.3, E11.9, E12.5-E12.6 (17) | step17-briefing-holdings-page.md |
| 18 | Settings + Invite + Auth | 4 | frontend | E3.4, E1.6, E2.3-E2.5, E8, E11.4-E11.7, G3.4, G3.10, G4 (17) | step18-settings-invite-auth.md |
| 19 | Deployment + Integration | 5 | all | F1-F5 (5) | step19-deployment-integration.md |

**总计: 303 项变更点，19 个 step**

## 覆盖率校验矩阵

下表确保 v2-change-inventory.md 中每个条目都映射到至少一个 step。

### A. DB richman migrations (38) -> Step 1

A1.1-A1.15, A2.1-A2.17, A3.1-A3.5, A4.1

### B. DB richson Alembic (6) -> Step 2

B1-B6

### C. richson 服务 (49) -> Steps 2, 4, 5, 6

| 子组 | 条目 | Step |
|------|------|------|
| C1 scaffold | C1.1-C1.6 | 2 |
| C2 API | C2.1-C2.11 | 6 |
| C3 quant | C3.1 (pipeline) | 6 |
| C3 quant | C3.2-C3.14 | 4 |
| C4 ADK | C4.1-C4.5 | 5 |
| C5 datasources | C5.1-C5.8 | 4 |
| C6 DB/schema | C6.1-C6.3 | 2 |
| C7 degradation | C7.1-C7.3 | 5 |
| C8 CLI | C8.1-C8.3 | 6 |
| C9 observability | C9.1-C9.5 | 6 |

### D. richman 后端 (96) -> Steps 3, 7-13

| 子组 | 条目 | Step |
|------|------|------|
| D1 richson client | D1.1-D1.16 | 7 |
| D2 v2 handlers | D2.1-D2.20 | 11 |
| D3 v2 services | D3.1-D3.5 | 8 |
| D3 v2 services | D3.6-D3.9 | 10 |
| D3 v2 services | D3.10-D3.11 | 8 |
| D3 v2 services | D3.12-D3.17 | 9 |
| D3 v2 services | D3.18 | 8 |
| D4 existing svc | D4.1-D4.3 | 8 |
| D4 existing svc | D4.4-D4.6 | 9 |
| D5 v2 repos | D5.1-D5.11 | 3 |
| D6 existing repos | D6.1-D6.6 | 3 |
| D7 models | D7.1-D7.8 | 3 |
| D8 cron | D8.1-D8.12 | 12 |
| D9 email templates | D9.1-D9.10 | 10 |
| D10 config | D10.1-D10.8 | 13 |
| D11 errors | D11.1-D11.8 | 11 |
| D12 deprecation | D12.1-D12.4 | 13 |

### E. 前端 (73) -> Steps 14-18

| 子组 | 条目 | Step |
|------|------|------|
| E1 new features | E1.1 | 15 |
| E1 new features | E1.2 | 16 |
| E1 new features | E1.3 | 15 |
| E1 new features | E1.4 | 17 |
| E1 new features | E1.5 | 17 |
| E1 new features | E1.6 | 18 |
| E2 existing features | E2.1-E2.2 | 16 |
| E2 existing features | E2.3-E2.5 | 18 |
| E3 pages | E3.1 | 15 |
| E3 pages | E3.2 | 16 |
| E3 pages | E3.3 | 17 |
| E3 pages | E3.4 | 18 |
| E4 market overview components | E4.1-E4.6 | 15 |
| E5 asset detail components | E5.1-E5.19 | 16 |
| E6 briefing components | E6.1-E6.4 | 17 |
| E7 holdings components | E7.1-E7.6 | 17 |
| E8 settings components | E8.1-E8.3 | 18 |
| E9 routes | E9.1-E9.9 | 14 |
| E10 API client | E10.1-E10.3 | 14 |
| E11 i18n | E11.1 | 15 |
| E11 i18n | E11.2 | 16 |
| E11 i18n | E11.3 | 17 |
| E11 i18n | E11.4-E11.7 | 18 |
| E11 i18n | E11.8 | 15 |
| E11 i18n | E11.9 | 17 |
| E12 SEO + deps | E12.1-E12.2 | 15 |
| E12 SEO + deps | E12.3 | 16 |
| E12 SEO + deps | E12.4 | 15 |
| E12 SEO + deps | E12.5-E12.6 | 17 |
| E13 cleanup | E13.1-E13.3 | 14 |

### F. 部署 (5) -> Step 19

F1-F5

### G. 已知问题 (37) -> 分散到相关 step

| 子组 | 条目 | Step |
|------|------|------|
| G1 richson | G1.1-G1.9 | 6 |
| G2 richman | G2.1-G2.7, G2.9, G2.11 | 13 |
| G2 richman | G2.8, G2.10 | 9 |
| G2 richman | G2.12-G2.13 | 1 (附带处理) |
| G2 richman | G2.14 | 12 |
| G2 richman | G2.15 | 11 |
| G3 frontend | G3.1-G3.2, G3.7-G3.8 | 14 |
| G3 frontend | G3.3, G3.6 | 16 |
| G3 frontend | G3.4, G3.10 | 18 |
| G3 frontend | G3.5, G3.9 | 15 |
| G4 invite | G4.1-G4.3 | 9 |

## 执行策略

### Worktree 隔离

每个 step 在独立 worktree 中执行。worktree 路径：`.claude/worktrees/<step-short-name>/`

### 并行组合入顺序

同一并行组内的 steps 各自完成后，按编号顺序 rebase -> ff-merge -> push。后续并行组从更新后的 main 开始。

### 跨 worktree 撞名预防

Step 1 (migrations) 和 Step 2 (alembic) 创建不同目录下的文件，无冲突。Step 3 (model/repo) 与 Step 1 无文件重叠（repo 只改 .go 文件）。前端并行组 R9 内各 step 操作不同 page/feature 目录，但 Step 15 和 Step 17 均修改 `frontend/src/i18n/locales/{zh,en}/common.json`（不同 key），合入时需按编号顺序 rebase 解决 JSON 合并冲突。

### 验收流程

每个 step 完成后在 worktree 内跑对应的 lint/test。合入 main 后在主仓库跑全量验证（`make check` / `pnpm lint:all`）。
