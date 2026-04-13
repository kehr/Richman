# Step 10: richman Email Push System

> Phase 3 | 并行组 R5 (可与 Step 8, 9 同时执行) | 前置: Step 7

## 任务目标

实现 richman v2 邮件推送系统：EmailPushService（每日简报/每周洞察/市场快讯/持仓建议 4 种邮件）、HTML 邮件模板引擎（Go html/template）、8 套 i18n 模板文件（中/英 x 4 类型）、邮件发送器（BCC 50 人分批）。

## 涉及文件

### 创建

**Service：**
- `backend/internal/service/emailpush/service.go` -- EmailPushService

**模板引擎：**
- `backend/internal/service/emailpush/template/engine.go` -- 模板加载 + 渲染

**HTML 模板（8 个文件）：**
- `backend/internal/service/emailpush/template/daily_briefing_zh.html`
- `backend/internal/service/emailpush/template/daily_briefing_en.html`
- `backend/internal/service/emailpush/template/weekly_insight_zh.html`
- `backend/internal/service/emailpush/template/weekly_insight_en.html`
- `backend/internal/service/emailpush/template/market_alert_zh.html`
- `backend/internal/service/emailpush/template/market_alert_en.html`
- `backend/internal/service/emailpush/template/holding_suggestion_zh.html`
- `backend/internal/service/emailpush/template/holding_suggestion_en.html`

**邮件发送器：**
- `backend/internal/service/emailpush/sender.go` -- SMTP Sender + SendBatch

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| EmailPushService 4 个方法 | SS10 通知推送 | richman SS7.2 |
| 每日简报内容组装 | SS10.3 推送场景 | richman SS7.5 |
| 每周洞察内容 | SS10.3 | richman SS7.6 |
| 市场快讯 | SS10.3 | richman SS7.2 |
| 持仓建议 | SS10.3 | richman SS7.2 |
| 推送频率控制 (每日 3 次) | SS10.1 | richman SS7.7 |
| HTML 模板设计原则 | - | richman SS7.4 |
| BCC 50 人分批 | - | richman SS7.3 |
| 退订机制 (email_push_enabled) | SS10.2 | richman SS7.4.1 |
| 游标分页 (大用户量) | - | richman SS7.5 |
| 模板 i18n | - | richman SS7.4 |

## 关键约束

- HTML 邮件必须：内联 CSS、表格布局、单列 600px、暗色模式基础支持
- 每封邮件底部包含：退订链接（/settings）+ 免责声明
- SendBatch 按 50 人 BCC 分批，批间间隔 1 秒
- 推送频率：每用户每日最多 3 次，通过 rm_notification_logs 统计
- 每日简报使用游标分页（200 用户/页），避免大量用户时内存溢出
- 查询用户时过滤 `email_push_enabled = TRUE`
- 周报 richson 调用失败时跳过（不降级生成），记录 ERROR 日志
- 邮件 Subject 跟随用户 locale

## 验证标准

- [ ] `cd backend && make check` 通过
- [ ] 模板引擎可加载全部 8 个模板文件
- [ ] 每个模板给定 mock 数据能正确渲染 HTML
- [ ] HTML 输出包含退订链接和免责声明
- [ ] SendBatch 对 120 个收件人正确分为 3 批
- [ ] 推送频率控制：第 4 次推送被跳过
- [ ] EmailPushService 4 个方法签名正确

## 变更点清单覆盖

D3.6-D3.9 (4), D9.1-D9.10 (10) = **14 项**
