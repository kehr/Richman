# Step 5: API Handlers + Routes

**依赖：** Step 3（service 层）
**可与 Step 6/7/8 并行**
**设计依据：** TRD §API 设计、§后端层级结构

## 任务目标

实现四个调度配置 API 端点并注册路由。

## 涉及文件

- 创建：`backend/internal/handlers/schedule/get_settings.go`
- 创建：`backend/internal/handlers/schedule/update_settings.go`
- 创建：`backend/internal/handlers/schedule/get_holding_schedule.go`
- 创建：`backend/internal/handlers/schedule/update_holding_schedule.go`
- 修改：`backend/cmd/server/main.go`（注册路由）

## 执行步骤

- [ ] 查看现有 handler 文件（如 `handlers/settings/` 任一文件），参照：DTO 定义方式、auth 中间件获取 userID 的方法、统一错误响应格式
- [ ] 创建 `get_settings.go`：`GET /api/v1/settings/schedule`，调用 `ScheduleService.GetUserScheduleSettings`，无记录时返回系统默认值；响应结构参照 TRD §API 设计中的 Response body
- [ ] 创建 `update_settings.go`：`PUT /api/v1/settings/schedule`，绑定 JSON body，校验字段（参照 TRD PUT 校验规则），调用 `UpsertUserScheduleSettings`；成功后调用 `scheduler.ReloadUser(userID)` 重载调度
- [ ] 创建 `get_holding_schedule.go`：`GET /api/v1/holdings/:id/schedule`，验证 holding 属于当前用户，调用 `GetHoldingScheduleOverride` + `ComputeNextAnalysisAt`，组装响应（含 `nextAnalysisAt` 字段）
- [ ] 创建 `update_holding_schedule.go`：`PUT /api/v1/holdings/:id/schedule`，校验 `window` 枚举值（pre/post/both/null），upsert 覆盖后返回更新结果含新的 `nextAnalysisAt`
- [ ] 修改 `main.go`，在 `/api/v1` 路由组中注册四个端点（参照现有注册模式，约 321-334 行）
- [ ] `cd backend && make check` 验证通过
- [ ] `git add backend/internal/handlers/schedule/ backend/cmd/server/main.go && git commit -m "feat(api): add schedule settings and holding schedule endpoints"`

## 验证标准

- `make check` 通过
- 四个路由在 main.go 中正确注册
- GET /settings/schedule 对无配置用户返回默认值（不写库）
- PUT /settings/schedule 非法频率值返回 400
