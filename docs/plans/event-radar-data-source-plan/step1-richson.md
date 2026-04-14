# Step 1: richson 数据源 + endpoint 重写

## 目标

把硬编码 `_FIXED_EVENTS` 替换为 FRED Releases Calendar + Polymarket 真实数据源，引入 `event_metadata.py` 静态 release 元数据表，扩展 EventItem schema 增加 `source_url` / `source_name` / `release_id`。

## 涉及文件

新建：
- `richson/src/richson/config/event_metadata.py`

修改：
- `richson/src/richson/datasources/fred.py`（新增 `FREDReleaseDate` 数据类 + `get_upcoming_releases()` 方法）
- `richson/src/richson/schemas/events.py`（EventItem 加三个字段）
- `richson/src/richson/api/events.py`（重写整体逻辑，删除 `_FIXED_EVENTS` 与 `_infer_gold_direction`）

## 设计依据

- PRD §4.1（三端 DTO 增量字段）
- PRD §4.3（数据源映射表）
- PRD §6（数据流图）
- PRD §7（错误处理与降级）
- PRD §8.6（已修复 gap 列表中的 FRED httpx 直连决定、include_release_dates_with_no_data=true 必填）
- TRD §2.1（fred.py 扩展细节，包括 disabled 短路、缓存 key、HTTP 参数、白名单过滤）
- TRD §2.2（event_metadata.py 完整内容）
- TRD §2.3（api/events.py 重写伪代码）
- TRD §2.4（schemas/events.py 扩展）
- TRD §7（已知问题：fred_api_key 小写下划线、polymarket end_date key、asyncio.gather 风格）

## 验证标准

每个文件改完立即跑：
- `cd richson && uv run ruff check src/richson/<file>` 通过
- `cd richson && uv run mypy src/richson/<file>` 通过
- 全部完成后跑 `cd richson && uv run pytest tests/` 全部通过（如有相关测试）
- 本地启 richson dev server（`uv run uvicorn richson.app:app --reload --port 8000`），手动 curl `http://localhost:8000/events/radar`：
  - 响应 JSON 字段含 `sourceUrl` / `sourceName` / `releaseId`
  - FRED 条目 sourceUrl 形如 `https://fred.stlouisfed.org/release?rid=N`
  - Polymarket 条目 sourceUrl 形如 `https://polymarket.com/event/<slug>`
  - 临时清空 `FRED_API_KEY` 重启，事件雷达只剩 Polymarket 条目，richson 日志无 ERROR 风暴
- 联调时调用 `https://api.stlouisfed.org/fred/releases?api_key=$FRED_API_KEY&file_type=json&limit=1000`，按 `name` 字段验证 9 个 release_id（CPI/PPI/Employment/PCE/GDP/FOMC/Industrial/Retail/Housing），如有偏差立即更正 `event_metadata.py`

## 依赖

- 无前置依赖
- 与 Step 2 / Step 3 完全独立，可并行执行
- 验收门：本 step 内的所有验证标准全部通过才能合入

## Commit 拆分

按 commit-hygiene 单主题原则建议拆 3 个 commit：
1. `feat(richson): add FRED release metadata module` （新建 event_metadata.py）
2. `feat(richson): fetch upcoming releases via FRED REST` （fred.py 扩展）
3. `feat(richson): rebuild event radar from FRED + Polymarket` （events.py + schemas/events.py）
