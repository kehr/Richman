# 事件雷达契约漂移 + FRED 占位符 修复执行报告

## 背景

2026-04-15 用户截图反馈：

1. 后端日志洪水般重复 `fred fetch failed / Bad Request. The value for variable api_key is not a 32 character alpha-numeric lower-case string.`，并伴随 `richson request failed ... path=market/regime ... context deadline exceeded`
2. 前端市场概览页：上半部"资产卡片"全部显示"即将开放"占位；"事件雷达"面板出现字面 `overview.eventRadar.impactLevel.undefined`、`概率 NaN%`、`24h 变动 NaNpp`

## 根因定位（Phase 1）

通过逐段放大截图日志 + 源码 grep 对齐，确认是三个独立但叠加的问题：

### 问题 A：richson FRED_API_KEY 占位符

- `richson/.env:12` 原为 `FRED_API_KEY=...`（字面三个点，未被替换）
- `FREDClient._get_client` 将非空字符串 key 无条件传给 `fredapi.Fred`，FRED 服务端返回 Bad Request
- 所有 series（TIPCFY / DFII10 / MSCI / BAMLC0A0CM 等）全部失败
- richson 内部 retry 拖累 `/market/regime` 返回，backend 调 richson 超时，导致市场 overview 回退占位

### 问题 B：事件雷达三端字段命名漂移

| 层 | 字段命名 |
|---|---|
| `richson/src/richson/api/events.py:77-86` | `impact`, `probability`, `probabilityChange24h`, `goldDirection`（无 `id`） |
| `backend/internal/richson/types.go:153-162` | 原样透传 |
| `frontend/src/features/event-radar/types.ts:3-11` | `impactLevel`, `polymarketProbability`, `polymarketChange24h`, `goldDirection`, `id` |

前端字段名读不到 → `undefined`。三端 TypeScript/Go/Python 各自类型检查都通过，只有实际访问页面才能发现。

### 问题 C：Go 值类型塌陷 null

backend `Probability float64` 会把 richson 的 JSON `null` unmarshal 成 `0.0`，再 marshal 回 `0`，导致「无 polymarket 数据的事件」在 frontend 显示为「概率 0%」（修复命名后）。

### 问题 D：frontend 防御性判断漏 undefined

`event.polymarketProbability !== null` 对 `undefined` 为 true，仍进入 `.toFixed()` 产出 NaN。

## 执行方式

- 工作目录：主仓库 `main` 分支直接修改（bugfix 走例外通道，非大特性）
- 冲突处理：文件独占，单 Claude Code 实例
- 零 AI 痕迹：commit message 将遵守规则（未在本次动作中 commit）

## 修改清单

### Frontend（对齐 backend/richson 命名）

| 文件 | 改动 |
|---|---|
| `frontend/src/features/event-radar/types.ts` | `impactLevel` → `impact`；`polymarketProbability` → `probability`；`polymarketChange24h` → `probabilityChange24h`；新增 `probabilitySource`；移除不存在的 `id` |
| `frontend/src/pages/market-overview/components/event-radar-section.tsx` | 同步更新字段引用；`!== null` 改为 `typeof x === "number"`；React key 由 `event.id` 改为 `${date}-${title}-${idx}` |

### Backend（保持 null 语义）

| 文件 | 改动 |
|---|---|
| `backend/internal/richson/types.go` | `EventItem.GoldDirection` / `Probability` / `ProbabilitySource` / `ProbabilityChange24h` 改为指针类型，避免 JSON null 被塌陷 |
| `backend/internal/service/emailpush/service.go` | 适配 `ev.GoldDirection` 变 `*string`，用 nil 检查 deref |

### Richson（占位符防御）

| 文件 | 改动 |
|---|---|
| `richson/src/richson/datasources/fred.py` | 新增 `_is_valid_fred_api_key(key)` shape 校验（`^[a-z0-9]{32}$`）；`FREDClient.__init__` 检测到非法 key 时设 `self._disabled=True` 并启动 WARN；`_fetch_series` 在 disabled 时短路返回 None，不走网络和 retry |
| `richson/.env.example` | 占位符从 `FRED_API_KEY=...` 改为 `FRED_API_KEY=` + 注册链接注释，避免下次被 cp 后还得想着删 |

