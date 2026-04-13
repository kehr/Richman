# Step 18: Settings + Invite + Auth Changes

> Phase 4 | 并行组 R9 (可与 Step 15, 16, 17 同时执行) | 前置: Step 14

## 任务目标

扩展设置页：邀请码区块（InviteSection + 码列表 + 解锁进度 + 已邀请列表）、邮件推送开关（EmailPushToggle）、账户注销入口（AccountDeletionSection），新增风险偏好子页面，以及 invite feature 模块、auth 注册表单增强（disclaimerAccepted + ref 参数）、user-settings 扩展（riskPreference + emailPush hooks）、剩余 i18n 翻译。

## 涉及文件

### 创建

**Feature 模块：**
- `frontend/src/features/invite/api.ts`
- `frontend/src/features/invite/types.ts`
- `frontend/src/features/invite/use-my-codes.ts`
- `frontend/src/features/invite/use-my-invites.ts`
- `frontend/src/features/invite/index.ts`

**页面 + 组件：**
- `frontend/src/pages/settings/components/invite-section.tsx`
- `frontend/src/pages/settings/components/email-push-toggle.tsx`
- `frontend/src/pages/settings/components/account-deletion-section.tsx`
- `frontend/src/pages/settings/risk-preference-sub-page.tsx` (或在 settings/tabs/ 下)

### 修改

- `frontend/src/features/user-settings/` -- 新增 riskPreference 字段 + usePatchRiskPreference + usePatchEmailPush hooks
- `frontend/src/features/auth/` -- 注册表单增加 disclaimerAccepted checkbox + ref 参数自动填充
- `frontend/src/pages/settings/settings-page.tsx` -- 新增三个区块
- `frontend/src/pages/auth/components/` -- RegisterForm 修改
- `frontend/src/i18n/locales/zh/settings.json` -- 新增 invite.* / emailPush.* / deleteAccount.* / riskPreference.* key
- `frontend/src/i18n/locales/en/settings.json`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| InviteSection 结构 | SS14.3 邀请裂变 | invite SS7.1 / frontend SS3.6 |
| EmailPushToggle | SS10.2 退订 | frontend SS3.6 |
| AccountDeletionSection | SS13 | frontend SS3.6 |
| RiskPreferenceSubPage | SS7.6 风险偏好 | frontend SS2.1 |
| invite feature 模块 | SS14.3 | frontend SS3.6 / invite SS10 |
| 注册表单 disclaimer | SS13 免责声明 | frontend SS3.6 |
| 注册表单 ref 参数 | SS14.3 分享链接 | invite SS6.2 |
| usePatchRiskPreference | SS7.6 | frontend SS3.6 |
| 邮件 CTA 链接目标 | SS10 | frontend SS16.10 |
| i18n namespace 拆分 | - | frontend SS16.4 |

## 关键约束 + 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G3.4 i18n namespace 拆分 | 评估 500 key 阈值，超过则拆分 | frontend SS16.4 |
| G3.10 邮件 CTA 链接 | FRONTEND_BASE_URL + route mapping | frontend SS16.10 |
| G4.1-G4.3 | 前端侧无需额外处理（后端已处理） | invite SS11 |

- 邀请码列表可复制（使用 navigator.clipboard.writeText）
- 解锁进度："再连续登录 X 天解锁新邀请码"（nextUnlockIn 字段）
- 账户注销需密码确认（调用 DELETE /api/v1/auth/account）
- ref 参数自动填充：`useSearchParams().get("ref")` -> 邀请码输入框
- disclaimerAccepted 必须勾选才能提交注册
- 风险偏好子页面：三个卡片选择（conservative/moderate/aggressive）
- EmailPushToggle 调用 PATCH /api/v2/user/email-push
- i18n 如果 settings.json 超 500 key，拆分为 settings-invite.json 等子文件

## 验证标准

- [ ] `cd frontend && pnpm lint:all` 全部通过
- [ ] `pnpm build` 成功
- [ ] 设置页显示邀请码列表 + 解锁进度
- [ ] 邀请码可复制到剪贴板
- [ ] 邮件推送开关切换正常
- [ ] 账户注销弹窗需输入密码
- [ ] /settings/risk-preference 页面三型选择正常
- [ ] 注册页 disclaimer checkbox 必须勾选
- [ ] 注册页 URL ref 参数自动填充邀请码
- [ ] 全部 i18n key 在 zh + en 两个 locale 文件中同步存在

## 变更点清单覆盖

E3.4 (1), E1.6 (1), E2.3-E2.5 (3), E8.1-E8.3 (3), E11.4-E11.7 (4), G3.4 (1), G3.10 (1), G4.1-G4.3 (3) = **17 项**
