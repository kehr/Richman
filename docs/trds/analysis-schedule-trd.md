# 自动分析调度策略 TRD

## 数据库 Schema

### user_schedule_settings

```sql
CREATE TABLE user_schedule_settings (
    id                      BIGSERIAL PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id),

    -- global
    global_frequency        TEXT NOT NULL DEFAULT 'daily',
    -- values: every_window | daily | every_2_days | every_3_days | weekly | custom
    global_frequency_days   INT,           -- only valid when global_frequency = 'custom', range 1-30

    -- a_share windows
    a_share_pre_enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    a_share_pre_time        TIME    NOT NULL DEFAULT '08:30',
    a_share_pre_custom      BOOLEAN NOT NULL DEFAULT FALSE,  -- true if user modified default
    a_share_post_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    a_share_post_time       TIME    NOT NULL DEFAULT '15:05',
    a_share_post_custom     BOOLEAN NOT NULL DEFAULT FALSE,
    a_share_frequency       TEXT,          -- null = follow global
    a_share_frequency_days  INT,

    -- us_stock / gold windows (times stored in Asia/Shanghai)
    us_pre_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
    us_pre_time             TIME    NOT NULL DEFAULT '20:30', -- EDT default; auto-adjusted if not custom
    us_pre_custom           BOOLEAN NOT NULL DEFAULT FALSE,
    us_post_enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    us_post_time            TIME    NOT NULL DEFAULT '04:05', -- EDT default; auto-adjusted if not custom
    us_post_custom          BOOLEAN NOT NULL DEFAULT FALSE,
    us_frequency            TEXT,
    us_frequency_days       INT,

    -- hk reserved (no active columns yet)

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted              BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (user_id)
);
```

### holding_schedule_overrides

```sql
CREATE TABLE holding_schedule_overrides (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id),
    holding_id      BIGINT NOT NULL REFERENCES holdings(id),
    frequency       TEXT,    -- null = follow market; same values as global_frequency
    frequency_days  INT,
    window          TEXT,    -- null = follow market; values: pre | post | both
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_deleted      BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (user_id, holding_id)
);
```

## API 设计

### GET /api/v1/settings/schedule

返回当前用户完整调度配置。若用户尚无记录，返回系统默认值（不自动写库）。

Response body:
```json
{
  "globalFrequency": "daily",
  "globalFrequencyDays": null,
  "markets": {
    "a_share": {
      "frequency": null,
      "frequencyDays": null,
      "preWindow":  { "enabled": true,  "time": "08:30", "isCustom": false },
      "postWindow": { "enabled": true,  "time": "15:05", "isCustom": false }
    },
    "us_stock": {
      "frequency": null,
      "frequencyDays": null,
      "preWindow":  { "enabled": false, "time": "20:30", "isCustom": false },
      "postWindow": { "enabled": true,  "time": "04:05", "isCustom": false }
    }
  }
}
```

### PUT /api/v1/settings/schedule

Upsert 用户调度配置。触发后端调度器重新加载该用户的 cron 条目。

Request body 同 GET response 结构，后端校验：
- `globalFrequency` 必须是合法枚举值
- `globalFrequencyDays` 在 custom 时必须为 1-30
- 时间字段格式 `HH:MM`，分钟必须是 5 的倍数
- 时间范围约束：A 股盘前 07:00-09:29，盘后 15:00-20:00；美股盘前 20:00-23:00，盘后 04:00-08:00

### GET /api/v1/holdings/:id/schedule

返回单持仓调度覆盖及计算后的下次分析时间。

Response body:
```json
{
  "holdingId": 123,
  "frequency": null,
  "frequencyDays": null,
  "window": null,
  "nextAnalysisAt": "2026-04-11T06:00:00+08:00"
}
```

`nextAnalysisAt` 由后端按「持仓覆盖 > 市场设置 > 全局默认」三层优先级计算。

### PUT /api/v1/holdings/:id/schedule

Upsert 持仓级覆盖。`null` 表示跟随上层，前端删除覆盖时传 `null`。

## 后端层级结构

```
handlers/schedule/
  get_settings.go
  update_settings.go
  get_holding_schedule.go
  update_holding_schedule.go
service/schedule/
  service.go          -- CRUD + next_analysis_at 计算
  dst.go              -- NYSE DST 时间表，返回当前 EDT/EST offset
  scheduler.go        -- 替换原 analysis/scheduler.go 的硬编码逻辑
repo/schedule/
  queries.sql         -- sqlc 源文件
  db.go               -- sqlc 生成（不手写）
```

### DST 感知逻辑（dst.go）

NYSE DST 规则（固定规律，不调外部 API）：
- 夏令（EDT, UTC-4）：每年 3 月第二个周日 02:00 开始
- 冬令（EST, UTC-5）：每年 11 月第一个周日 02:00 结束

`IsEDT(t time.Time) bool` 返回给定时刻是否处于夏令时。