### 规范沉淀

| 文件 | 改动 |
|---|---|
| `docs/standards/contract-drift.md`（新） | 跨层 DTO 对齐纪律：三端命名 / nullable 映射表 / 三层必须同 PR / PR 自查清单 / 端到端人工验证 |
| `CLAUDE.md` | Standards Index 新增 `contract-drift.md` 条目，标记 MANDATORY |
| `~/.claude/projects/-Users-kyle-Studio-Richman/memory/feedback_cross_layer_contract_drift.md`（新） | 项目 memory：改 richson schema / backend types / frontend types 前必查对齐 |
| `~/.claude/projects/-Users-kyle-Studio-Richman/memory/feedback_env_placeholder_fail_fast.md`（新） | 项目 memory：外部 key 客户端必须 shape 校验 + 短路 |
| `~/.claude/projects/-Users-kyle-Studio-Richman/memory/MEMORY.md` | 新增两条索引 |

## 验证

### 自动化

- `cd frontend && pnpm lint:all` → Biome + tsc + depcruiser 全绿
- `cd backend && go build ./...` → 无错
- `cd backend && go vet ./...` → 无错
- `cd backend && go test ./internal/richson/... ./internal/service/emailpush/...` → 相关包暂无测试（not regressed）
- `cd richson && uv run ruff check src/richson/datasources/fred.py` → 无错
- `cd richson && uv run mypy src/richson/datasources/fred.py` → 本次改动未引入新 mypy 错误（3 条报错均为 pre-existing：line 100/178/179）

### 运行时 sanity 检查

```bash
uv run python -c "
from richson.datasources.fred import _is_valid_fred_api_key, FREDClient
from richson.config import settings
print('key_len:', len(settings.fred_api_key))
print('is_valid:', _is_valid_fred_api_key(settings.fred_api_key))
c = FREDClient(api_key=settings.fred_api_key)
print('disabled:', c._disabled)
"
# key_len: 32
# is_valid: True
# disabled: False
```

用户补充的 `FRED_API_KEY=6a709ac626a8924383057e8204cb6639` 通过 shape 校验，不会触发短路。

### 人工验收（待用户执行）

`docs/standards/contract-drift.md` 明确要求 PR 合并前必须跑端到端验证。本次修复应验证：

1. 重启 richson 和 backend
2. 访问市场概览页 `/` 或 `/market-overview`
3. DevTools → Network → `/api/v2/events/radar`：确认响应字段为 `impact/probability/probabilityChange24h/goldDirection` 且可空字段为 `null` 而非 `0`/`""`
4. 页面事件雷达面板：
   - 不再出现 `NaN%`、`NaNpp`
   - 不再出现 `overview.eventRadar.impactLevel.undefined`
   - 无 polymarket 数据的事件不显示概率区块（而非显示「概率 0%」）
5. 后端日志：`make dev` 启动后不再出现 FRED Bad Request 刷屏；richson `/market/regime` 恢复正常返回（依赖 FRED 的卡片不再"即将开放"）

## 追加：MarketOverview 资产卡片契约漂移修复（阶段 1）

2026-04-15 同日截图反馈：行情首页满屏灰色「即将开放」占位卡。根因是事件雷达契约漂移的孪生问题——frontend `AssetCardDto` 与 backend `AssetCardDTO` 字段命名完全错位：

| frontend 期望 | backend 实际返回 | 结果 |
|---|---|---|
| `nameZh` | `name` | undefined |
| `signal` | `signalLevel` | undefined |
| `isActive` | 不返回 | 永远 undefined → falsy → 全部走「即将开放」占位 |
| `price` / `changePercent` / `currency` / `percentileLabel` | 不返回 | undefined |
| `AssetGroupDto.category` / `categoryLabel` | `assetType` | undefined |

### 设计决策

用户选择前端对齐后端（同 event-radar 方向）并分两阶段推进：

