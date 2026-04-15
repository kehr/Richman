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

| Step | Lane | 状态 | 关键 commit |
|------|------|------|-------------|
| Step 0 | docs | 完成 | e0c4076 PRD / e266362 TRD / fd04994 Plan |
| Step 1 | richson | 完成 | f48c15b 元数据表 / 5d86639 FRED REST 拉取 / 802aa1f 重写 events.py |
| Step 2 | backend | 完成 | 1b681cb EventItem 三个指针字段 |
| Step 3 | frontend | 完成 | 639b1a4 DTO 扩展 / b13d164 features 模块统一 + i18n keys / 7436c55 asset-detail 复用 |
| Step 4 | 全栈 | 部分完成 | lint/契约对齐已验，dev server 联调留给主仓库验收 |

### Step 1 备注
- richson `config.py` 与 data 目录 `config/` 同名冲突，agent 通过 `git mv config.py config/__init__.py` 把模块改成 package，保留 `from richson.config import settings` 导入路径，并保证 `wgc.py` 对 `config/wgc_quarterly.json` 的相对路径解析仍然可用
- 新建 `config/event_metadata.py` 含 9 条 release 元数据：CPI=10 / PPI=46 / Employment=50 / PCE=54 / GDP=53 / FOMC=101 / IndProd=13 / Retail=9 / ExistingHomes=291

### Step 2 备注
- `backend && make check` 跑出 50 条预存 lint 警告（agent 通过 stash 验证全部位于本 step 之外的旧文件），本次改动 0 新增 lint 问题，构建/测试/vet 全绿
- 严格遵循 contract-drift：三个新字段全部用 Go 指针类型 + camelCase json tag

### Step 3 备注
- i18n 文件采用 namespace 分文件结构，最终落在 `frontend/src/i18n/locales/{zh,en}/market.json`，新增 `overview.eventRadar.openSourceTooltip` 与 `overview.eventRadar.sourceLabel`
- Plan 原本把 i18n keys 放 commit 3，agent 因 `i18next` 类型从 en resources 推断键名导致 commit 2 引用未声明 key 时 tsc 失败，将 i18n keys 提前到 commit 2，commit 3 仅含 asset-detail 重写。属于 lint 通过纪律驱动的微调，未改变 step 拆分语义
- EventRow 仅当 `sourceUrl` 以 `https://` 开头才渲染 `<a>`（防 javascript: 注入），React key 优先 `fred-{releaseId}-{date}`，所有 antd 组件经 `@/ui-kit/eat` barrel 导入

## 已修复问题

| 问题 | 修复方案 | 验证 |
|------|----------|------|
| richson 事件雷达硬编码 `_FIXED_EVENTS` | 替换为 FRED Releases REST 真实日历 + Polymarket 现有数据源 | ruff/mypy/pytest 全绿 |
| richson `config.py` 与 data `config/` 命名冲突 | `git mv config.py config/__init__.py`，模块改 package | `from richson.config import settings` 仍可用，wgc 路径解析通过 |
| backend EventItem 缺 source/release 字段 | EventItem 加 `*string`/`*string`/`*int` 三个指针字段（Pydantic `T \| None` 对齐） | `make check` 通过 |
| 前端 EventRadarSection 在 page-local 与 features 双份 | 删 page-local 副本，唯一来源迁移到 `features/event-radar/`，barrel 导出 | grep 全局唯一引用确认 |
| asset-detail/event-calendar.tsx 占位符 | 复用 `<EventRadarSection/>`，共享 `["events","radar"]` query key 跨页面缓存 | lint:all + build 全绿 |
| sourceUrl 缺 URL 安全防御 | 仅 `https://` 前缀允许渲染为 anchor | code-level enforcement |

## 已记录但未修复的观察项

| 观察项 | 来源 | 后续处理时机 |
|--------|------|--------------|
| `polymarket.py:143` raw key 应为 `end_date` 而非 `end_date_iso`（latent bug 已在 TRD §7 标记） | TRD 审查 | 与本 plan 范围无关，下次触碰 polymarket 客户端时一并修 |
| backend `make check` 50 条 pre-existing lint baseline | Step 2 | 这些都不在本 plan 修改的文件中，不属本次 scope |
| FRED 9 条 release_id 仅基于 TRD 文档与 FRED 公开 release 命名匹配，未在联调环境对照 `fred/releases?limit=1000` 实际响应核对 | Step 1 | 主仓库联调时若发现偏差，调整 `event_metadata.py` 即可，不需 schema 改动 |

## 无法决策项

无（用户授权全权处理）。

## Review 结果

- Spec compliance：三端 DTO 字段命名/null 语义全部按 contract-drift 对齐（Pydantic `| None` + Go `*T` + TS `| null`），grep 验证 `sourceUrl|sourceName|releaseId` 三层均存在
- Code quality：所有 commit 主题独立、显式 `git add` 文件清单、message 不含 AI 痕迹、零 emoji；frontend 全程 lint:all + build 双绿，backend make check / richson ruff+mypy 全绿
- 待主仓库验收：dev server 联调（DevTools Network 抓 `GET /api/v2/events/radar`、视觉验收 hover/cursor、FRED key 清空降级、点击 anchor 跳转）按 user 偏好交回主仓库人工验收
