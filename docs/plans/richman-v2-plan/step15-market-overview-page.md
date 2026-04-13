# Step 15: Market Overview Page

> Phase 4 | 并行组 R9 (可与 Step 16, 17, 18 同时执行) | 前置: Step 14

## 任务目标

实现 Market Overview 页面（/market）：MarketRegimeBar 体制判断条、AssetCardWall 标的卡片墙（分组 + 激活/置灰两态）、EventRadarSection 事件雷达列表、RegisterCTA 注册引导条，以及相关 feature 模块（market-overview + event-radar）和 i18n 翻译。

## 涉及文件

### 创建

**Feature 模块：**
- `frontend/src/features/market-overview/api.ts`
- `frontend/src/features/market-overview/types.ts`
- `frontend/src/features/market-overview/use-market-regime.ts`
- `frontend/src/features/market-overview/use-market-overview.ts`
- `frontend/src/features/market-overview/index.ts`
- `frontend/src/features/event-radar/api.ts`
- `frontend/src/features/event-radar/types.ts`
- `frontend/src/features/event-radar/use-event-radar.ts`
- `frontend/src/features/event-radar/index.ts`

**页面 + 组件：**
- `frontend/src/pages/market-overview/market-overview-page.tsx`
- `frontend/src/pages/market-overview/components/market-regime-bar.tsx`
- `frontend/src/pages/market-overview/components/asset-card-wall.tsx`
- `frontend/src/pages/market-overview/components/asset-group-section.tsx`
- `frontend/src/pages/market-overview/components/asset-card.tsx`
- `frontend/src/pages/market-overview/components/event-radar-section.tsx`
- `frontend/src/pages/market-overview/components/register-cta.tsx`

**通用工具：**
- 涨跌颜色函数 getPriceChangeColor（放置在合适的通用位置）

### 修改

- `frontend/src/i18n/locales/zh/market.json` (或对应 namespace 文件) -- 新增
- `frontend/src/i18n/locales/en/market.json`
- `frontend/src/i18n/locales/zh/common.json` -- 新增 signal.* / regime.* key
- `frontend/src/i18n/locales/en/common.json`

## 设计依据

| 内容 | PRD 章节 | TRD 章节 |
|------|----------|----------|
| Market Overview 页面结构 | SS4 Market Overview | frontend SS4.1 |
| MarketRegimeBar | SS4.2.1 体制判断 | frontend SS4.2 |
| AssetCardWall 分组 | SS4.2.2 标的卡片 | frontend SS4.3 |
| 激活/置灰卡片 | SS1.7 框架完整灯亮一盏 | frontend SS4.3 |
| EventRadarSection | SS4.2.4 事件雷达 | frontend SS4.6 |
| RegisterCTA | SS4.2.5 注册引导 | frontend SS4.5 |
| 涨跌颜色 (A 股红涨绿跌) | SS4.2.3 | frontend SS4.4 |
| 货币展示 (USD/CNY) | SS4.2.4 | frontend SS5.4 |
| SEO meta 标签 | - | frontend SS12.1 |
| i18n market.* key | - | frontend SS9.2 |
| richson 503 降级 | - | frontend SS16.9 |

## 关键约束 + 已知问题处理

| 已知问题 | 处理要求 | TRD 引用 |
|----------|----------|----------|
| G3.5 涨跌颜色 | 用 assetCode 判定 A 股（6 位数字） | frontend SS16.5 |
| G3.9 richson 503 降级 | MarketRegimeBar 隐藏，EventRadar 占位+retry | frontend SS16.9 |

- 公开页面，不需要 JWT
- 置灰标的显示名称 + "即将开放" 标签，不可点击
- 体制标签颜色：risk_on 绿 / neutral 灰 / risk_off 红
- RegisterCTA 仅未登录用户可见
- 使用 Helmet 设置 SEO meta 标签
- 所有 Ant Design 组件通过 ui-kit/eat barrel 导入
- i18n 同时更新 zh + en 两个 locale

## 验证标准

- [ ] `cd frontend && pnpm lint:all` 全部通过
- [ ] `pnpm build` 成功
- [ ] 浏览器访问 /market 页面正常渲染
- [ ] 体制判断条显示三种状态颜色
- [ ] 激活标的卡片可点击跳转 /market/:code
- [ ] 置灰标的不可点击
- [ ] 事件雷达列表正常展示
- [ ] 未登录时显示 RegisterCTA
- [ ] A 股标的涨跌使用红涨绿跌

## 变更点清单覆盖

E3.1 (1), E1.1 (1), E1.3 (1), E4.1-E4.6 (6), E11.1 (1), E11.8 (1), E12.1-E12.2 (2), E12.4 (1), G3.5 (1), G3.9 (1) = **16 项**
