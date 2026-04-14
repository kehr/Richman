# 事件雷达数据源彻底修复 执行报告

## 执行方式

- Worktree: `.claude/worktrees/event-radar-data-source/`
- 分支: `event-radar-data-source`（基于 origin/main）
- 目标: 用 FRED Releases Calendar 真实日历替换 richson 内的 _FIXED_EVENTS 硬编码列表，三端 DTO 增加 sourceUrl/sourceName，前端事件雷达条目可点击新标签打开权威源；顺带把资产详情页 event-calendar.tsx 占位符改为复用同一组件
- 派发策略: PRD/TRD/Plan 写完后用 subagent-driven-development，richson 与 frontend 改动可并行，backend 透传等 richson schema 完成后串行

## 全局规则

- 零 AI 痕迹（commit/message/branch/code 不含 AI/Claude 关键词）
- 三端 DTO 命名同步（snake_case ↔ json tag ↔ camelCase）
- 跨层 DTO 改动必须同步 richson schema + backend types + frontend types（contract-drift）
- 文档中文，代码英文，禁 emoji
- i18n 中英双 locale 同步
- 每次文件改动后立即跑对应 lint，全部通过才进入下一步
- 用户偏好：worktree 完成后直接 rebase → ff-merge → push，无需逐步确认

## 决策记录（brainstorming 阶段确认）

| 决策 | 选择 | 影响 |
|------|------|------|
| 数据源 | FRED Releases Calendar + Polymarket | richson 接 fred/releases/dates，移除 _FIXED_EVENTS |
| 点击行为 | 新标签打开 source_url | 前端整行 anchor，无 Drawer/详情页 |
| 范围 | 同时实现 asset-detail/event-calendar.tsx | 复用同一份 hook + 组件 |
| FRED ↔ FIXED 关系 | FRED 完全替换 | FRED disabled 时退回 Polymarket-only，不再保留硬编码 |
| asset-detail 过滤 | 不过滤、复用同一份 | DTO 不加 relevant_asset_types |
| impact / goldDirection 元数据 | richson 内置静态 release_id → metadata 表 | 新建 config/event_metadata.py |
| 时间窗口 | 块定 7 天 | richson 内部硬编码，无 query param |

## Step 执行状态

待 PRD/TRD/Plan 完成后填写。

## 已修复问题

待执行后追加。

## 已记录但未修复的观察项

待执行后追加。

## 无法决策项

无（用户授权全权处理）。

## Review 结果

待 review 后填写。
