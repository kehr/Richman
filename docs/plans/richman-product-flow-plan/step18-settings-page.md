# Step 18 Settings 页面 4 个 Tab

## 任务目标

把现有的 SettingsPage（占位"coming soon"）重写为 PRD §6 的 4 tab 结构：账户 / 推送渠道 / 偏好 / 订阅与额度。同时把原 NotificationsPage 的渠道配置功能迁入 Tab 2。

## 涉及文件

修改：
- `frontend/src/pages/settings/SettingsPage.tsx`

创建：
- `frontend/src/pages/settings/components/SettingsTabsLayout.tsx`（左 tab 栏 + 右内容区）
- `frontend/src/pages/settings/tabs/AccountTab.tsx`
- `frontend/src/pages/settings/tabs/AccountTab.test.tsx`
- `frontend/src/pages/settings/tabs/ChannelsTab.tsx`
- `frontend/src/pages/settings/tabs/PreferencesTab.tsx`
- `frontend/src/pages/settings/tabs/SubscriptionTab.tsx`
- `frontend/src/features/notification-channels/`（从原 features/notification 重命名 / 提取，仅保留与渠道配置相关的 hooks 和组件）
  - `api.ts` `use-channels.ts` `components/ChannelList.tsx` `components/AddChannelDrawer.tsx` `components/ChannelTestButton.tsx` `index.ts`

修改：
- `frontend/src/routes.tsx`（如需 ?tab=account 这样的 query 路由参数支持）

## 设计依据

- PRD §6 Settings 设置页 4 tab 完整规格
- TRD §5.4 risk_preference 字段
- TRD §6.1 dev 环境"重置 Onboarding"按钮
- TRD §6.2 帮助页锚点链接
- 工程规范 features/notification-channels 隔离

## 实施要点

- SettingsTabsLayout：
  - 左侧 200px tab 栏 + 右侧内容区
  - 当前 tab 左侧 3px 黑边 + 浅灰底
  - tab 切换通过 URL query 参数（?tab=account 等）以便外链定位
- AccountTab：
  - 邮箱（只读）
  - 修改密码按钮（发送邮件链接，调现有 auth API）
  - 总资金输入框（数字 + 货币 CNY 标签 + 保存按钮 + 隐私提示小字 "总资金仅本地保存用于金额换算..."）
  - 风险偏好下拉（稳健 / 中性 / 激进）+ 说明小字 "影响 LLM 在权重微调范围内的倾向"
  - 退出登录红色次按钮
  - 仅 dev 环境显示"重置 Onboarding"按钮，调 DELETE /api/v1/onboarding
- ChannelsTab：
  - 接入 features/notification-channels 的组件
  - 顶部"当前已启用 N 个渠道"
  - 渠道列表（类型图标 + 摘要 + 启用开关 + 测试发送 + 删除）
  - "+ 添加渠道"按钮打开 AddChannelDrawer，按渠道类型显示对应表单
  - 底部推送时段说明 + 链接 /help#push
- PreferencesTab：
  - 语言（中文 / English）单选
  - 时区下拉
  - 主题（固定亮色，灰字"MVP 暂不支持暗色"）
  - 数字格式可折叠高级选项
- SubscriptionTab：
  - 当前订阅徽章（invite）
  - 额度使用网格（持仓数 / 每日分析次数 / 渠道数 / LLM 模型）
  - "申请升级"按钮置灰显示"敬请期待"
- 所有写操作通过 user_settings PATCH 接口，写入后 invalidate user-settings query

## 验证标准

1. `pnpm test src/pages/settings` 通过
2. 浏览器手动测：
   - 4 个 tab 切换正常，URL query 同步
   - AccountTab 修改总资金后 Dashboard 顶部 strip 立即出现金额（依赖 query invalidate）
   - ChannelsTab 添加 / 删除 / 测试渠道
   - dev 环境重置 Onboarding 按钮可以触发引导流程重新出现
3. 删除原 NotificationsPage 后无 dead import
4. `pnpm lint:all` 通过

## 依赖说明

- 前置：step10 删除 NotificationsPage、step11 useUserSettings hook、step09 user_settings API、step08 onboarding 重置 API

## 预估提交

- commit 1: `feat(settings): add tabs layout and account tab`
- commit 2: `feat(settings): migrate notification channels into channels tab`
- commit 3: `feat(settings): add preferences and subscription tabs`
