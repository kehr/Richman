# Step 7: Notification System

## 任务目标

实现推送通知中心：统一调度器、可插拔渠道适配器（微信公众号/飞书/邮件）、Cron 定时任务调度（早盘/收盘/美股三个时段）。

## 涉及文件路径

### 创建

- `backend/internal/notification/dispatcher.go` -- 统一调度器 send(userId, message, channels[])
- `backend/internal/notification/dispatcher_test.go`
- `backend/internal/notification/adapter/adapter.go` -- NotificationAdapter 接口定义
- `backend/internal/notification/adapter/wechat/wechat.go` -- 微信公众号模板消息
- `backend/internal/notification/adapter/wechat/wechat_test.go`
- `backend/internal/notification/adapter/feishu/feishu.go` -- 飞书机器人卡片消息
- `backend/internal/notification/adapter/feishu/feishu_test.go`
- `backend/internal/notification/adapter/email/email.go` -- HTML 邮件
- `backend/internal/notification/adapter/email/email_test.go`
- `backend/internal/notification/adapter/email/templates/` -- 邮件 HTML 模板
- `backend/internal/service/notification/service.go` -- 推送业务逻辑
- `backend/internal/service/analysis/scheduler.go` -- Cron 定时任务调度
- `backend/internal/service/analysis/scheduler_test.go`
- `backend/internal/api/v1/notification.go` -- 推送渠道配置路由
- `backend/internal/model/notification_channel.go` -- 推送渠道配置模型
- `backend/internal/model/notification_log.go` -- 推送日志模型
- `backend/db/migration/004_notification.up.sql` -- notification_channels + notification_logs 表
- `backend/db/migration/004_notification.down.sql`
- `backend/db/query/notification_channel.sql`
- `backend/db/query/notification_log.sql`

## PRD/TRD 章节引用

- PRD 3.4 每日推送通知（时区、时段、渠道、架构）
- `docs/standards/api.md` 推送渠道端点
- `docs/standards/logging.md` 推送相关日志点
- `docs/standards/backend.md` 定时任务

## 验证标准

- [ ] NotificationAdapter 接口定义清晰，新增渠道只需实现接口
- [ ] 微信适配器：构造正确的模板消息请求体
- [ ] 飞书适配器：构造正确的卡片消息请求体
- [ ] 邮件适配器：生成正确的 HTML 邮件
- [ ] 调度器：send() 按用户配置的渠道列表分发
- [ ] 调度器：单个渠道失败不影响其他渠道
- [ ] 推送日志正确记录（渠道、状态、错误信息）
- [ ] `POST /api/v1/notification/channels` 添加渠道配置
- [ ] `GET /api/v1/notification/channels` 查询已配置渠道
- [ ] `PUT /api/v1/notification/channels/:id` 更新渠道
- [ ] `DELETE /api/v1/notification/channels/:id` 删除渠道
- [ ] Cron 调度器按 PRD 时段触发分析 + 推送：
  - 08:30 CST: A 股标的
  - 15:30 CST: A 股 + 黄金
  - 06:00 CST (次日): 美股
- [ ] Cron 只对有对应类型持仓的用户触发
- [ ] `go test ./internal/notification/...` 全部通过
- [ ] `go test ./internal/service/analysis/...` 调度器测试通过
- [ ] `golangci-lint run ./...` 零错误
- [ ] `go vet ./...` 零警告

## 依赖说明

- Step 6 完成（分析引擎和决策卡生成就绪）
