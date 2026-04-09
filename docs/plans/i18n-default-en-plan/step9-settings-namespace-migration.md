# Step 9: settings namespace 迁移

## 任务目标

将 settings 页面组件（PreferencesTab、AccountTab、SubscriptionTab、ChannelsTab）、LLM 配置组件、通知渠道组件中的所有硬编码中文替换为 `t("settings:...")` 调用。同时将 PreferencesTab 从旧 useLocale 迁移到 useTranslation。

## 涉及文件

Settings 页面 tabs：
- 修改: `frontend/src/pages/settings/SettingsPage.tsx`
- 修改: `frontend/src/pages/settings/tabs/PreferencesTab.tsx`
- 修改: `frontend/src/pages/settings/tabs/AccountTab.tsx`
- 修改: `frontend/src/pages/settings/tabs/SubscriptionTab.tsx`
- 修改: `frontend/src/pages/settings/tabs/ChannelsTab.tsx`

LLM 配置组件：
- 修改: `frontend/src/features/settings-llm/LLMSection.tsx`
- 修改: `frontend/src/features/settings-llm/LLMConfigForm.tsx`
- 修改: `frontend/src/features/settings-llm/LLMHealthyCard.tsx`
- 修改: `frontend/src/features/settings-llm/LLMFailingCard.tsx`
- 修改: `frontend/src/features/settings-llm/LLMProbeButton.tsx`
- 修改: `frontend/src/features/settings-llm/LLMEmptyState.tsx`

通知渠道组件：
- 修改: `frontend/src/features/notification-channels/components/AddChannelDrawer.tsx`
- 修改: `frontend/src/features/notification-channels/components/ChannelList.tsx`
- 修改: `frontend/src/features/notification-channels/components/ChannelTestButton.tsx`

## PRD/TRD 引用

- PRD §5.5.1（PreferencesTab Radio 绑定 i18n.language + changeLanguage）
- TRD §8（PreferencesTab 迁移：删 useLocale → useTranslation，Radio 选项硬写）
- TRD §12（字符串迁移约定）
- TRD §12.3（Form rule message 约定）

## 验证标准

- [ ] `pnpm lint:all` 通过
- [ ] `pnpm test` 通过
- [ ] `rg '[\u4e00-\u9fff]' frontend/src/pages/settings frontend/src/features/settings-llm frontend/src/features/notification-channels --type tsx` 结果只剩 Radio 选项 label "中文"
- [ ] PreferencesTab 不再 import `useLocale`，改用 `useTranslation`
- [ ] `pnpm dev` 启动后 Settings 页面默认英文
- [ ] 切中文后 Settings 所有 tab（Preferences / Account / Subscription / Channels）+ LLM 配置 + 通知渠道全中文
- [ ] PreferencesTab 的语言 Radio 切换行为正常（切语言 → 整页即时变化 → 刷新持久）

## 依赖

- Step 3（settings namespace JSON 已就绪）

## 实施注意

- PreferencesTab 是旧 useLocale 的 3 个消费者之一，本 step 完成后旧 provider 的消费者减少到 2（HelpPage + HelpPage.test）
- LLMConfigForm 有 25 处中文，是 settings 最密集的组件，包含大量 Form.Item label / placeholder / tooltip / validation
- AccountTab 有 24 处，包含个人信息表单、密码修改表单
- ChannelList 有 17 处，包含渠道类型名称、状态标签、操作按钮
- AddChannelDrawer 有 19 处，大量 Form 表单
- SettingsPage 有 6 处，是 Tabs 标签名
- Radio 选项 label（"中文" / "English"）保持硬写不走 t()（TRD §8.2）
