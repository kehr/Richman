# Step 3: Schedule Service Layer

**依赖：** Step 2（sqlc 代码已生成）
**设计依据：** TRD §后端层级结构、§API 设计

## 任务目标

实现调度配置的 CRUD 服务层，包含三层优先级的 `nextAnalysisAt` 计算逻辑。

## 涉及文件

- 创建：`backend/internal/service/schedule/service.go`
- 创建：`backend/internal/service/schedule/defaults.go`（系统默认值常量）
- 创建：`backend/internal/service/schedule/next_analysis.go`（下次分析时间计算）

## 执行步骤

- [ ] 查看现有 service 层结构：`ls backend/internal/service/` 和任一 service.go 文件，参照构造函数和接口风格
- [ ] 创建 `defaults.go`，定义系统默认调度设置的常量值：
  - A 股盘前默认 08:30、盘后 15:05
  - 美股夏令盘前 20:30/盘后 04:05，冬令 21:30/05:05
  - 全局默认频率 `daily`
- [ ] 创建 `service.go`，实现：
  - `ScheduleService` 结构体，依赖 repo 层查询函数
  - `GetUserScheduleSettings(ctx, userID)` — 查库，若无记录返回系统默认值（不写库）
  - `UpsertUserScheduleSettings(ctx, userID, input)` — 校验后 upsert，返回更新后完整记录
  - `GetHoldingScheduleOverride(ctx, userID, holdingID)` — 查库，无记录返回 nil
  - `UpsertHoldingScheduleOverride(ctx, userID, holdingID, input)` — upsert
  - 校验规则参照 TRD §API 设计的 PUT 校验描述
- [ ] 创建 `next_analysis.go`，实现 `ComputeNextAnalysisAt(userID, holdingID, market, now)` ：
  - 优先级：持仓覆盖 > 市场设置 > 全局默认（三层合并）
  - 根据合并后的 frequency + 窗口时间，从 now 起找到下一个满足间隔的触发时间点
  - 需要读取该持仓的 `last_analyzed_at`（从 holdings 表或 decision_cards 表中获取，调研现有字段）
- [ ] 执行 `cd backend && make check` 验证 lint + build 通过
- [ ] `git add backend/internal/service/schedule/ && git commit -m "feat(service): add schedule service with CRUD and next analysis computation"`

## 验证标准

- `make check` 无 lint 错误，无编译错误
- `GetUserScheduleSettings` 对无配置用户返回系统默认值
- `ComputeNextAnalysisAt` 三层优先级逻辑通过逻辑走读验证（无 UI 测试）
