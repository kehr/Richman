# Step 4: GET /tasks/:taskId 端点 + 后端验收

**设计依据：** TRD § 1.4（Handler 签名、路由注册）；PRD § 5.2（响应结构）

**依赖：** Step 2（TaskStore.Get 已可返回完整数据）

## 任务目标

新增 `GET /api/v1/analysis/tasks/:taskId` 端点，Handler 从 TaskStore 读取任务状态并序列化返回；在路由文件中注册；完成后对后端整体做 build + vet + test 验收。

## 涉及文件

- 修改：`backend/internal/api/v1/analysis.go`
- 修改：`backend/internal/api/router.go`（或项目中实际路由注册位置）

## 实施内容

**Handler（`analysis.go`）：**

参照 TRD § 1.4 的 `GetTask` 方法：
- 从路由参数取 `taskId`
- 从 JWT middleware 取 `userID`
- 调用 `h.taskStore.Get(taskID)`，若 nil 或 UserID 不匹配返回 404
- 返回 `200 { "data": task }`

**路由注册（`router.go`）：**

在现有 `analysis` 路由组中添加：
```
GET /tasks/:taskId → analysisHandler.GetTask
```

注意确认路由组前缀，保持与现有端点风格一致（`/api/v1/analysis/...`）。

## 验证标准

- `cd backend && go build ./...` 无报错
- `go vet ./...` 无警告
- `make check` 全部通过（或等效的 lint + test + build 命令）
- 手动 curl 或使用 httpie 确认端点存在（dev server 启动后）：
  - 触发分析后取得 taskId，`GET /api/v1/analysis/tasks/{taskId}` 返回正确的 JSON 结构

## 提交

```
feat(backend): add GET /analysis/tasks/:taskId endpoint
```
