# 命名规范

## 文件与目录

### 前端 (TypeScript/React)

| 类型 | 规则 | 示例 |
|------|------|------|
| TypeScript 文件 | kebab-case | `rate-limiter.ts` |
| React 组件 | PascalCase.tsx | `DecisionCard.tsx` |
| 测试文件 | *.test.ts / *.test.tsx | `analysis.test.ts` |
| 目录 | kebab-case | `decision-card/` |

### 前端目录约定

| 目录 | 用途 |
|------|------|
| `pages/{module}/` | 页面组装层 |
| `features/{module}/` | 自包含业务模块 |
| `domain/{module}/` | 跨模块基础设施 |
| `layouts/` | 页面布局 |
| `ui-kit/eat/` | Ant Design 组件 barrel |
| `config/` | 路由和主题配置 |

### 后端 (Go)

| 类型 | 规则 | 示例 |
|------|------|------|
| Go 文件 | snake_case | `trend_analyzer.go` |
| Go 测试文件 | *_test.go | `trend_analyzer_test.go` |
| 目录 | 小写单词，不用分隔符 | `analysis/`、`datasource/` |
| 包名 | 小写单词，不用分隔符 | `package notification` |

### 文档

| 类型 | 规则 | 示例 |
|------|------|------|
| PRD | 英文 kebab-case + `-prd` 后缀 | `richman-prd.md` |
| TRD | 英文 kebab-case + `-trd` 后缀 | `analysis-engine-trd.md` |
| Plan | 英文 kebab-case + `-plan` 后缀 | `mvp-setup-plan.md` |
| Spec | 英文 kebab-case + `-spec` 后缀 | `decision-card-spec.md` |
| 标准文档 | 英文 kebab-case | `naming.md` |


## TypeScript 标识符

| 类型 | 规则 | 示例 |
|------|------|------|
| 变量、函数 | camelCase | `fetchPortfolio` |
| 类、接口、类型 | PascalCase | `DecisionCardDto` |
| 接口 | 不加 `I` 前缀 | `AnalysisResult`（不是 `IAnalysisResult`） |
| 模块级常量 | SCREAMING_SNAKE_CASE | `MAX_HOLDINGS` |
| 枚举成员 | PascalCase | `TrendDirection.Upward` |
| 泛型参数 | 单大写字母或语义名 | `T`、`TData` |
| Zod schema | camelCase + Schema 后缀 | `createHoldingSchema` |
| Zod 推断类型 | PascalCase | `CreateHolding` |


## Go 标识符

| 类型 | 规则 | 示例 |
|------|------|------|
| 导出函数/类型 | PascalCase | `AnalyzeTrend`、`DecisionCard` |
| 非导出函数/变量 | camelCase | `calculateScore` |
| 常量 | PascalCase 或 SCREAMING_SNAKE_CASE | `MaxHoldings` 或 `MAX_HOLDINGS` |
| 接口 | PascalCase，动词/名词 | `Analyzer`、`NotificationSender` |
| 结构体 | PascalCase 名词 | `TrendResult`、`Portfolio` |
| 方法接收者 | 类型名首字母小写 | `func (t *TrendAnalyzer) Analyze()` |


## 数据库

| 类型 | 规则 | 示例 |
|------|------|------|
| 表名 | snake_case 复数 | `holdings`、`decision_cards` |
| 列名 | snake_case | `cost_price`、`created_at` |
| 主键 | `{表名单数}_id` | `holding_id`、`user_id` |
| 外键列 | 与引用的主键同名 | `user_id` |
| 索引 | `idx_{表}_{列}` | `idx_holdings_user_id` |
| 复合索引 | `idx_{表缩写}_{列1}_{列2}` | `idx_hld_user_asset` |
| 唯一约束 | `uq_{表}_{列}` | `uq_users_email` |

**单词原则：** 优先使用简短列名，外键关系已提供上下文。
- 用 `name`（不是 `asset_name`）
- 用 `cost`（不是 `cost_price`，如果表上下文已经明确）
- 保留复合词：`created_at`、`is_deleted`、`invite_code`


## API

| 类型 | 规则 | 示例 |
|------|------|------|
| URL 路径 | kebab-case 复数 | `/api/v1/decision-cards` |
| 查询参数 | camelCase | `?pageSize=20&assetType=etf` |
| JSON 字段 | camelCase | `{ "costPrice": 3.85 }` |
| API 版本 | URL 前缀 | `/api/v1/` |


## Git

| 类型 | 规则 | 示例 |
|------|------|------|
| 分支名 | kebab-case + 类型前缀 | `feat/decision-card`、`fix/push-timing` |
| 提交信息 | 英文、祈使语气 | `Add three-dimension analysis engine` |


## 环境变量

| 规则 | 示例 |
|------|------|
| SCREAMING_SNAKE_CASE | `DATABASE_URL`、`LLM_API_KEY`、`NEXT_PUBLIC_API_BASE` |