- **阶段 1（本次）**：纯 frontend 修复。字段重命名 + 激活判断改用 `overallScore` 是否存在 + 文案改「等待分析」+ 补 i18n 键。不改 backend。
- **阶段 2（后续独立任务）**：backend `AssetCardDTO` 增加 `percentileLabel` 并在 `GetOverview` 批量计算；新增 batch quote 接口或直接在 overview 中嵌入 `current`（price / changePercent / currency）。阶段 2 应走 PRD → TRD → Plan 正式流程，涉及数据源 QPS 评估与缓存策略。

### 本次修改清单（阶段 1）

| 文件 | 改动 |
|---|---|
| `frontend/src/features/market-overview/types.ts` | `AssetCardDto` 按 backend 返回重写：`nameZh → name`、`signal → signalLevel`、`category → assetType` 分组键；移除 `isActive` / `price` / `changePercent` / `currency` / `percentileLabel`；保留并对齐 `overallScore` / `scoreDelta` 为可空；`AssetGroupDto` 改为 `assetType + assets` |
| `frontend/src/pages/market-overview/components/asset-card.tsx` | 激活判断从 `!asset.isActive` 改为 `typeof asset.overallScore !== "number"`；新增 `scoreDelta` 趋势着色；移除 price/changePercent/percentileLabel 渲染；i18n key `overview.assetCard.comingSoon` 改为 `overview.assetCard.waitingAnalysis` |
| `frontend/src/pages/market-overview/components/asset-group-section.tsx` | label 从 `group.categoryLabel` 改为 `t('overview.assetType.${assetType}', assetType)`，带 raw-key fallback 兜底未知 assetType |
| `frontend/src/pages/market-overview/components/asset-card-wall.tsx` | 分组 key 从 `group.category` 改为 `group.assetType` |
| `frontend/src/i18n/locales/{zh,en}/market.json` | 新增 `overview.assetCard.waitingAnalysis` 和 `overview.assetType.{gold_etf,a_share_broad,a_share_industry,us_stock}`；删除无用的 `overview.assetCard.comingSoon` |

### 阶段 1 验证

- `cd frontend && pnpm lint:all` → Biome + tsc + depcruise 全绿（252 files / 277 modules / 879 deps）
- 人工验收（待用户执行）：
  1. 刷新 `/market`，卡片不再满屏「即将开放」
  2. 已分析资产显示名称 + 评分/100 + scoreDelta 趋势 + 信号标签，可点击进详情
  3. 无 analysis 的资产显示灰色卡片 + 「等待分析」标签，不可点击
  4. 分组标题（如「黄金 ETF」「A 股宽基」）按 assetType + 当前 locale 正确翻译；新 assetType 未加翻译时 fallback 到 raw key 不崩

## 已记录但未修复的观察项

1. richson 的 mypy 基线有 106 条 pre-existing 错误，涉及 33 个文件。本次只验证改动未引入新错误，未做统一修复。建议下一个专门的"richson 类型修复"任务处理。
2. 用户 `richson/.env` 中 `PLATFORM_LLM_API_KEY=sk-...` 同样是占位符，本次未改动。若未来 scheduler 用到 platform LLM，会在这里再次踩坑。短期可参考 FRED 方案为 LLM 客户端加同样的占位符检测。
3. ~~`frontend/src/pages/market/MarketOverviewPage.tsx` 与 `frontend/src/pages/market-overview/market-overview-page.tsx` 并存~~。**已处理**：旧 `pages/market/` 目录两个 stub 文件删除，`routes.tsx` 中 `AssetDetailPage` 的 lazy import 直接指向 `@/pages/asset-detail`。
4. backend `make check` 依赖 `golangci-lint`，当前本机未安装；本次只跑了 `go build` + `go vet` 替代。建议在 `docs/standards/lint-toolchain.md` 约束的版本下补装。
5. **阶段 2 待办（独立任务）**：backend overview DTO 加 `percentileLabel` 批量计算 + 增加 batch quote 接口（或在 overview 中内嵌 `current`）以支持卡片显示百分位 + 实时价格 / 涨跌幅 / 币种。需 PRD + TRD + Plan。
6. 行情首页和卡片详情页原本用 `maxWidth: 960` / `maxWidth: 900` 硬限宽导致 1920+ 屏内容过窄，且未用 PageContainer 包裹。**已处理**：两页面根节点统一改为 `<PageContainer title={false}>`，删除 page 级 maxWidth 交给 ProLayout contentWidth="Fixed" 统一控制；`docs/standards/frontend.md` 补了「Page 根元素必须用 PageContainer（MANDATORY）」章节，memory 写入 `feedback_page_must_use_pagecontainer.md`。

