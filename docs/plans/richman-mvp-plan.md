# Richman MVP Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Richman MVP -- an AI-driven investment research assistant with portfolio management, three-dimension analysis engine, decision cards, and daily push notifications.

**Architecture:** Monorepo with Go backend (Gin + sqlc + PostgreSQL) and Next.js frontend (Ant Design 6 + TanStack Query). Frontend follows Pages + Features dual architecture borrowed from Orbiter project. Backend follows API handlers -> Service -> Repo three-layer pattern.

**Tech Stack:** Go, Gin, sqlc, PostgreSQL, Next.js 15, Ant Design 6, @ant-design/pro-components, TanStack Query v5, Biome, zap, lumberjack, next-intl

**PRD:** `docs/prds/richman-prd.md`
**Standards:** `docs/standards/` (naming, frontend, backend, database, api, testing, logging)


## Plan Overview

MVP 拆分为 9 个 step，按依赖顺序执行。每个 step 产出可独立验证的工作成果。

```
Step 1: Project Scaffolding
    |
    v
Step 2: Backend Foundation (DB + Auth + Config + Logger)
    |
    v
Step 3: Asset Catalog + Portfolio Management (Backend)
    |
    +---> Step 4: Data Source Integrations (AKShare / Yahoo / Polymarket)
    |         |
    |         v
    |     Step 5: Three-Dimension Analysis Engine
    |         |
    |         v
    +---> Step 6: LLM Integration + Decision Card Generation (depends on Step 3 + Step 5)
              |
              v
          Step 7: Notification System (Push Hub + Cron)
              |
              v
          Step 8: Frontend Shell + All Pages (depends on Step 3 + Step 6 + Step 7)
    |
    v
Step 9: Integration Testing + Polish
```


## Step List

### Step 1: Project Scaffolding
- [step1-project-scaffolding.md](richman-mvp-plan/step1-project-scaffolding.md)
- 目标：搭建 monorepo 骨架，初始化前后端项目，配置工具链
- 依赖：无

### Step 2: Backend Foundation
- [step2-backend-foundation.md](richman-mvp-plan/step2-backend-foundation.md)
- 目标：Go 项目骨架、数据库 schema、auth 系统、配置管理、日志系统
- 依赖：Step 1

### Step 3: Asset Catalog + Portfolio Management
- [step3-portfolio-management.md](richman-mvp-plan/step3-portfolio-management.md)
- 目标：标的目录 API、持仓 CRUD API、交易记录 API、成本计算逻辑
- 依赖：Step 2

### Step 4: Data Source Integrations
- [step4-data-sources.md](richman-mvp-plan/step4-data-sources.md)
- 目标：AKShare / Yahoo Finance / Polymarket API 集成，数据拉取和缓存
- 依赖：Step 2

### Step 5: Three-Dimension Analysis Engine
- [step5-analysis-engine.md](richman-mvp-plan/step5-analysis-engine.md)
- 目标：趋势/位置/催化剂三维量化计算，权重管理，信心度计算
- 依赖：Step 4

### Step 6: LLM Integration + Decision Card Generation
- [step6-llm-decision-card.md](richman-mvp-plan/step6-llm-decision-card.md)
- 目标：多模型 LLM 抽象层，催化剂 LLM 增强，LLM 综合输出决策卡
- 依赖：Step 5

### Step 7: Notification System
- [step7-notification-system.md](richman-mvp-plan/step7-notification-system.md)
- 目标：推送调度器、可插拔渠道适配器（微信/飞书/邮件）、Cron 定时任务
- 依赖：Step 6

### Step 8: Frontend Shell + All Pages
- [step8-frontend-pages.md](richman-mvp-plan/step8-frontend-pages.md)
- 目标：前端完整实现 -- 布局、认证、持仓管理、决策卡展示、推送设置、i18n、主题
- 依赖：Step 3, Step 6, Step 7（后端 API 就绪）

### Step 9: Integration Testing + Polish
- [step9-integration-polish.md](richman-mvp-plan/step9-integration-polish.md)
- 目标：端到端集成测试、前后端联调、风险声明、部署验证
- 依赖：Step 8
