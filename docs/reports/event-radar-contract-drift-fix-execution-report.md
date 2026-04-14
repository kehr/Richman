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

## 已记录但未修复的观察项

1. richson 的 mypy 基线有 106 条 pre-existing 错误，涉及 33 个文件。本次只验证改动未引入新错误，未做统一修复。建议下一个专门的"richson 类型修复"任务处理。
2. 用户 `richson/.env` 中 `PLATFORM_LLM_API_KEY=sk-...` 同样是占位符，本次未改动。若未来 scheduler 用到 platform LLM，会在这里再次踩坑。短期可参考 FRED 方案为 LLM 客户端加同样的占位符检测。
3. `frontend/src/pages/market/MarketOverviewPage.tsx` 与 `frontend/src/pages/market-overview/market-overview-page.tsx` 并存，页面路由上似乎选用了新版。旧文件是否可删除未在本次确认。
4. backend `make check` 依赖 `golangci-lint`，当前本机未安装；本次只跑了 `go build` + `go vet` 替代。建议在 `docs/standards/lint-toolchain.md` 约束的版本下补装。

## 验收后续

- 用户确认端到端可见事件雷达恢复正常后，本次改动可直接 commit 到 main
- commit 建议拆分为三个独立 commit（遵守 `docs/standards/commit-hygiene.md` 一次一主题）：
  1. `fix(events): align event radar DTO names across richson/backend/frontend`（frontend types + section + backend pointers + emailpush 适配）
  2. `fix(richson): short-circuit FRED fetches when api_key is a placeholder`（fred.py + .env.example）
  3. `docs(standards): add cross-layer contract drift discipline`（contract-drift.md + CLAUDE.md 索引）
