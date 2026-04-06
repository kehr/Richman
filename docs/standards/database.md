# 数据库设计规范

## 数据库

PostgreSQL（Supabase 托管或自建）


## 主键

每张表的主键为 `{表名单数}_id`，使用 BIGSERIAL 自增。

| 表 | 主键 | 自增起始值 |
|---|------|-----------|
| users | user_id | 10000 |
| holdings | holding_id | 20000 |
| trades | trade_id | 30000 |
| analysis_results | analysis_result_id | 40000 |
| decision_cards | decision_card_id | 50000 |
| notification_channels | notification_channel_id | 60000 |
| notification_logs | notification_log_id | 70000 |
| asset_catalog | asset_catalog_id | 80000 |
| invite_codes | invite_code_id | 90000 |
| plans | plan_id | 100000 |

新增表以 10000 为间隔递增。


## 审计字段

每张表必须包含以下审计字段：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| created_at | TIMESTAMPTZ | NOW() | 创建时间 |
| updated_at | TIMESTAMPTZ | NOW() | 更新时间（UPDATE 时自动刷新） |
| creator | VARCHAR(64) | 'system' | 创建者 |
| modifier | VARCHAR(64) | 'system' | 最后修改者 |
| is_deleted | SMALLINT | 0 | 0=有效，1=已删除 |

使用 sqlc 模板或辅助函数避免重复定义。


## 软删除

- 所有删除使用软删除（`is_deleted = 1`），不执行物理 DELETE
- Repo 查询默认过滤 `WHERE is_deleted = 0`
- 需要查询已删除记录时通过参数覆盖


## 命名

| 类型 | 规则 | 示例 |
|------|------|------|
| 表名 | snake_case 复数 | `holdings`、`decision_cards` |
| 列名 | snake_case | `cost_price`、`asset_code` |
| 主键 | `{表名单数}_id` | `holding_id` |
| 外键列 | 与引用的主键同名 | `user_id` |
| 索引 | `idx_{表}_{列}` | `idx_holdings_user_id` |
| 复合索引 | `idx_{表缩写}_{列1}_{列2}` | `idx_hld_user_asset` |
| 唯一约束 | `uq_{表}_{列}` | `uq_users_email` |

**单词原则：** 优先简短列名。外键关系已提供上下文。
- 用 `name`（不是 `asset_name`，如果表上下文已明确）
- 保留必要复合词：`created_at`、`is_deleted`、`invite_code`、`cost_price`


## JSON 列

- 通用高频字段 -> 独立列（可搜索、可索引）
- 特定来源的扩展数据 -> JSON 列
- JSON 列不建索引
- 大 JSON 使用 JSONB 类型（PostgreSQL 原生支持）
- 使用 PostgreSQL JSON 操作符查询子字段


## 索引策略

所有查询默认 `is_deleted = 0`。创建复合索引时以 is_deleted 为前缀：

```sql
CREATE INDEX idx_hld_deleted_user ON holdings (is_deleted, user_id);
CREATE INDEX idx_dc_deleted_holding ON decision_cards (is_deleted, holding_id);
CREATE INDEX idx_users_deleted_email ON users (is_deleted, email);
```


## 查询模式

**Upsert（原子操作）：**
使用 `ON CONFLICT ... DO UPDATE`，不用 check-then-insert（有竞态条件）：

```sql
INSERT INTO holdings (user_id, asset_code, cost_price, position_ratio)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, asset_code) WHERE is_deleted = 0
DO UPDATE SET cost_price = EXCLUDED.cost_price, position_ratio = EXCLUDED.position_ratio;
```

**批量写入：**
单条 INSERT 多 VALUES，不逐行插入。

**并行查询：**
Go 中使用 goroutine + errgroup 并行执行独立读查询。

**列选择：**
只 SELECT 需要的列，避免 SELECT *，尤其是包含 JSON 列的表。


## 表写入所有权

每张表有唯一的写入所有者，其他域只读：

| 域 | 写入表 | 只读表 |
|---|--------|--------|
| auth | users、invite_codes、plans | - |
| portfolio | holdings、trades | users |
| analysis | analysis_results、decision_cards | holdings |
| notification | notification_channels、notification_logs | users、decision_cards |
| asset | asset_catalog | - |


## 连接池配置

```go
// PostgreSQL 连接池建议配置
MaxOpenConns: 25
MaxIdleConns: 10
ConnMaxLifetime: 5 * time.Minute
ConnMaxIdleTime: 1 * time.Minute
```


## 迁移管理

- 使用 Go 迁移工具（如 golang-migrate 或 goose）
- 迁移文件放在 `db/migration/` 目录
- 文件命名：`{序号}_{描述}.up.sql` / `{序号}_{描述}.down.sql`
- 每次变更必须有对应的 up 和 down 迁移
- 生产环境迁移前先在测试环境验证
