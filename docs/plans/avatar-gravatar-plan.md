# Avatar Gravatar Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 接入 Gravatar 为每个用户提供基于邮箱的唯一头像，并在设置页提供预览和引导入口。

**Architecture:** 纯前端实现，零后端改动。新增 `gravatar.ts` 工具函数封装 MD5 hash + URL 拼接逻辑；MainLayout 导航栏头像和 AccountTab 设置页头像区均调用该函数；antd Avatar 的 src + icon 组合天然支持加载失败回退。

**Tech Stack:** React 19, Ant Design 6, blueimp-md5, react-i18next

**设计依据：**
- PRD：`docs/prds/avatar-gravatar-prd.md`
- TRD：`docs/trds/avatar-gravatar-trd.md`

**Worktree 隔离：**
```bash
git worktree add .claude/worktrees/avatar-gravatar -b feat/avatar-gravatar
```

---

## Step 1: 安装依赖

**目标：** 添加 blueimp-md5 到 frontend 依赖

**涉及文件：**
- 修改：`frontend/package.json`（pnpm 自动更新）

**设计依据：** TRD §依赖安装

- [ ] 在 frontend 目录执行：
  ```bash
  cd frontend && pnpm add blueimp-md5 && pnpm add -D @types/blueimp-md5
  ```
- [ ] 验证 `package.json` 中出现 `blueimp-md5` 依赖
- [ ] 执行 `pnpm lint:all` 确认无新错误
- [ ] 提交：
  ```bash
  git add frontend/package.json frontend/pnpm-lock.yaml
  git commit -m "chore(deps): add blueimp-md5 for Gravatar URL generation"
  ```

---

## Step 2: 创建 gravatar.ts 工具函数

**目标：** 封装 Gravatar URL 生成逻辑，并添加单元测试

**涉及文件：**
- 新增：`frontend/src/domain/auth/gravatar.ts`
- 新增：`frontend/src/domain/auth/gravatar.test.ts`

**设计依据：** TRD §gravatar.ts、§Avatar 回退机制

- [ ] 创建 `src/domain/auth/gravatar.ts`，实现 `gravatarUrl(email, size?)` 函数（接口见 TRD）
- [ ] 创建 `src/domain/auth/gravatar.test.ts`，测试以下纯逻辑：
  - 空 email 返回空字符串
  - email 大小写和前后空格被规范化（同 email 不同格式得到相同 URL）
  - 返回 URL 包含 `d=identicon`、`r=g`、正确的 size 参数
  - 默认 size 为 32
- [ ] 执行 `pnpm test` 确认测试通过（无需 UI 测试，只测纯函数）
- [ ] 执行 `pnpm lint:all` 通过
- [ ] 提交：
  ```bash
  git add frontend/src/domain/auth/gravatar.ts frontend/src/domain/auth/gravatar.test.ts
  git commit -m "feat(auth): add gravatarUrl utility with tests"
  ```

---

## Step 3: 更新 i18n 翻译文件

**目标：** 添加头像区翻译 key，中英文同步

**涉及文件：**
- 修改：`frontend/src/i18n/locales/zh/settings.json`
- 修改：`frontend/src/i18n/locales/en/settings.json`

**设计依据：** PRD §i18n Keys、TRD §i18n

- [ ] 在 zh/settings.json 的 `account` 对象下新增（见 TRD §i18n 中的具体 key 和值）
- [ ] 在 en/settings.json 的 `account` 对象下新增（见 TRD §i18n 中的具体 key 和值）
- [ ] 验证两个文件中新 key 的路径和值完全对应，无遗漏
- [ ] 执行 `pnpm lint:all` 通过
- [ ] 提交：
  ```bash
  git add frontend/src/i18n/locales/zh/settings.json frontend/src/i18n/locales/en/settings.json
  git commit -m "feat(i18n): add avatar section translation keys"
  ```

---

## Step 4: 更新 MainLayout 导航栏头像

**目标：** 导航栏 Avatar 从图标占位改为 Gravatar 头像图片

**涉及文件：**
- 修改：`frontend/src/layouts/MainLayout.tsx`

**设计依据：** TRD §MainLayout 改动

- [ ] 在 `MainLayout.tsx` 中导入 `gravatarUrl`
- [ ] 将 `<Avatar icon={<UserOutlined />} />` 替换为带 `src` 的形式（见 TRD §MainLayout 改动）
- [ ] 确认 `email` 来源：`user?.email ?? ""`（user 来自 `useCurrentUser()` 已有）
- [ ] 执行 `pnpm lint:all` 通过
- [ ] 人眼验收：启动 dev server，确认导航栏头像显示 identicon 图案
- [ ] 提交：
  ```bash
  git add frontend/src/layouts/MainLayout.tsx
  git commit -m "feat(layout): use Gravatar identicon in top nav avatar"
  ```

---

## Step 5: 更新 AccountTab 设置页头像区

**目标：** 在设置页账户 Tab 顶部添加头像预览区，吸收原邮箱展示行

**涉及文件：**
- 修改：`frontend/src/pages/settings/tabs/AccountTab.tsx`

**设计依据：** TRD §AccountTab 改动、PRD §AccountTab 头像区

- [ ] 在 `AccountTab.tsx` 中导入 `gravatarUrl` 和 `Avatar`（Avatar 从 `@/ui-kit/eat` 导入）
- [ ] 在组件顶部计算 `displayName`：`email.split("@")[0] || "—"`
- [ ] 用 TRD §AccountTab 改动中定义的头像区结构替换原有邮箱区（Flex + Avatar 64px + displayName + email + Gravatar 外链）
- [ ] 改密按钮保留在头像区下方（见 TRD §改密按钮保留位置）
- [ ] 确认 `Typography.Link` 使用 `target="_blank"` 和 `rel="noopener noreferrer"`
- [ ] 执行 `pnpm lint:all` 通过
- [ ] 人眼验收：设置页账户 Tab 显示 64px identicon + 用户名 + 邮箱 + Gravatar 外链，点击外链在新 Tab 打开 gravatar.com
- [ ] 提交：
  ```bash
  git add frontend/src/pages/settings/tabs/AccountTab.tsx
  git commit -m "feat(settings): add avatar preview section with Gravatar link"
  ```

---

## 验收清单

- [ ] 所有用户（包括未注册 Gravatar 的）导航栏显示 identicon 头像
- [ ] 网络不可达时头像回退到 UserOutlined 图标
- [ ] 设置页账户 Tab 顶部显示 64px 头像 + 用户信息 + Gravatar 引导链接
- [ ] 原邮箱行不重复展示
- [ ] `pnpm lint:all` 全部通过
- [ ] 中英文切换后头像区文案正确切换
