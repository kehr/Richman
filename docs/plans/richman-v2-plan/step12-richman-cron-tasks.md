# Step 12: richman Cron Tasks

> Phase 3 | 并行组 R6 (可与 Step 11 同时执行) | 前置: Steps 8, 9, 10

## 任务目标

实现 richman v2 全部定时任务：每日标的分析触发、每日持仓分析、每日简报邮件、A 股收盘快讯、评分变化市场快讯、每周投研洞察、事件告警轮询、过期 Job 清理、richson 健康检查，以及推送频率控制和 Cron 互斥锁。

## 涉及文件

### 创建/修改

- `backend/internal/service/schedule/v2_cron.go` -- RegisterV2CronJobs + 全部任务实现

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| Cron 任务总览 (9 个) | SS10.1 推送时机 | richman SS8.1 |
| 每日标的分析 (06:00) | SS3.1 平台预计算 | richman SS8.3 |
| 每日持仓分析 (07:30) | SS10.3 持仓建议 | richman SS8.3.1 |
| 每日简报邮件 (08:30) | SS10.3 每日简报 | richman SS8.1 |
| A 股收盘快讯 (15:30) | SS10.1 三窗口 | richman SS8.3.2 |
| 评分变化市场快讯 | SS10.3 | richman SS8.3.3 |
| 每周投研洞察 (周一 08:30) | SS10.3 | richman SS8.1 |
| 事件告警轮询 (每小时) | SS3.6 | richman SS8.4 |
| 过期 Job 清理 (每 10 分钟) | - | richman SS8.5 |
| richson 健康检查 (每 30 秒) | - | richman SS3.6 |
| 推送频率控制 (每日 3 次) | SS10.1 | richman SS7.7 |
| Cron 互斥锁 | - | richman SS8.7 |
| 持仓分析并发控制 | - | richman SS8.3.1 |
| 数据新鲜度保障 (08:30 检查) | - | richman SS8.3 |
| 15:30 快讯去重 | - | richman SS8.3.2 + SS22.14 |

## 关键约束

- 全部 cron 时间使用 UTC（对齐 robfig/cron 默认行为），注释标明 UTC+8 对应时间
- 持仓分析并发：不同用户并行（goroutine pool 限 5），同用户串行
- 过期 Job 清理是 richman 对 rs_* 表的跨服务写入例外（仅更新 status/error 列）
- 互斥锁使用 sync.Mutex + TryLock，失败跳过并 WARN 日志
- richson 健康检查更新 Client.IsHealthy()（atomic.Bool）
- 08:30 简报检查 analyzed_at 是否在当日 06:00 后，否则 WARN 但不阻塞
- 15:30 快讯跳过已在 06:00 推送过的标的（查 rm_notification_logs 去重）
- score alert (>=10 分) 与 market alert 去重边界（已知问题 SS22.14）
- v1 cron 任务保持不变，v2 cron 在同一调度器中注册

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] RegisterV2CronJobs 注册全部 9 个 cron 表达式
- [ ] 互斥锁测试：同时触发两次同一任务，第二次跳过
- [ ] 健康检查每 30 秒调用 richsonClient.HealthCheck()
- [ ] 过期 Job 清理 SQL 仅更新 pending/running 状态的过期记录
- [ ] 推送频率控制：第 4 次推送被跳过

## 变更点清单覆盖

D8.1-D8.12 (12), G2.14 (1) = **13 项**
