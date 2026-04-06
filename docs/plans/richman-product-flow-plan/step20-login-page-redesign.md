# Step 20 LoginPage 左右双栏改造

## 任务目标

把现有的 LoginPage 从单列简版改为 PRD §2.1 的左右双栏布局：左侧产品介绍 + 样例决策卡截图，右侧登录表单。同时为推送链接回流补全 ?returnTo= 处理。

## 涉及文件

修改：
- `frontend/src/pages/auth/LoginPage.tsx`
- `frontend/src/pages/auth/RegisterPage.tsx`（顺手对齐布局风格 + 邀请码字段无变化）
- `frontend/src/features/auth/use-login.ts`（如不存在则创建，处理 returnTo 回流逻辑）

创建：
- `frontend/src/pages/auth/components/AuthSplitLayout.tsx`（左右双栏布局组件，复用给 Login 和 Register）
- `frontend/src/pages/auth/components/SampleDecisionCard.tsx`（左侧用的样例卡截图，纯静态展示）
- `frontend/src/pages/auth/LoginPage.test.tsx`

## 设计依据

- PRD §2.1 登录页 / §2.2 注册页规格
- TRD §7 前端工程改造概述

## 实施要点

- AuthSplitLayout 布局：
  - 桌面 ≥ 1200px：左 60% / 右 40% 双栏
  - 1024-1199px：左右各 50%
  - < 1024px：折叠为单列，左侧内容放表单上方
- 左侧固定内容：
  - 产品名 "Richman" + 标语 "把基金经理的思维方式装进你的口袋"
  - 三行简介
  - 一张静态 SampleDecisionCard 组件（不调 API，用硬编码数据展示三维 badge + 建议 + 信心度，仅用于视觉示例）
  - 不要在登录页放完整营销页内容，保持极简
- 右侧表单：
  - 邮箱 + 密码 + 登录按钮
  - 底部"还没账号？用邀请码注册"链接跳 /register
- returnTo 处理：
  - useLocation 解析 ?returnTo=...
  - 登录成功后 navigate(returnTo || '/dashboard')
  - returnTo 必须是相对路径（防 open redirect）；不是相对路径时忽略并跳 /dashboard
- RegisterPage 用同一个 AuthSplitLayout，左侧内容相同，右侧改为注册表单（邮箱 + 密码 + 邀请码）
- 所有 antd 经 ui-kit/eat barrel

## 验证标准

1. `pnpm test src/pages/auth` 通过
2. 浏览器手动测：
   - 桌面看到左右双栏，左侧样例卡视觉清晰
   - 缩小窗口到平板 / 手机宽度看到响应式折叠
   - 登录后无 returnTo 参数跳 /dashboard
   - 登录后有合法 returnTo 跳到对应路径
   - returnTo 是 https://evil.com 时被忽略
3. `pnpm lint:all` 通过

## 依赖说明

- 无前置硬依赖，可与 step13-19 的任意 step 并行

## 预估提交

- commit 1: `feat(auth): add split layout with sample decision card`
- commit 2: `refactor(auth): handle returnTo on login success`
