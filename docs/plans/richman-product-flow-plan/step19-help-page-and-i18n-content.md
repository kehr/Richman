# Step 19 帮助页与 i18n 内容

## 任务目标

实现 PRD §7 帮助页：单页长文档 + 左侧锚点导航，内容按 9 个章节组织，源文件用 i18n JSON 存储中英双语。

## 涉及文件

创建：
- `frontend/src/pages/help/HelpPage.tsx`
- `frontend/src/pages/help/components/HelpSidebar.tsx`（左侧锚点导航）
- `frontend/src/pages/help/components/HelpSection.tsx`（单章节渲染）
- `frontend/src/pages/help/HelpPage.test.tsx`
- `frontend/src/i18n/help/zh-CN.json`
- `frontend/src/i18n/help/en-US.json`
- `frontend/src/i18n/help/types.ts`（HelpContent 类型定义）
- `frontend/src/i18n/help/index.ts`（按当前语言加载对应 JSON）

修改：
- `frontend/src/routes.tsx`（注册 /help 路由）
- `frontend/src/layouts/MainLayout.tsx`（如未在 step10 完成，补上底部"? 帮助"入口）

## 设计依据

- PRD §7 帮助页 9 章节
- TRD §6.2 帮助页静态 i18n JSON 方案
- TRD §6.2 锚点跳转规则

## 实施要点

- HelpPage 布局：左侧 240px sticky 锚点导航 + 右侧主内容区
- 9 章节按 PRD §7.2 顺序：
  1. 变化徽章 #badge
  2. 三维分析 #dimensions
  3. 建议动作 #actions
  4. 执行计划 #plan
  5. 信心度 #confidence
  6. 数据源 #data
  7. 降级策略 #degradation
  8. 推送时段 #push
  9. 风险偏好 #risk
- HelpContent 类型：
  - sections: 数组，每项 { id, title, body }
  - body 是 markdown 字符串
- 用 react-markdown（如已在依赖）渲染章节内容；若未安装则在本 step 添加并在 package.json 记录
- HelpSidebar 渲染所有章节标题，点击平滑滚动到对应锚点；当前可见章节高亮（IntersectionObserver）
- 路由支持锚点：/help#badge 加载页面后自动滚到对应位置
- i18n 切换：当 user_settings.language = 'en-US' 时加载 en-US.json，否则 zh-CN
- 内容写作：
  - 每章节给出 PRD 中提到的所有要点（徽章 8 种、维度 3 个、动作 5 个等）
  - 不需要太啰嗦，目标是用户能查到准确定义
  - 支持代码块、表格、列表

## 验证标准

1. `pnpm test src/pages/help` 通过
2. 浏览器手动测：
   - 访问 /help 看到 9 个章节
   - 点击侧边栏锚点平滑滚动 + 高亮
   - 决策卡 / 详情页 / Settings 中所有指向 /help#xxx 的链接都能正确定位
   - 切换语言后内容相应变化
3. 中英双语 JSON 字段对齐（同样 9 个 section id）
4. `pnpm lint:all` 通过

## 依赖说明

- 前置：step10 路由 + 菜单底部入口、step11 useUserSettings 拿语言

## 预估提交

- commit 1: `feat(i18n/help): add bilingual help content for 9 sections`
- commit 2: `feat(help): add help page with anchor sidebar`