US 窗口默认时间（非 custom）按 DST 计算：
- 盘前 = NYSE 开盘（09:30 东部时间）- 1h，转换为 Asia/Shanghai
- 盘后 = NYSE 收盘（16:00 东部时间）+ 5min，转换为 Asia/Shanghai

DST 边界日（3 月第二个周日、11 月第一个周日）凌晨 03:00 Asia/Shanghai 时间，后端触发一次所有 us_pre_custom=false / us_post_custom=false 用户的时间更新，并重载调度器。

### Scheduler 重设计

原 `scheduler.go` 硬编码三个 cron 条目，改为动态方案：

1. 启动时从 `user_schedule_settings` 加载所有用户配置；无配置的用户使用系统默认（等价于 daily + 原有三个时间窗口）
2. 为每个启用的时间窗口注册 cron 条目（`cron.AddFunc`），条目 ID 格式：`{userID}:{market}:{pre|post}`
3. PUT /settings/schedule 后，服务层调用 `scheduler.ReloadUser(userID)`，移除该用户旧条目并重新注册
4. cron 触发时，检查该用户此持仓的「上次分析时间」是否满足 frequency 最小间隔；不满足则跳过

### 盘前信息增量

`runPreWindowJob()` 在调用现有 AI/规则分析前，额外拉取：
- 区间价格变动（上次分析后至当前的 OHLCV，来源同现有数据管道）
- 新闻摘要（若数据管道已有，注入 prompt context；若无则跳过，不阻塞分析）

信息增量以结构化 context 附加到 prompt，不改变现有分析输出格式。

## 前端组件树

```
SettingsPage
  ScheduleTab                     (新增)
    GlobalFrequencySelector        (新增，可复用)
    MarketWindowCard (x2: A股/美股) (新增)
      WindowToggleRow (x2)         (新增)
    HKPlaceholderCard              (新增)

HoldingDetailPage / MetaSidebar
  AnalysisMetaSection             (现有，扩展)
    HoldingScheduleSection         (新增)
      FrequencyOverrideSelect      (复用 GlobalFrequencySelector 逻辑)
      WindowOverrideSelect         (新增)
```

## 前端 API Hooks

```typescript
// features/schedule/api.ts
interface ScheduleSettingsDTO { ... }
interface HoldingScheduleDTO { ... }

function fetchScheduleSettings(): Promise<ScheduleSettingsDTO>
function updateScheduleSettings(data: ScheduleSettingsDTO): Promise<ScheduleSettingsDTO>
function fetchHoldingSchedule(holdingId: number): Promise<HoldingScheduleDTO>
function updateHoldingSchedule(holdingId: number, data: Partial<HoldingScheduleDTO>): Promise<HoldingScheduleDTO>

// features/schedule/useSchedule.ts
function useScheduleSettings(): UseQueryResult<ScheduleSettingsDTO>
function useUpdateScheduleSettings(): UseMutationResult<...>
function useHoldingSchedule(holdingId: number): UseQueryResult<HoldingScheduleDTO>
function useUpdateHoldingSchedule(): UseMutationResult<...>
```

## i18n 键命名空间

新增 `settings.schedule.*`：

```json
{
  "schedule": {
    "title": "调度策略",
    "description": "配置各市场的分析时间窗口和触发频率。",
    "globalFrequency": {
      "label": "全局默认频率",
      "every_window": "每个窗口",
      "daily": "每日",
      "every_2_days": "每两日",
      "every_3_days": "每三日",
      "weekly": "每周",
      "custom": "自定义",
      "customDays": "每 {{n}} 天",
      "customPlaceholder": "天数（1–30）",
      "followGlobal": "跟随全局",
      "followMarket": "跟随市场默认"
    },
    "window": {
      "pre": "盘前",
      "post": "盘后",
      "preHint": "开盘前研判 + 信息增量",
      "postHint": "收盘后完整分析",
      "defaultLabel": "默认",
      "customLabel": "已修改",
      "resetTooltip": "重置为默认时间"
    },
    "markets": {
      "a_share": "A 股",
      "us_stock": "美股 / 黄金",
      "hk_stock": "港股",
      "hkComingSoon": "规划中"
    },
    "holdingOverride": {
      "frequency": "分析频率",
      "window": "分析窗口",
      "windowOptions": {
        "follow": "跟随市场默认",
        "pre": "仅盘前",
        "post": "仅盘后",
        "both": "盘前和盘后"
      }
    }
  }
}
```

## analysis-schedule.ts 变更

现有 `computeNextAnalysisTime(now: Date)` 硬编码三个时间点，需改为接受用户设置参数。

后端 `GET /holdings/:id/schedule` 直接返回 `nextAnalysisAt`（后端计算，三层优先级），前端 `MetaSidebar` 改为读取该字段，移除本地计算逻辑。

`analysis-schedule.ts` 保留 `computeNextAnalysisTime` 作为无配置时的客户端 fallback，供无法获取后端数据时降级使用。

## 迁移文件命名

迁移序号需在执行前 `git log --all -- backend/migrations/` 确认已用编号，当前最高为 `007`，本次使用 `008_schedule_settings`。
