# richson 数据源路由修复执行报告

## 任务背景

用户报告后端日志大量 warning + ERROR：

- `/api/v2/market/159915/ohlcv` 返回 503
- richson 日志刷屏 `yahoo: empty data` + `stooq: fetch failed` (Expected 1 fields in line 6, saw 2)
- richman 后端 `market.go:97` 持续打 stacktrace

根因：richson `/market/ohlcv` 与批分析 `core/pipeline.py` 都硬编码 Yahoo -> Stooq fall-through，A 股 ETF（159915 等 6 位纯数字代码）从未走过 AKShare。同样的硬编码也使 Stooq 对 200 OK 的 HTML 错误页做 3 次 1.4s 重试，浪费时间且刷脏日志。

## 执行方式

- 隔离方式：worktree `.claude/worktrees/fix-richson-datasource-routing/`，分支 `fix-richson-datasource-routing`，基于 `55c5631`
- 设计审查：执行 `docs/standards/design-review.md` 的 5-Pass（详见对话）
- 文档分层：bugfix 走简化流程，跳过 PRD/TRD/Plan
- 用户验收偏好：worktree 工作完成后直接 rebase -> ff-merge -> push，主仓库自行验收

## 全局规则遵循

- 零 AI 痕迹：commit message 不含 AI/Claude 字样，分支名不含 ai/claude
- 严格 lint：mypy baseline 105 错（pre-existing），本次改动无新增；ruff 全过；pytest 19/19 过
- 库 API 验证：所有 yfinance/akshare/pandas/structlog 调用均已存在于既有代码中
- 中文文档英文代码：standards 文档用中文，代码注释/log 用英文

## 设计阶段（5-Pass 产物）

完整 5-Pass 分析见对话历史。简记：

- **Pass 1 状态空间**：code 形态分 5 类（6 位纯数字 / 字母 / Yahoo 符号含 `.`/`^`/`=` / 空 / 异常），其中 6 位纯数字 -> AKShare，其他 -> Yahoo->Stooq
- **Pass 2 文件契约**：5 个文件 × 现有契约 × 改动影响表
- **Pass 3 替代路径**：AKShare 临时失败不降级；catalog 与 routing pattern 须同步；A 股个股暂不支持
- **Pass 4 Pre-mortem**：5 条潜在 bug 已分别加防御（注释/grep/Pass 3 排除/可接受/区分 transient vs permanent）
- **Pass 5 自反驳**：pattern-based vs 查 catalog data_source 的 trade-off，选 pattern-based 更简洁

## 实施结果

### 修改文件清单

| 文件 | 变更 |
|---|---|
| `richson/src/richson/datasources/routing.py` | 新建：`is_a_share_code` / `resolve_currency` / `fetch_ohlcv` |
| `richson/src/richson/datasources/__init__.py` | barrel 导出 routing 函数 |
| `richson/src/richson/datasources/stooq.py` | parse error 短路：检测 body 是否以 `Date,` 开头，非 CSV 视为 permanent failure 不重试 |
| `richson/src/richson/api/market.py` | `/ohlcv` endpoint 改用 `fetch_ohlcv` + `resolve_currency`；删除硬编码 currency 判断 |
| `richson/src/richson/core/pipeline.py` | `run_layer1_gold._fetch_all` 改用 `fetch_ohlcv` |
| `richson/tests/__init__.py` | 新建（首次创建 tests 目录） |
| `richson/tests/test_routing.py` | 新建：parametrized 覆盖 `is_a_share_code` / `resolve_currency` / 一致性约束 |
| `docs/standards/richson-datasource-routing.md` | 新建规范，5 条强制规则 |
| `CLAUDE.md` | standards index 追加 richson-datasource-routing 条目 |
| `~/.claude/projects/.../memory/feedback_richson_datasource_routing.md` | 新建 memory entry |
| `~/.claude/projects/.../memory/MEMORY.md` | 新增索引 |

### 关键决策

- routing 用 pattern-based（`len==6 and isdigit()`），不查数据库，理由：catalog seed 全部资产符合此 pattern，引入 DB 查询徒增复杂度
- A 股个股（如 600519 茅台）未在本次扩展支持：catalog 内本无个股资产；若后续加入，需扩展 `AKShareClient` 增加 `ak.stock_zh_a_hist` 包装，再扩 routing pattern
- `/market/regime` 不动：使用 `^GSPC` / `000001.SS` / `GC=F` 等 Yahoo 原生符号，不会触发路由分发
- Stooq parse error 短路用「response body 是否以 `Date,` 开头」而非 catch 特定异常类型：更严格、覆盖所有非 CSV 响应（HTML/纯文本/截断）

## 验证

| 步骤 | 结果 |
|---|---|
| `uv run ruff check src tests` | All checks passed |
| `uv run mypy src` | 105 errors (baseline 不变，无新增) |
| `uv run pytest tests/test_routing.py -v` | 19 passed in 22.23s |
| 主仓库 manual verify (curl 159915 / AAPL / 510300) | 待主仓库 ff-merge 后由用户执行 |

mypy 105 错为 richson 项目历史债务，全部不在本次改动行附近，按"严格 lint"原则不在本次 bugfix 范围内清理。

## 已修复问题

- `/api/v2/market/159915/ohlcv` 503：由 routing 选 AKShare 解决
- 后端日志 stacktrace 风暴：根因是 richson 502，根除 richson 502 即根除日志刷屏
- Stooq 对未知 ticker 的 3 次无效重试：检测非 CSV body 立即短路

## 已记录但未修复的观察项

- richson 全局 mypy 105 错（pre-existing，与本次无关）
- `richman/backend/internal/datasource/akshare/client.go` 是 mock 数据（注释明确写 TODO），生产 A 股数据完全依赖 richson 的 `AKShareClient`
- A 股个股代码（如 600519）若未来加入 catalog，需扩展 `AKShareClient.get_stock_ohlcv` 走 `ak.stock_zh_a_hist`
- richson 端 mypy 历史债清理可作为独立任务跟进

## 无法决策项

无。所有设计决策均已在 5-Pass 阶段收敛，用户已确认推荐方案。

## 验收说明

按用户偏好直接 rebase -> ff-merge -> push。主仓库 manual verify 项目：

```bash
# 启动 richson + backend
cd richson && make dev &  # port 8100
cd backend && make dev    # port 8080

# 验证
curl -s 'http://localhost:8080/api/v2/market/159915/ohlcv' | jq .data.candles[0:2]   # 期望 200 + 候选数据
curl -s 'http://localhost:8080/api/v2/market/518880/ohlcv' | jq .data.currency       # 期望 "CNY"
curl -s 'http://localhost:8080/api/v2/market/QQQ/ohlcv'    | jq .data.currency       # 期望 "USD"
curl -s 'http://localhost:8080/api/v2/market/UNKNOWN/ohlcv' -w '%{http_code}\n'      # 期望 503/502 + 后端日志一次 ERROR 不重试
```
