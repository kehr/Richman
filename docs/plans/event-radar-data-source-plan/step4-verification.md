# Step 4: 三端联调 + lint 总验 + 视觉验收

## 目标

在 Step 1/2/3 全部 lint 通过后，启动整套环境做端到端联调，按 contract-drift checklist 验证三端字段对齐，并完成视觉验收。

## 涉及文件

无新文件；只跑命令 + 抓包 + 修复发现的问题。

修复任何发现问题后，相应改动以 `fix(...)` commit 为主题。

## 设计依据

- TRD §5（端到端数据链路追踪表）
- TRD §6（多角色审查矩阵）
- TRD §8（验证清单）
- contract-drift.md §对齐规则
- contract-drift.md §验证检验点

## 验证标准

### 4.1 三端独立 lint 总检

- `cd richson && uv run ruff check src/ && uv run mypy src/`
- `cd backend && make check`
- `cd frontend && pnpm lint:all && pnpm build`

### 4.2 启动顺序

```
1. docker-compose up -d  (PostgreSQL 5433)
2. cd richson && uv run uvicorn richson.app:app --reload --port 8000
3. cd backend && make dev (port 8100)
4. cd frontend && pnpm dev (port 3000)
```

### 4.3 端到端 Network 抓包

DevTools Network 打开 http://localhost:3000/market：

- [ ] `GET /api/v2/events/radar` 200
- [ ] 响应 JSON `data.events[]` 每个对象包含 `sourceUrl` / `sourceName` / `releaseId` 字段
- [ ] FRED 条目：`sourceUrl=https://fred.stlouisfed.org/release?rid=N`，`sourceName="FRED"`，`releaseId=N`，`probability=null`
- [ ] Polymarket 条目：`sourceUrl=https://polymarket.com/event/<slug>`，`sourceName="Polymarket"`，`releaseId=null`，`probability=number`
- [ ] 所有 nullable 字段在数据缺失时是 JSON `null`，不是 `0` / `""`（contract-drift 的 null 语义防御）

### 4.4 视觉验收

行情概览页 (http://localhost:3000/market)：
- [ ] 事件雷达条目日期不再是 `today + 3/5/7/10/14`，而是 FRED 真实发布日历
- [ ] 鼠标悬停事件行：cursor=pointer，背景色变浅
- [ ] HTML title 显示 "Source: FRED" 或 "来源：FRED"
- [ ] 点击 FRED 事件：新标签打开正确的 release 页（手动验证 CPI rid=10）
- [ ] 点击 Polymarket 事件：新标签打开 polymarket.com/event/<slug>

资产详情页 (任选一只资产)：
- [ ] risk-tab 下"近期事件"区域不再显示 "—"
- [ ] 显示与行情概览页一致的事件列表（同一个 query，TanStack Query 跨页面缓存）
- [ ] 同样可点击跳转

i18n 切换：
- [ ] 英文环境下提示文案为 "Source: FRED" / "Open source in new tab"
- [ ] 中文环境下提示文案为 "来源：FRED" / "在新标签页打开数据源"

降级测试：
- [ ] 临时清空 `FRED_API_KEY` 重启 richson，事件雷达只显示 Polymarket 条目，不出现红色 Alert
- [ ] richson 日志无 `ERROR` 级 fred 风暴，仅 startup 一次 warning

### 4.5 contract-drift checklist

按 contract-drift.md §审查清单：
- [ ] 字段名三端完全一致（sourceUrl / sourceName / releaseId 大小写）
- [ ] 可空字段三端 nullable 表达：Pydantic `| None` + Go `*T` + TS `| null`
- [ ] 没有用 Go 值类型暗示 nullable（grep `EventItem` 检查无 `string SourceUrl`）
- [ ] backend struct 已包含三个新字段
- [ ] frontend 防御 null：`event.sourceUrl != null` 判空，不用 `!== null`

## 依赖

- Step 1 / Step 2 / Step 3 全部完成

## Commit 拆分

发现问题修复时按主题独立 commit。验收完成后追加：
- `docs(reports): finalize event radar data source execution report`（更新执行报告 §Step 执行状态）
