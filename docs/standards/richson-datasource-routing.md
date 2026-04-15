# richson 数据源路由规范

richson 是 richman 的量化计算 sidecar，统一对接 AKShare / Yahoo Finance / Stooq 等行情数据源。本规范定义任何涉及 OHLCV / 价格 / 历史行情的 endpoint 必须遵守的数据源选择纪律，防止 A 股 ETF 被错路到只覆盖美股的数据源、再衍生出 502/503 雪崩。

## 强制规则

### 规则 1 必须走 routing 层

richson 任何返回行情数据的代码（API endpoint / 批分析 pipeline / CLI 工具）禁止直接实例化 `YahooFinanceClient` / `StooqClient` / `AKShareClient` 然后硬编码 fall-through 链。必须通过 `richson.datasources.routing.fetch_ohlcv(code)` 取数。

理由：硬编码 `Yahoo -> Stooq` 会让 A 股 ETF（159915 / 518880 等 6 位纯数字代码）触发以下故障链：

1. Yahoo 返回 "possibly delisted" 空数据
2. Stooq 返回 200 OK 但 body 是 HTML 错误页
3. pandas 解析报 "Expected 1 fields"，被 retry loop 当瞬时错重试 3 次（每次 1.4 s）
4. richson 502 -> richman 503 -> 前端渲染失败

这条故障链在 richman 后端早已通过 `backend/internal/datasource/fetcher.go:resolveGoldETFFetcher` 解决（注释明确写"前一个规则只匹配首位 5 导致 SZSE 159xxx 被错路到 Yahoo 然后 429"）。richson 端长期遗漏同样的路由。

### 规则 2 routing 与 catalog 必须同步

`backend/db/seed/asset_catalog.sql` 的每条记录都有 `data_source` 字段（值为 `akshare` / `yahoo` 等）。该字段是契约的权威面，routing 的 pattern 判断必须与之一致：

- 6 位纯数字代码（159xxx / 160xxx / 51xxxx / 56xxxx / 588xxx）-> `akshare`
- 字母 / 含 `.` `^` `=` 等 Yahoo 符号 -> `yahoo`（Stooq 兜底）

新增任何 catalog 资产时，data_source 与 routing pattern 必须吻合。如果未来引入新的命名空间（如 HK 股 0xxxx.HK / 加密货币 BTC-USD），同步在 `richson.datasources.routing` 加 pattern 分支。

### 规则 3 currency 跟随 routing

`/market/ohlcv` 等 endpoint 的 `currency` 字段不允许独立硬编码。必须调用 `richson.datasources.routing.resolve_currency(code)`，与 routing 的 source 选择保持同源，禁止两套判断条件分别维护后悄悄漂移。

历史教训：旧实现写过 `currency = "CNY" if "518880" in asset_code else "USD"`，只对单个特例正确，所有其他 A 股 ETF 都被标 USD。

### 规则 4 datasource client 区分 transient vs permanent 错误

`StooqClient.get_ohlcv` 类的 fetch 方法在重试前必须先判断错误是否可重试：

- transient（HTTP timeout / 5xx / 网络错）-> 重试
- permanent（200 OK 但 body 不是合法 CSV / 数据为空）-> 立即返 None，不进 retry loop

判定依据：response body 是否符合预期格式（如 Stooq CSV 必须以 `Date,` 开头）。把不可恢复的错放进重试只会浪费 N × timeout 秒、刷脏日志、推迟向上游报错。

### 规则 5 路由变更必须更新 routing 测试

`tests/test_routing.py` 用 parametrize 覆盖每种 code 形态。新增 pattern 分支（HK / crypto / 个股 6 位）必须同步加测例，断言 `is_a_share_code` / `resolve_currency` / `fetch_ohlcv` 的取值，禁止只改 routing 不改测试。

## 适用范围

| 文件 | 是否受规范约束 |
|---|---|
| `richson/api/market.py:get_ohlcv` | 是，必须走 routing |
| `richson/core/pipeline.py:run_layer1_gold` | 是，批分析也要走 routing |
| `richson/api/market.py:get_market_regime` | 否，使用 Yahoo 原生指数符号（`^GSPC` / `^IXIC` / `000001.SS` / `GC=F`）不需要路由分发 |
| `richson/cli/backtest.py` | 是，未来接入 routing |
| `richson/datasources/akshare_client.py` | 否，作为 routing 的目标 client，本身不做选择 |

## 触发条件提醒

任何下列改动启动前都必须检查本规范：

- 新增/修改 `richson/api/*.py` 中触及 OHLCV / price 的 endpoint
- 新增/修改 `richson/core/pipeline.py` 中的 `_fetch_*` 函数
- 新增 catalog 资产到 `backend/db/seed/asset_catalog.sql`
- 在 richson 引入新的数据源 client 包

## 历史

2026-04-15 用户报告 `/api/v2/market/159915/ohlcv` 503 + 后端日志刷屏。根因是 richson 端 OHLCV 路径硬编码 Yahoo -> Stooq、漏掉 AKShare 分支；同样的硬编码也存在于批分析 pipeline。本规范沉淀路由纪律，防止后续 endpoint / 分析模块再出同类问题。