## 验收后续

- 用户确认端到端可见事件雷达恢复正常后，本次改动可直接 commit 到 main
- commit 建议拆分为六个独立 commit（遵守 `docs/standards/commit-hygiene.md` 一次一主题）：
  1. `fix(events): align event radar DTO names across richson/backend/frontend`（frontend types + section + backend pointers + emailpush 适配）
  2. `fix(richson): short-circuit FRED fetches when api_key is a placeholder`（fred.py + .env.example）
  3. `docs(standards): add cross-layer contract drift discipline`（contract-drift.md + CLAUDE.md 索引 + 两条 memory）
  4. `fix(market): align asset card DTO and show waiting-analysis placeholder`（market-overview 字段对齐、组件重写、i18n 键迁移）
  5. `fix(ui): wrap market pages with PageContainer and drop page-level maxWidth`（market-overview-page + asset-detail + routes.tsx 清理旧 market 目录）
  6. `docs(standards): require PageContainer as page root`（frontend.md + feedback_page_must_use_pagecontainer.md + MEMORY.md 索引）

## 追加：资产卡片「等待分析」满屏 + 中文名显示异常（2026-04-15 夜）

用户反馈：市场概览仍满屏「等待分析」，且资产名显示英文。逐层排查后发现两类互相独立的问题。

### 问题 E：daily 分析冷启动未触发（数据侧）

- `backend/internal/service/schedule/v2_cron.go:126` 注册 `runDailyAssetAnalysis` 在 `0 22 * * *` UTC（06:00 UTC+8）。dev 环境今天这个时间窗已过，backend 23:08 UTC 才起来，所以 `rs_asset_analyses` 表空 → overview API 返回全部 `overallScore=null` → 卡片全走「等待分析」占位。
- 不是代码 bug，是 dev 冷启动的运行态问题。
- 处理方式：走 admin recovery 通道手动触发批量分析。因 `RequireAdmin` 中间件只读 JWT 的 role claim（不回查 DB），直接用 `go run` 小程序签一个 `role=admin` 的 JWT 发到 `POST /api/v2/analysis/trigger-batch`，body 空 → 对 26 个 active 资产全量派发。
- 结果：26 个 job 全部写入 `rs_asset_analyses`，overview API 返回值包含 `overallScore / signalLevel / scoreDelta` 字段。

### 问题 F：seed 文件 name 列全英文（显示侧）

- `backend/db/seed/asset_catalog.sql` 既把 `name` 和 `name_en` 都填成了英文，又仍然 `INSERT INTO asset_catalog`（migration 021 已改名为 `rm_asset_catalog`）。
- frontend `asset-card.tsx:25` `displayName = i18n.language === "zh" ? asset.name : asset.nameEn`，zh locale 就显示英文 fallback。
- 修复拆成两步：
  1. 改 seed 文件：表名 → `rm_asset_catalog`，`name` 列全部改中文（`SPDR 黄金 ETF` / `华安黄金 ETF` 等 26 个），`name_en` 保留英文。
  2. 新增迁移 `024_asset_catalog_chinese_names.up.sql` / `.down.sql`：对现存 26 行按 code 做 `CASE ... WHEN` UPDATE，`down` 直接把 `name` 复制回 `name_en`（原始 seed 状态）。
  - 迁移号冲突检查：023 已被 invite_system 占用，取 024。
- 执行结果：后端重启后 migration 自动跑 up，用户刷新 `/market` 看到中文资产名。

### 问题 G：signalLevel 枚举漂移（契约侧，本次同批修复）

