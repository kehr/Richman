# i18n Default English Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking.

**Goal:** 将 Richman 前端从中英混杂状态迁移到 react-i18next 驱动的完整国际化体系，默认英文、可切中文、两处入口切换。

**Architecture:** react-i18next 同步静态加载 4 个 namespace（common / auth / app / settings），ConfigProvider.locale 响应式绑定 i18n.language，格式化工具接受 locale 参数保持纯函数。

**Tech Stack:** i18next, react-i18next, i18next-browser-languagedetector, Ant Design 6 locale packages

## 文档引用

| 文档 | 路径 |
|------|------|
| PRD | `docs/prds/i18n-default-en-prd.md` |
| TRD | `docs/trds/i18n-default-en-trd.md` |

## 全局规则

- 所有 step 在 `feat/i18n-default-en` 分支直接开发（worktree `.claude/worktrees/i18n-default-en`）
- 每个 step 完成后必须通过 `pnpm lint:all`，未通过不进入下一步
- 每个 step 结束产出一个 commit，commit message 遵循 conventional commits
- 代码和注释用英文，JSON 翻译值按对应语言（en 用英文，zh 用中文）
- 迁移中文字符串时，先在 en JSON 加 key，再在 zh JSON 补对应值

## 迁移规模

- 76 个 `.tsx` 文件含 612 处硬编码中文
- 3 个 `.ts` 文件含 14 处硬编码中文
- 3 个 useLocale 消费者需迁移到 useTranslation
- 2 个 format 模块需重构（money/format.ts, ui/format.ts）

## Step 总览

| Step | 目标 | 涉及文件数 | 依赖 |
|------|------|-----------|------|
| 1 | i18n 基础设施（依赖、config、类型、antd-locale） | ~6 新建 | 无 |
| 2 | App.tsx 集成 + 测试工具更新 | ~3 修改 | Step 1 |
| 3 | JSON 资源文件骨架（4 namespace x 2 locale） | 8 新建 | Step 1 |
| 4 | MainLayout i18n（语言 Dropdown + nav 菜单） | 1 修改 | Step 2, 3 |
| 5 | 格式化工具重构（Intl 缓存 + locale 参数） | ~3 修改/新建 | Step 1 |
| 6 | auth namespace 迁移（auth + onboarding 页面） | ~15 修改 | Step 3 |
| 7 | app namespace 迁移 -- dashboard + decision-card | ~25 修改 | Step 3, 5 |
| 8 | app namespace 迁移 -- portfolio | ~15 修改 | Step 3, 5 |
| 9 | settings namespace 迁移 | ~15 修改 | Step 3 |
| 10 | HelpPage 迁移 | ~3 修改 | Step 2 |
| 11 | 测试文件迁移 | ~20 修改 | Step 6-10 |
| 12 | 清理 + 验证 | ~5 删除/修改 | Step 1-11 |

每个 step 的详细内容见 `i18n-default-en-plan/` 目录下同名 step 文件。
