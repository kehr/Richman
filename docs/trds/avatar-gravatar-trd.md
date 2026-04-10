# 头像功能 TRD — Gravatar 接入

## 依赖 PRD

`docs/prds/avatar-gravatar-prd.md`

## 依赖安装

```bash
pnpm add blueimp-md5
pnpm add -D @types/blueimp-md5
```

`blueimp-md5`：纯 JS MD5，2KB gzip，无依赖，有 TypeScript 类型。

## gravatar.ts

路径：`src/domain/auth/gravatar.ts`

```typescript
import md5 from "blueimp-md5";

// gravatarUrl returns the Gravatar avatar URL for the given email.
// An empty email returns "" — antd Avatar treats empty src as load failure
// and automatically falls back to the icon prop.
export function gravatarUrl(email: string, size = 32): string {
  if (!email) return "";
  const hash = md5(email.trim().toLowerCase());
  return `https://www.gravatar.com/avatar/${hash}?d=identicon&s=${size}&r=g`;
}
```

参数规范：
- `email`：原始邮箱字符串，函数内部 trim + toLowerCase，调用方无需预处理
- `size`：像素值，默认 32；导航栏传 32，设置页传 64

## Avatar 回退机制

antd Avatar 的 `src` + `icon` 组合天然支持回退，无需额外 `onError` 逻辑：
- `src` 加载成功 → 显示图片
- `src` 加载失败（网络问题、域名不通）→ 自动降级到 `icon`

用法：
```tsx
<Avatar src={gravatarUrl(email, 32)} icon={<UserOutlined />} />
```

## MainLayout 改动

文件：`src/layouts/MainLayout.tsx`

`email` 已通过 `useCurrentUser()` 取得（`user?.email ?? ""`）。

改动：将
```tsx
<Avatar icon={<UserOutlined />} />
```
替换为
```tsx
<Avatar src={gravatarUrl(user?.email ?? "", 32)} icon={<UserOutlined />} />
```

同时在顶部导入 `gravatarUrl`：
```typescript
import { gravatarUrl } from "@/domain/auth/gravatar";
```

## AccountTab 改动

文件：`src/pages/settings/tabs/AccountTab.tsx`

### 新增头像区（替换原有邮箱区）

原邮箱区（只读邮箱 + 改密按钮）整体替换为以下头像区，结构：

```tsx
<Flex align="center" gap={16} data-testid="account-avatar-section">
  <Avatar
    src={gravatarUrl(email, 64)}
    icon={<UserOutlined />}
    size={64}
  />
  <Flex vertical gap={4}>
    <Typography.Text strong>{displayName}</Typography.Text>
    <Typography.Text type="secondary">{email}</Typography.Text>
    <Typography.Link
      href="https://gravatar.com"
      target="_blank"
      rel="noopener noreferrer"
      style={{ fontSize: 12 }}
    >
      {t("account.avatar.changeLink")}
    </Typography.Link>
  </Flex>
</Flex>
```

`displayName` 定义：`email.split("@")[0] || "—"`（与 MainLayout 保持一致）

### 改密按钮保留位置

改密按钮（disabled placeholder）移至头像区下方单独一行，不并入头像区：

```tsx
<Space>
  <Tooltip title={t("account.changePasswordTooltip")}>
    <Button disabled data-testid="account-change-password">
      {t("account.changePassword")}
    </Button>
  </Tooltip>
</Space>
```

### 新增导入

```typescript
import { gravatarUrl } from "@/domain/auth/gravatar";
```

`Avatar` 已在 eat 桶中，直接从 `@/ui-kit/eat` 引入。

## i18n

### zh/settings.json

在 `account` 对象下新增：
```json
"avatar": {
  "changeLink": "在 Gravatar 更换头像"
}
```

### en/settings.json

在 `account` 对象下新增：
```json
"avatar": {
  "changeLink": "Change avatar on Gravatar"
}
```

## 类型声明

`blueimp-md5` 的 `@types/blueimp-md5` 已包含完整类型，默认导出为 `(value: string) => string`，无需额外声明。
