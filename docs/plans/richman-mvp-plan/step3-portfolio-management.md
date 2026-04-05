# Step 3: Asset Catalog + Portfolio Management

## 任务目标

实现标的目录 API（分类浏览 + 搜索）、持仓 CRUD API（快速模式 + 明细模式）、交易记录 API、持仓成本自动计算逻辑。

## 涉及文件路径

### 创建

- `backend/internal/model/asset.go` -- 标的目录模型
- `backend/internal/model/holding.go` -- 持仓模型
- `backend/internal/model/trade.go` -- 交易记录模型
- `backend/internal/api/v1/asset_catalog.go` -- 标的目录路由
- `backend/internal/api/v1/portfolio.go` -- 持仓路由
- `backend/internal/service/portfolio/service.go` -- 持仓业务逻辑（含成本计算）
- `backend/internal/service/portfolio/cost_calculator.go` -- 成本计算核心逻辑
- `backend/internal/service/portfolio/cost_calculator_test.go` -- 成本计算测试
- `backend/db/migration/002_portfolio.up.sql` -- 标的目录 + 持仓 + 交易记录表
- `backend/db/migration/002_portfolio.down.sql`
- `backend/db/query/asset_catalog.sql` -- 标的查询
- `backend/db/query/holding.sql` -- 持仓查询
- `backend/db/query/trade.sql` -- 交易记录查询
- `backend/db/seed/asset_catalog.sql` -- MVP 标的种子数据（30-50 个精选标的）

## PRD/TRD 章节引用

- PRD 2.1-2.4 支持标的类型
- PRD 3.1 持仓管理（快速模式 + 明细模式、成本计算、限制 5 个）
- `docs/standards/database.md` 表设计约定
- `docs/standards/api.md` 持仓 + 标的端点定义

## 验证标准

- [ ] `GET /api/v1/assets` 返回标的列表，支持 type 筛选和 keyword 搜索
- [ ] `GET /api/v1/assets/:code` 返回单个标的详情
- [ ] `POST /api/v1/holdings` 快速模式创建持仓成功
- [ ] `GET /api/v1/holdings` 返回当前用户持仓列表
- [ ] `PUT /api/v1/holdings/:id` 更新持仓成功
- [ ] `DELETE /api/v1/holdings/:id` 软删除持仓成功
- [ ] 第 6 个持仓创建被拒绝（限制 5 个）
- [ ] `POST /api/v1/holdings/:id/trades` 明细模式添加交易记录
- [ ] `GET /api/v1/holdings/:id/trades` 返回交易记录列表
- [ ] 添加交易记录后，综合成本自动重新计算
- [ ] 快速模式成本 + 明细模式记录的混合计算正确
- [ ] `go test ./internal/service/portfolio/...` 成本计算测试全部通过

## 依赖说明

- Step 2 完成（数据库迁移工具、认证中间件就绪）
