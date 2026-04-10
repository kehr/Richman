# 头像功能 PRD — Gravatar 接入

## 背景与目标

当前导航栏头像为静态 `UserOutlined` 图标占位，无个性化信息。目标是以最低工程成本让每个用户拥有唯一头像，并为有需求的用户提供自定义路径。

## 决策摘要

| 维度 | 决策 |
|------|------|
| 方案 | Gravatar 自动接入（零后端改动） |
| 默认风格 | identicon（基于邮箱 hash 的几何图案，每用户唯一） |
| 设置页 | 带头像预览区 + Gravatar 外链，用户知道从哪改 |
| 图片存储 | 不存储，完全由 Gravatar 托管 |
| 后端改动 | 无 |

## 用户体验

### 导航栏头像

- 顶部导航右侧用户区，头像从 `UserOutlined` 图标改为 Gravatar 头像图片（32px 圆形）
- 未注册 Gravatar 的用户自动显示 identicon（基于邮箱唯一生成的几何图案）
- Gravatar 不可达时（网络问题）静默回退到 `UserOutlined` 图标，用户无感知

### AccountTab 头像区

在设置页「账户」Tab 最顶部新增头像区，布局为：

```
[64px 圆形头像]  [用户名（邮箱前缀）]
                 [邮箱地址]
                 [在 Gravatar 更换头像 →]（外链）
```

- 头像下方不再单独显示邮箱行（邮箱已在头像区展示，避免重复）
- "在 Gravatar 更换头像 →" 点击在新标签页打开 `https://gravatar.com`

## 改动范围

| 文件 | 类型 | 说明 |
|------|------|------|
| `src/domain/auth/gravatar.ts` | 新增 | Gravatar URL 工具函数 |
| `src/layouts/MainLayout.tsx` | 修改 | Avatar 改用 Gravatar src |
| `src/pages/settings/tabs/AccountTab.tsx` | 修改 | 新增头像预览区 |
| `package.json` | 修改 | 添加 blueimp-md5 依赖 |
| `src/i18n/locales/zh/settings.json` | 修改 | 新增头像区翻译 key |
| `src/i18n/locales/en/settings.json` | 修改 | 新增头像区翻译 key |

## Gravatar URL 规则

```
https://www.gravatar.com/avatar/{md5(email.trim().toLowerCase())}?d=identicon&s={size}&r=g
```

- `d=identicon`：未注册时显示几何图案默认头像
- `s`：像素尺寸，导航栏 32，设置页 64
- `r=g`：过滤限制级内容

## i18n Keys

| Key | 中文 | English |
|-----|------|---------|
| `settings.account.avatar.changeLink` | 在 Gravatar 更换头像 | Change avatar on Gravatar |

## 不在范围内

- 本地图片上传（MVP 阶段不做）
- 自定义 avatar_url 字段（无后端改动）
- 头像裁剪工具
- 头像缓存策略（浏览器默认缓存即可）
