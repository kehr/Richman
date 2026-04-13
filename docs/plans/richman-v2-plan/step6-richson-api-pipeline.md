# Step 6: richson Pipeline + API + Middleware + CLI + Observability

> Phase 2 | 并行组 R3 (单独执行) | 前置: Steps 4, 5

## 任务目标

实现 richson 的核心编排层和对外接口：L1->L2->L3 pipeline 编排、全部 11 个 API 端点、认证/追踪中间件、CLI 命令（backfill/backtest/update-weights）、structlog 日志配置、数据生命周期清理和 asyncio 内部调度。同时处理全部 richson 已知问题。

## 涉及文件

### 创建

**Pipeline：**
- `richson/src/richson/core/pipeline.py` -- L1 -> L2 -> L3 编排

**API 端点：**
- `richson/src/richson/api/__init__.py`
- `richson/src/richson/api/health.py` -- GET /health
- `richson/src/richson/api/jobs.py` -- POST /jobs/analyze-asset, POST /jobs/batch-analyze, GET /jobs/{jobId}
- `richson/src/richson/api/analysis.py` -- POST /analyze/holding, POST /analyze/demo-plan
- `richson/src/richson/api/market.py` -- GET /market/regime, GET /market/ohlcv/{code}
- `richson/src/richson/api/assets.py` -- GET /assets/{code}/score-history
- `richson/src/richson/api/events.py` -- GET /events/radar
- `richson/src/richson/api/content.py` -- POST /content/weekly-insight

**CLI：**
- `richson/src/richson/cli/__init__.py`
- `richson/src/richson/cli/backfill.py` -- backfill --days 90
- `richson/src/richson/cli/backtest.py` -- backtest --start --end
- `richson/src/richson/cli/weights.py` -- update-weights

**Middleware + Observability：**
- `richson/src/richson/middleware/__init__.py`
- `richson/src/richson/middleware/auth.py` -- INTERNAL_API_KEY 校验
- `richson/src/richson/middleware/tracing.py` -- X-Request-ID 追踪
- `richson/src/richson/middleware/logging_config.py` -- structlog JSON 配置

**Scheduling：**
- `richson/src/richson/tasks/__init__.py`
- `richson/src/richson/tasks/scheduler.py` -- asyncio 事件调度 + 数据清理

### 修改

- `richson/src/richson/main.py` -- 注册全部 router + middleware + lifespan hooks

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| Pipeline L1->L2->L3 编排 | SS3.1 架构设计 | richson SS7.1 |
| 异步标的分析 (POST /jobs/*) | SS3.1 | richson SS5.1 |
| 同步持仓分析 (POST /analyze/holding) | SS8.1 | richson SS5.2 |
| Demo plan (POST /analyze/demo-plan) | SS5.2.4 | richson SS5.3 |
| 市场体制 (GET /market/regime) | SS4.2.1 | richson SS5.3 |
| OHLCV (GET /market/ohlcv) | SS5.2.2 | richson SS5.3 |
| 评分历史 (GET /assets/score-history) | SS5.2.2 | richson SS5.3 |
| 事件雷达 (GET /events/radar) | SS4.2.4 | richson SS5.3 |
| 周报内容 (POST /content/weekly-insight) | SS10.3 | richson SS9.3 |
| 健康检查 (GET /health) | - | richson SS5.4 |
| INTERNAL_API_KEY 认证 | - | richson SS4.2 |
| X-Request-ID 追踪 | - | richson SS4.3 |
| 错误响应格式 | - | richson SS4.4 |
| structlog JSON 日志 | - | richson SS11.1 |
| backfill CLI | - | richson SS13.3 |
| backtest CLI | - | richson SS18 |
| update-weights CLI | - | richson SS6.2 |
| 数据清理 (rs_asset_analyses 保留策略) | - | richson SS13.1 |
| asyncio 事件调度 (Polymarket 每小时) | - | richson SS9.3 |

## 关键约束 + 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G1.1 端口暴露 | 仅绑定 127.0.0.1，expose 不 publish | richson SS21.1 |
| G1.2 healthcheck curl 依赖 | Dockerfile 安装 curl 或用 python httpx 替代 | richson SS21.2 |
| G1.3 asyncio 阻塞 | 耗时计算用 to_thread / ProcessPoolExecutor | richson SS21.3 |
| G1.4 序列 RESTART WITH | rs_event_alerts + dimensions 表序列设置 | richson SS21.4 |
| G1.5 LLM 成本上限 | DAILY_LLM_BUDGET_USD 环境变量 + 检查 | richson SS21.5 |
| G1.6 集中度文本硬编码 | locale 参数化 | richson SS21.6 |
| G1.7 backfill 标记 | backfill 数据 percentile 排除评估 | richson SS21.7 |
| G1.8 validDays 下界 | Pydantic Field(ge=1, le=90) | richson SS21.8 |
| G1.9 apiKey 日志脱敏 | repr=False + middleware sanitization | richson SS21.9 |

- pipeline.py 需处理 L2 失败降级（跳过 LLM 调整，使用量化基础分）
- 异步分析通过 asyncio.create_task 后台执行，状态写入 rs_analysis_jobs
- demo-plan 请求先查 DB 预计算结果，无结果时 fallback 调用 pipeline
- 同一标的重复 job 由 partial unique index 防护（返回 409）
- API 错误码与 richman SS3.4 errorMap 对齐

## 验证标准

- [ ] `uvicorn richson.main:app --host 127.0.0.1 --port 8001` 启动成功
- [ ] GET /health 返回 200 + 组件状态
- [ ] POST /jobs/analyze-asset 返回 202 + jobId（需连接 DB）
- [ ] GET /market/regime 返回体制数据（需外部数据源可用）
- [ ] 认证中间件拒绝无 API key 请求（返回 401）
- [ ] X-Request-ID 在日志中正确关联
- [ ] `python -m richson.cli.backfill --days 1` 可执行（连接 DB + 数据源）
- [ ] structlog 输出 JSON 格式
- [ ] 全部 9 项已知问题在代码中有对应处理

## 变更点清单覆盖

C2.1-C2.11 (11), C3.1 (1), C8.1-C8.3 (3), C9.1-C9.5 (5), G1.1-G1.9 (9) = **29 项**
