# API 设计规范

## 版本管理

- URL 前缀版本：`/api/v1`
- 不提供无版本别名
- 非破坏性变更（新增字段/端点）：更新当前版本
- 破坏性变更（重命名、删除、语义变化）：新建版本
- 废弃：使用 `Deprecation` 响应头


## 资源命名

- 复数名词：`/holdings`、`/decision-cards`
- kebab-case：`/decision-cards`、`/notification-channels`
- 嵌套表示从属关系：`/holdings/:id/trades`
- 不用动词：`/holdings`（不是 `/getHoldings`）


## Gin 路由参数冲突预检（强制）

Gin 的 radix tree 路由不允许同一路径层级使用不同的参数名。例如 `/assets/:code` 和 `/assets/:assetType/...` 会在运行时 panic，但编译期不报错。

设计任何新 API 端点前，必须执行以下检查：

```bash
grep -rn '\.GET\|\.POST\|\.PUT\|\.DELETE\|\.PATCH' internal/api/ | grep '<同前缀>'
```

检查内容：
1. 新路径的每个前缀段（如 `/assets/`）是否已被其他 handler 注册
2. 已注册路由中该层级的参数名（如 `:code`）是否与新路由的参数名（如 `:assetType`）一致
3. 不一致则必须改路径前缀（如 `/quotes/` 替代 `/assets/.../quote`）

此规则无例外。违反不会被编译器或 lint 拦截，只会在 `main()` 运行时 panic。


## HTTP 方法和状态码

| 操作 | 方法 | 成功码 | 说明 |
|------|------|--------|------|
| 列表 | GET | 200 | 分页列表 |
| 详情 | GET | 200 | 单个资源 |
| 创建 | POST | 201 | 返回创建的资源 |
| 更新 | PUT/PATCH | 200 | 返回更新后的资源 |
| 删除 | DELETE | 204 | 无响应体 |
| 异步任务 | POST | 202 | 返回 taskId |

**错误码：**

| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验错误 |
| 401 | 未认证 |
| 403 | 权限不足（plan 限制） |
| 404 | 资源不存在 |
| 409 | 业务冲突（如重复创建） |
| 429 | 请求频率限制 |
| 500 | 内部错误 |


## 请求校验

- 所有路由使用 Gin binding tag + validator 做参数校验
- 校验 schema 集中定义
- 不手动解析参数
- 查询参数的类型转换由框架处理


## 响应格式

**单个资源：**

```json
{
  "data": {
    "holdingId": 20001,
    "assetName": "CSI 300 ETF",
    "costPrice": 3.85,
    "positionRatio": 25.0
  }
}
```

**分页列表：**

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "pageSize": 20,
    "total": 150,
    "totalPages": 8
  }
}
```

**错误：**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request parameters",
    "details": [
      { "field": "costPrice", "message": "must be positive" }
    ]
  }
}
```


## 分页参数

| 参数 | 默认值 | 范围 |
|------|--------|------|
| page | 1 | >= 1 |
| pageSize | 20 | 1 - 100 |


## JSON 字段命名

API 使用 camelCase（与前端 TypeScript 一致）。Go 结构体通过 json tag 转换：

```go
type Holding struct {
	HoldingID     int64   `json:"holdingId"`
	AssetName     string  `json:"assetName"`
	CostPrice     float64 `json:"costPrice"`
	PositionRatio float64 `json:"positionRatio"`
}
```


## 异步任务

分析任务是耗时操作，使用异步模式：

```
POST /api/v1/analysis/trigger -> 202 { "data": { "taskId": "abc123" } }
GET  /api/v1/tasks/abc123     -> 200 { "data": { "taskId": "...", "status": "running", "progress": 0.6 } }
```

任务状态：`pending` -> `running` -> `completed` / `failed`


## 认证

- 使用 JWT Bearer Token
- 请求头：`Authorization: Bearer <token>`
- Token 过期时间：24 小时（可配置）
- 刷新机制：后续迭代


## 核心 API 端点（MVP）

### Auth
- `POST /api/v1/auth/register` -- 邮箱注册（需邀请码）
- `POST /api/v1/auth/login` -- 邮箱密码登录
- `GET /api/v1/auth/me` -- 获取当前用户信息

### Portfolio
- `GET /api/v1/holdings` -- 持仓列表
- `POST /api/v1/holdings` -- 添加持仓（快速模式）
- `PUT /api/v1/holdings/:id` -- 更新持仓
- `DELETE /api/v1/holdings/:id` -- 删除持仓
- `POST /api/v1/holdings/:id/trades` -- 添加交易记录（明细模式）
- `GET /api/v1/holdings/:id/trades` -- 交易记录列表

### Analysis
- `POST /api/v1/analysis/trigger` -- 触发分析（异步）
- `GET /api/v1/tasks/:taskId` -- 查询任务状态

### Decision Cards
- `GET /api/v1/decision-cards` -- 决策卡列表（最新一轮）
- `GET /api/v1/decision-cards/:id` -- 决策卡详情
- `GET /api/v1/decision-cards/history` -- 历史决策卡

### Notification
- `GET /api/v1/notification/channels` -- 已配置渠道列表
- `POST /api/v1/notification/channels` -- 添加推送渠道
- `PUT /api/v1/notification/channels/:id` -- 更新渠道配置
- `DELETE /api/v1/notification/channels/:id` -- 删除渠道

### Asset Catalog
- `GET /api/v1/assets` -- 标的目录（分类浏览 + 搜索）
- `GET /api/v1/assets/:code` -- 标的详情