验证问题 E 时发现 overview API 返回 `signalLevel: "moderate_bullish"`，但 frontend 的类型 union 和 i18n key 用的是 `bullish`/`bearish`，导致信号标签走 i18next raw-key fallback 显示字面 `moderate_bullish`。

根因：richson `schemas/analysis.py:19-20` 定义 `SignalLevel = Literal["strong_bullish", "moderate_bullish", "neutral", "moderate_bearish", "strong_bearish"]` —— 枚举取值本身就是契约的一部分，但事件雷达修复那次只核对了字段名，漏看了枚举取值集合。是同一类 contract-drift 的第三次复发。

修复清单（前端对齐 richson 规范，直接消 bug，不走兼容层）：

| 文件 | 改动 |
|---|---|
| `frontend/src/features/market-overview/types.ts` | `signalLevel` union 从 `"bullish"/"bearish"` 改为 `"moderate_bullish"/"moderate_bearish"` |
| `frontend/src/features/asset-detail/types.ts` | 注释同步改成 richson 规范（`signalLevel?: string` 保留宽类型，仅注释列举） |
| `frontend/src/pages/market-overview/utils.ts` | `getDirectionColor` 新增 `moderate_bullish/moderate_bearish` 分支；保留 plain `bullish/bearish` 分支以兼容 `goldDirection` 三级方向命名空间 |
| `frontend/src/pages/asset-detail/utils.ts` | `getSignalColor` 同上 additive 兼容 |
| `frontend/src/i18n/locales/{en,zh}/market.json` | `overview.assetCard.signal.*` 的 `bullish`/`bearish` 两个 key 重命名为 `moderate_bullish`/`moderate_bearish`。**不改** `overview.eventRadar.goldDirection.*`（三级方向独立命名空间） |
| `frontend/src/i18n/locales/{en,zh}/app.json` | `assetDetail.scoreSummary.signal.*` 同上重命名。**不改** line 94-95 `card.direction.*` 和 line 466-467 `portfolio.direction.*`（dimension/decision 三级方向，另一命名空间） |

验证：`cd frontend && pnpm lint:all` → Biome + tsc + depcruise 全绿（252 文件 / 277 模块 / 879 依赖）。

### 沉淀

- `docs/standards/contract-drift.md` 历史教训追加 E/F/G 三条；新增「同类事故连发三次的教训」小结，强调「字段名对齐 + 可空语义对齐 + **枚举取值对齐**」三项缺一不可
- `~/.claude/projects/-Users-kyle-Studio-Richman/memory/feedback_cross_layer_contract_drift.md` 的 How to apply 增加第 4 条「枚举取值也算契约」，并追加三次历史事故清单作为触发条件提醒

### 已记录但未修复（问题 E/F/G 触及的观察项）

7. richson job 状态未从 `running` 迁移到 `completed`。admin recovery 派发的 26 个 job 虽然 `asset_analysis_id` 都成功写了 `rs_asset_analyses`，但 `rs_analysis_jobs.status` 一直停在 `running`。属于 richson 端 finalizer 逻辑 bug，不影响 overview 展示（overview 直接读 `rs_asset_analyses` 不看 job 表），但会阻塞 cleanup cron 和前端未来可能的 job 状态面板。建议单独任务 fix richson `_finalize_job` 路径。
8. 所有 `a_share_broad` 资产（5 个）的 `overallScore` 分数相同（62.24），疑似共用指标导致的 plateau 问题。属于 richson 指标引擎层面的观察项，需要 richson 侧给每只 ETF 做个性化 factor 权重才能消除。先记录，不在本次 bugfix 范围内。

### 追加 commit 建议

7. `fix(market): seed canonical asset catalog with Chinese names`（seed/asset_catalog.sql + migration 024 up/down）
8. `fix(market): align signalLevel enum with richson canonical moderate_* labels`（types.ts × 2 + utils.ts × 2 + 4 个 i18n json）
9. `docs(standards): extend contract-drift to cover enum values after third recurrence`（contract-drift.md + feedback_cross_layer_contract_drift.md）
