# 前端 v2 重构 TRD

> 版本 1.0 | 关联 PRD: docs/prds/richman-prd-v2.md | 关联 TRD: docs/trds/richson-service-trd.md

## 1. 文档范围

本 TRD 覆盖 v2 版本前端的完整重构设计：

- 路由结构重新设计（零 onboarding + Market Overview 首页）
- Market Overview 页面（PRD SS4）
- 标的详情页（Sticky Header + 三 Tab，PRD SS5）
- 投研简报页重构（PRD SS6）
- 持仓管理页适配（标记模式 + 风险偏好）
- 新增 feature 模块与 API 层
- i18n namespace 扩展
- 权限矩阵与路由守卫重构

不在本 TRD 范围：richson 服务内部设计、数据库 schema、richman API 实现细节（见 richson-service-trd.md）。

## 2. 路由结构

### 2.1 v2 路由表

| 路由 | 页面组件 | 权限 | 布局 |
|------|----------|------|------|
| `/` | 重定向至 `/market` | 公开 | - |
| `/market` | MarketOverviewPage | 公开 | MainLayout（简化版导航） |
| `/market/:code` | AssetDetailPage | 公开（执行 Tab 需登录+持仓） | MainLayout |
| `/portfolio` | PortfolioListPage | JWT | MainLayout |
| `/portfolio/:id` | PortfolioEditPage | JWT | MainLayout |
| `/portfolio/:id/transactions` | PortfolioTransactionsPage | JWT | MainLayout |
| `/briefing` | ResearchBriefingPage | JWT | MainLayout |
| `/settings` | SettingsPage | JWT | MainLayout |
| `/settings/risk-preference` | RiskPreferenceSubPage | JWT | MainLayout |
| `/help` | HelpPage | 公开 | MainLayout |
| `/login` | LoginPage | 匿名 | AuthLayout |
| `/register` | RegisterPage | 匿名 | AuthLayout |

### 2.2 移除的路由

| 原路由 | 处理 |
|--------|------|
| `/onboarding/*` (welcome, categories, first-holding, llm-consent, first-analysis) | 全部删除，零 onboarding |
| `/briefing`（原 `/dashboard`） | 路径已在 v1 后期改名为 `/briefing`（当前代码已生效），v2 保持不变 |
| `/decision-cards/:id` | 功能合并到 `/market/:code` 标的详情页的执行 Tab |

### 2.3 导航栏

```
未登录: [百万 Richman Logo]   [行情]                              [登录] [注册]
已登录: [百万 Richman Logo]   [行情]  [持仓]  [投研简报]          [头像 Dropdown]
                               /market  /portfolio  /briefing      ├ 设置
                                                                    ├ 帮助
                                                                    └ 退出
```

导航栏始终展示，根据登录状态切换项目。"行情" 对所有用户可见。

### 2.4 路由守卫重构

v1 的 `OnboardingGuard` 完全移除。路由守卫简化为：

- `AuthGuard`：检查 JWT token 有效性，无效则重定向至 `/login?returnTo={currentPath}`
- `AssetDetailGuard`：标的详情页内部逻辑——执行 Tab 根据登录状态和持仓状态展示不同内容（PRD SS9.2）

不再有全局 onboarding 拦截。用户注册后直接进入 `/market`。

## 3. 新增 Feature 模块

### 3.1 feature/market-overview

Market Overview 页面的数据层。

```
features/market-overview/
  api.ts              # fetchMarketRegime, fetchMarketOverview
  types.ts            # MarketRegimeDto, MarketOverviewDto, AssetCardDto
  use-market-regime.ts
  use-market-overview.ts
  index.ts
```

API 映射：
- `fetchMarketRegime()` -> `GET /api/v2/market/regime`
- `fetchMarketOverview()` -> `GET /api/v2/market/overview`

Query Keys：
```typescript
const MARKET_REGIME_KEY = ["market", "regime"] as const;
const MARKET_OVERVIEW_KEY = ["market", "overview"] as const;
```

### 3.2 feature/asset-detail

标的详情页的数据层。

```
features/asset-detail/
  api.ts              # fetchAssetDetail, fetchAssetOhlcv, fetchScoreHistory, fetchDemoPlan, triggerHoldingAnalysis
  types.ts            # AssetDetailDto, OhlcvDto, ScoreHistoryDto, DemoPlanDto, DimensionDetailDto, MajorChangeRecapDto
  use-asset-detail.ts
  use-asset-ohlcv.ts
  use-score-history.ts
  use-demo-plan.ts
  use-trigger-holding-analysis.ts
  use-analysis-job.ts   # polling hook for job progress
  index.ts
```

API 映射：
- `fetchAssetDetail(code)` -> `GET /api/v2/market/{code}`
- `fetchAssetOhlcv(code, period)` -> `GET /api/v2/market/{code}/ohlcv?period=3M`
- `fetchScoreHistory(code, days)` -> `GET /api/v2/market/{code}/scores?days=90`
- `fetchDemoPlan(code)` -> `GET /api/v2/market/{code}/demo-plan`
- `triggerHoldingAnalysis(holdingId)` -> `POST /api/v2/analysis/holding/{holdingId}`
- `fetchAnalysisJob(jobId)` -> `GET /api/v2/analysis/jobs/{jobId}`
- `triggerAssetAnalysis(code)` -> `POST /api/v2/analysis/trigger-asset`

### 3.3 feature/event-radar

事件雷达数据层。

```
features/event-radar/
  api.ts              # fetchEventRadar
  types.ts            # EventRadarDto, EventDto
  use-event-radar.ts
  index.ts
```

API 映射：
- `fetchEventRadar()` -> `GET /api/v2/events/radar`

### 3.4 feature/research-briefing

投研简报数据层（重构自 dashboard-summary）。

```
features/research-briefing/
  api.ts              # fetchBriefing
  types.ts            # BriefingDto, BriefingCardDto
  use-briefing.ts
  index.ts
```

API 映射：
- `fetchBriefing()` -> `GET /api/v2/briefing`

### 3.5 feature/user-feedback

用户反馈（PRD SS6.3）。

```
features/user-feedback/
  api.ts              # postFeedback
  types.ts            # FeedbackDto
  use-submit-feedback.ts
  index.ts
```

### 3.6 已有 feature 模块变更

| 模块 | 变更 |
|------|------|
| portfolio | 新增 `useHoldingByAssetCode(code)` hook，标的详情页检查用户是否持有该标的；Holding 类型新增 `entryMode: "tag" | "quick" | "detail"` 字段 |
| user-settings | 新增 `riskPreference` 字段（conservative/moderate/aggressive），新增 `usePatchRiskPreference` hook |
| decision-card | v1 保留兼容，不再新增功能；v2 决策卡片数据通过 asset-detail feature 获取 |
| asset-catalog | 无变更，继续服务标的选择器 |
| settings-llm | 无变更 |
| notification-channels | 无变更 |
| auth | 无接口变更，注册表单增加 disclaimerAccepted checkbox 和 ref 参数自动填充 |
| dashboard-llm-status | 评估是否迁移到 ResearchBriefingPage 或废弃 |
| schedule | 无变更 |

新增 feature 模块 `invite`（完整设计见 invite-system-trd.md SS7/SS10）：
- `api.ts` / `types.ts` / `use-my-codes.ts` / `use-my-invites.ts` / `index.ts`
- SettingsPage 中新增 InviteSection 区块：邀请码列表（可复制）+ 连续登录解锁进度 + 已邀请用户列表
- SettingsPage 中新增 EmailPushToggle：平台邮件推送开关（Switch 组件，调用 `PATCH /api/v2/user/email-push`）
- SettingsPage 中新增 AccountDeletionSection：账户注销入口（需密码确认，调用 `DELETE /api/v1/auth/account`，richman-backend-v2-trd SS21）

## 4. Market Overview 页面

### 4.1 页面结构

```
MarketOverviewPage
├── MarketRegimeBar          # 顶部体制判断条
├── AssetCardWall            # 标的卡片墙（分组）
│   ├── AssetGroupSection    # 每个资产类别一组
│   │   └── AssetCard[]      # 单个标的卡片
├── EventRadarSection        # 事件雷达
└── RegisterCTA              # 注册引导（仅未登录）
```

### 4.2 MarketRegimeBar

展示一句话市场判断 + 关键指数快照（PRD SS4.2.1）。

数据来源：`useMarketRegime()` hook。

UI 结构：
- 左侧：体制标签（风险偏好/中性/风险规避）+ 一句话原因
- 右侧：4 个指数迷你卡片（标普500、纳斯达克、上证、黄金），每个显示价格 + 涨跌幅

体制标签颜色映射：
- `risk_on` -> 绿色背景
- `neutral` -> 灰色背景
- `risk_off` -> 红色背景

### 4.3 AssetCardWall

按资产类别分组展示（PRD SS4.2.2）。

分组顺序：商品（黄金）-> 权益（A股/美股）-> 固定收益 -> 数字资产。

激活标的卡片内容：
- 标的名称（中/英文随 locale 切换）
- 当前价格 + 涨跌幅（颜色规则见 SS4.4）。货币展示规则（PRD SS4.2.4）：USD 标的显示 `$4,750.00`，CNY 标的显示 `CN4.85`
- 综合评分 + 方向标签（看涨/中性/看空）
- 历史分位标签（自然语言，如"近一年中高"）
- 点击跳转 `/market/:code`

置灰标的：
- 标的名称 + "即将开放" 标签
- 灰度样式，不可点击

### 4.4 涨跌颜色逻辑

```typescript
function getPriceChangeColor(assetCode: string, changePercent: number): string {
  const isAShare = /^\d{6}$/.test(assetCode);
  if (changePercent > 0) return isAShare ? "red" : "green";
  if (changePercent < 0) return isAShare ? "green" : "red";
  return "gray";
}

// Direction labels always use international convention
function getDirectionColor(signal: string): string {
  if (signal === "bullish" || signal === "strong_bullish") return "green";
  if (signal === "bearish" || signal === "strong_bearish") return "red";
  return "gray";
}
```

A 股判定规则：`assetCode` 为纯 6 位数字的标的使用中国市场颜色惯例（红涨绿跌）；其余使用国际惯例（绿涨红跌）。方向标签始终绿=看涨，红=看空（PRD SS4.2.3）。

### 4.5 RegisterCTA

仅未登录用户可见（PRD SS4.2.5）：

- 底部固定条
- 文案："注册 百万 Richman，获取个性化投资执行计划"
- 按钮跳转 `/register`

### 4.6 EventRadarSection

未来 7 天关键宏观事件列表（PRD SS4.2.4）。

数据来源：`useEventRadar()` hook。

每个事件行：日期 + 事件标题 + 影响级别 + 对黄金方向 + Polymarket 概率 + 24h 变动。

## 5. 标的详情页

### 5.1 页面结构

```
AssetDetailPage
├── StickyHeader               # 顶部固定区（始终可见）
│   ├── AssetIdentity          # 名称 + 价格 + 涨跌
│   ├── ScoreSummary           # 评分 + 方向 + 分位
│   ├── ChangeSummary          # 今日变化摘要（条件展示，delta >= 5）
│   ├── MajorChangeRecap       # 重大变化复盘（条件展示，|delta| > 20）
│   ├── ConflictWarning        # 冲突警告（条件展示）
│   └── FreshnessIndicator     # 数据新鲜度（条件展示）
└── Tabs
    ├── AnalysisTab            # [分析] Tab
    ├── RiskTab                # [风险] Tab
    └── ExecutionTab           # [执行] Tab
```

### 5.2 StickyHeader

固定在页面顶部，滚动不消失（PRD SS5.2.1）。

组件树：
```
<div style={{ position: "sticky", top: 0, zIndex: 10, background: token.colorBgContainer }}>
  <AssetIdentity code={code} name={name} price={price} currency={currency} changePercent={changePercent} />
  <ScoreSummary score={overallScore} signal={signalLevel} percentileLabel={percentileLabel} />
  {scoreDelta >= 5 && <ChangeSummary changes={changeSummary} />}
  {Math.abs(scoreDelta) > 20 && majorChangeRecap && <MajorChangeRecap text={majorChangeRecap} />}
  {conflictType && <ConflictWarning type={conflictType} message={conflictMessage} />}
  {priceDrift > 2 && <FreshnessIndicator drift={priceDrift} analysisTime={analyzedAt} />}
</div>
```

FreshnessIndicator 三级警告（PRD SS5.2.1）：

| 价格偏移 | 级别 | 样式 |
|----------|------|------|
| > 2% | 黄色提示 | Alert type="warning" |
| > 5% | 橙色警告 | Alert type="warning" + 橙色自定义样式 |
| > 10% | 红色强警告 | Alert type="error" + 评分灰度化 |

价格偏移计算：`abs(currentPrice - priceAtAnalysis) / priceAtAnalysis * 100`。`currentPrice` 从 market-quote feature 获取（实时），`priceAtAnalysis` 从分析记录获取。

### 5.3 [分析] Tab

回答"为什么是这个评分"（PRD SS5.2.2）。

组件树：
```
<AnalysisTab>
  <OhlcvChart code={code} period={period} sma200={sma200} supports={supports} resistances={resistances} />
  <InterpretationCard text={marketInterpretation} />
  <DimensionPanelList dimensions={dimensions} />
  <ScoreTrendChart code={code} days={days} versionChanges={versionChanges} />
</AnalysisTab>
```

#### OhlcvChart

K 线图组件。支持 1D/1W/1M/3M/1Y 切换。

- 使用 Ant Design Charts 或 lightweight-charts
- 叠加 200 日均线（虚线）
- 标注支撑位（绿色水平线）和阻力位（红色水平线）
- 数据来源：`useAssetOhlcv(code, period)`

#### DimensionPanelList

四维分析折叠面板。每个面板标题：维度名称 + 得分 + 方向箭头 + 权重%。

LLM 调整时展示双层分数："结构性需求 82 分（量化基础 70 -> LLM +12）"。

展开后：
- 子指标表格（指标名、原始值、百分位、归一化得分、权重）
- LLM 调整原因（如有）
- 维度名称旁 "?" 图标，hover/click 展示概念解释（PRD SS5.2.2 用户教育）

概念解释文本通过 i18n 管理：`assetDetail.dimension.d1.explanation`。

#### ScoreTrendChart

评分趋势线图。默认 90 天。

- X 轴：日期
- Y 轴：0-100 分
- 主线：综合评分
- 模型版本变更处画竖线 + 标签
- 天数切换：30 / 90 / 180 / 240（与 API 参数对齐，PRD SS5.2.2）

数据来源：`useScoreHistory(code, days)`

### 5.4 [风险] Tab

回答"有什么可能出错"（PRD SS5.2.3）。

```
<RiskTab>
  <RiskFactorList factors={riskFactors} />
  <KeyPriceLevels supports={supports} resistances={resistances} currentPrice={price} />
  <DrawdownReference />
  <EventCalendar events={events} />
</RiskTab>
```

RiskFactorList：2-3 条 LLM 生成的风险因子，每条 30-50 字。

KeyPriceLevels：支撑/阻力位表格，展示当前价与各水平的距离百分比。CNY 标的附注 USD 等价（PRD SS4.2.4），格式："支撑 CN4.85（约 $4,600）"。

**货币展示统一规则**（PRD SS4.2.4，适用于 AssetCard / StickyHeader / RiskTab / ExecutionTab 所有价格展示位置）：
- API 返回 `currency` 字段（"USD" | "CNY"）
- USD 标的：`$` 前缀，如 `$4,750.00`
- CNY 标的：`CN` 前缀，如 `CN4.85`。在支撑/阻力位、止损/止盈等关键价格位置附注 USD 等价（括号内 `约 $X,XXX` 格式）
- USD 等价计算：API 返回 `usdExchangeRate` 字段（richson 在分析时快照当日 CNY/USD 汇率并存入 `rs_asset_analyses.usd_exchange_rate`）。前端对 CNY 关键价格做简单乘法 `price * usdExchangeRate` 得到 USD 等价，不需要独立获取汇率。USD 标的该字段为 null，跳过等价标注

DrawdownReference：展示当前牛市中的最大回撤幅度及历史对比（PRD SS5.2.3）。数据来自 API 响应中的 `analysis.drawdownReference` 字段（richson SS7.7 计算，存入 rs_asset_analyses.analysis_metadata），展示格式："当前轮牛市最大回撤 -8.5%（2026-02-15），历史均值 -12%"。使用 Ant Design Statistic 组件。

EventCalendar：复用 EventRadarSection 组件，筛选与当前标的相关的事件。

### 5.5 [执行] Tab

回答"我该怎么做"（PRD SS5.2.4）。

根据用户状态展示不同内容：

```typescript
function ExecutionTab({ code, assetAnalysis }: Props) {
  const { user } = useCurrentUser();
  const { data: holding } = useHoldingByAssetCode(code);

  if (!user) {
    return <DemoPlanWithRegisterCTA code={code} />;
  }
  if (!holding) {
    return <DemoPlanWithAddHoldingCTA code={code} />;
  }
  return <FullExecutionPlan holding={holding} assetAnalysis={assetAnalysis} onTriggerAnalysis={triggerHoldingAnalysis} />;
}
```

#### DemoPlanWithRegisterCTA / DemoPlanWithAddHoldingCTA

展示示范执行计划（PRD SS5.2.4）：
- 预设假设持仓参数生成的执行计划
- 底部 CTA：未登录 -> "注册获取专属计划"；已登录无持仓 -> "录入持仓获取专属计划"
- 顶部提示："以上基于示范持仓，非为您定制"

数据来源：`useDemoPlan(code)`

#### FullExecutionPlan

完整条件分支执行计划（PRD SS8.1）：
- 当前持仓概况（成本、仓位、浮盈亏）
- 操作建议标题 + 默认建议
- 条件分支场景列表（IF/THEN，含优先级标记）
- 止损/止盈
- 有效期
- 集中度警告（如有）
- 免责声明（PRD SS13.2）

场景优先级视觉：priority=1（止损）用红色边框标记。

执行计划新鲜度：执行计划基于标的分析结果生成，当标的分析数据过期（`validDays` 到期或 `priceDrift > 5%`）时，展示"执行计划可能已过时"提示 + "刷新分析"按钮。刷新按钮调用 `triggerHoldingAnalysis(holdingId)` 触发重新分析，进入 job 轮询流程。

### 5.6 Tab 加载策略

- [分析] Tab：页面进入时预加载（默认展示）
- [风险] Tab：懒加载（用户点击时请求数据）
- [执行] Tab：懒加载

实现方式：Ant Design Tabs 的 `items` 配置中对每个 TabItem 设置 `destroyOnHidden: false`（antd 5.25+ 新属性，替代已 deprecated 的 `destroyInactiveTabPane`；该属性在 Tabs 组件级别和 items 级别均可设置） + 每个 Tab 内部 hook 的 `enabled` 参数控制。

## 6. 投研简报页

### 6.1 页面结构

重构自 DashboardPage，改名为 ResearchBriefingPage（PRD SS6）。

```
ResearchBriefingPage
├── BriefingHeader             # 标题 + 切换模式（简洁/详细）
├── BriefingCardList           # 持仓决策卡片列表
│   └── BriefingCard[]         # 单张简报卡片
└── EmptyBriefingState         # 无持仓空状态
```

### 6.2 BriefingCard

每张卡片对应一个持仓标的（PRD SS6.2）：

| 区域 | 内容 |
|------|------|
| 头部 | 标的名称 + 综合评分 + 方向标签 |
| 持仓信息 | 成本价、仓位比例、浮盈亏 |
| 迷你趋势图 | 90 天评分趋势 sparkline |
| 今日变化 | 评分变化归因（评分变化 >= 5 分时展示） |
| 冲突警告 | 维度冲突提示（如有） |
| 操作摘要 | 执行计划首要场景的一句话摘要 |
| 反馈区 | 点赞/点踩按钮（PRD SS6.3） |

点击卡片跳转 `/market/:code`（执行 Tab）。

### 6.3 简洁/详细模式

- 简洁模式：只展示头部 + 持仓信息 + 操作摘要
- 详细模式：展示全部区域

默认简洁模式，用户切换后记住偏好（localStorage key = `richman_briefing_view_mode`）。

## 7. 持仓管理页适配

### 7.1 标记模式新增

v2 新增"标记模式"作为最轻量的录入方式（PRD SS7.1）：

```
AddHoldingDrawer
├── ModeSelector              # 标记模式 / 快速模式 / 明细模式
├── TagModeForm               # 标记模式表单（新增）
│   ├── AssetSelector         # 标的选择（复用）
│   └── PositionTierRadio     # 仓位档位：轻仓(<10%) / 中仓(10-25%) / 重仓(>25%)
├── QuickModeForm             # 快速模式表单（已有，微调）
└── DetailModeForm            # 明细模式表单（已有）
```

标记模式创建 holding 时：
- `costPrice` = 当前市场价（自动填充）
- `positionRatio` = 档位中值（轻仓 5%、中仓 17.5%、重仓 30%）
- 标记 `entryMode: "tag"` 以便后续提示用户升级

### 7.2 风险偏好设置

首次录入持仓时弹出风险偏好选择（PRD SS7.6）：

```
RiskPreferenceModal
├── ConservativeCard          # 保守型
├── ModerateCard              # 稳健型（默认高亮）
└── AggressiveCard            # 进取型
```

也可在 Settings -> 账户 中调整。

### 7.3 持仓列表升级提示

标记模式的持仓在列表中显示"补充详情"标签，提示用户升级到快速模式或明细模式。

### 7.4 持仓集中度警告

持仓列表页展示同类标的集中度警告（PRD SS8.3）。当同一二级分类下合计仓位超阈值时，在列表顶部展示 Alert 组件：

| 合计仓位 | 级别 | 展示 |
|----------|------|------|
| >= 35% | 红色 Alert type="error" | "黄金配置严重集中，强烈建议控制敞口" |
| >= 25% | 橙色 Alert type="warning" | "黄金配置集中度较高，请注意分散风险" |
| >= 15% | 蓝色 Alert type="info" | "黄金配置已达 X%，处于机构建议区间上限" |

集中度计算：`sumBy(holdings.filter(h => h.assetType === currentType), 'positionRatio')`。

### 7.5 LLM 配置引导

零 onboarding 后，LLM 配置引导时机改为：用户首次录入持仓并触发持仓级分析时（PRD SS9.1）。

触发逻辑：
1. 用户录入第一个持仓
2. 系统检查 rm_llm_configs 是否已有配置
3. 未配置 -> 弹出 LLMConfigModal（BYOK 或使用系统默认），配置完成后自动触发首次分析
4. 已配置 -> 直接触发分析

LLMConfigModal 复用已有 features/settings-llm 模块的组件，仅在 PortfolioListPage 中增加首次触发逻辑。

## 8. API 客户端层

### 8.1 v2 API base path

```typescript
// domain/http/client.ts
const API_V1_BASE = `${import.meta.env.VITE_API_BASE}/api/v1`;
const API_V2_BASE = `${import.meta.env.VITE_API_BASE}/api/v2`;

export function requestV1<T>(url: string, options?: RequestInit): Promise<T> {
  return request<T>(`${API_V1_BASE}${url}`, options);
}

export function requestV2<T>(url: string, options?: RequestInit): Promise<T> {
  return request<T>(`${API_V2_BASE}${url}`, options);
}
```

v1 端点继续使用 `requestV1`，v2 端点使用 `requestV2`。共享认证逻辑（JWT 注入、401 跳登录）。

**迁移说明**：现有 `domain/http/client.ts` 将 `/api/v1` 硬编码在 `API_BASE` 中。v2 重构时拆为 host-only base + 版本前缀常量，现有 `request()` 重命名为内部 `_request()`，`requestV1`/`requestV2`/`requestPublic` 作为公开 API。现有 feature 的调用点逐步从 `request()` 迁移到 `requestV1()`。

### 8.2 公开 API 无认证

Market Overview 和标的详情页的 API 不携带 JWT token：

```typescript
export function requestPublic<T>(url: string, options?: RequestInit): Promise<T> {
  // Same as requestV2 but without Authorization header
  return fetch(`${API_V2_BASE}${url}`, {
    ...options,
    headers: { "Content-Type": "application/json", ...options?.headers },
  }).then(handleResponse<T>);
}
```

### 8.3 Job 进度轮询 hook

```typescript
function useAnalysisJob(jobId: string | null) {
  return useQuery({
    queryKey: ["analysis", "job", jobId],
    queryFn: () => fetchAnalysisJob(jobId!),
    enabled: !!jobId,
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === "completed" || status === "failed") return false;
      return 3000; // 3s polling interval
    },
  });
}
```

轮询策略（PRD 讨论结论）：
- 初始间隔 3 秒
- job 完成或失败后停止轮询
- 前端最大轮询 60 次（3 分钟），超时后展示"分析超时，请稍后重试"
- MVP 不使用 WebSocket

## 9. i18n 扩展

### 9.1 新增 namespace

不新增 namespace，在现有 4 个 namespace 中扩展：

| namespace | 新增 key 前缀 | 用途 |
|-----------|--------------|------|
| common | `assetCategory.*`（已有）, `signal.*`, `regime.*` | 评分信号标签、体制标签 |
| app | `market.*`, `assetDetail.*`, `briefing.*`, `eventRadar.*` | 三大核心页面文案 |
| settings | `settings.account.riskPreference.*`, `settings.account.emailPush.*`, `settings.account.deleteAccount.*`, `settings.invite.*` | 风险偏好、邮件推送开关、账户注销、邀请码 |

### 9.2 关键翻译 key 示例

```json
{
  "market": {
    "regime": {
      "riskOn": "风险偏好",
      "neutral": "中性",
      "riskOff": "风险规避"
    },
    "card": {
      "comingSoon": "即将开放",
      "score": "综合评分"
    },
    "registerCTA": "注册 百万 Richman，获取个性化投资执行计划"
  },
  "assetDetail": {
    "tab": {
      "analysis": "分析",
      "risk": "风险",
      "execution": "执行"
    },
    "freshness": {
      "mild": "金价自上次分析后已变动 {{percent}}%，当前评分可能未反映最新走势",
      "moderate": "金价大幅变动 {{percent}}%，当前分析可能已失效，请谨慎参考",
      "severe": "金价剧烈波动 {{percent}}%，当前分析不再可靠，请勿基于此评分做交易决策"
    },
    "dimension": {
      "d1": { "name": "宏观利率", "explanation": "TIPS 收益率是扣除通胀后的真实利率..." },
      "d2": { "name": "美元流动性", "explanation": "..." },
      "d3": { "name": "结构性需求", "explanation": "..." },
      "d4": { "name": "技术位置", "explanation": "..." }
    },
    "demoPlan": {
      "disclaimer": "以上基于示范持仓，非为您定制",
      "registerCTA": "注册获取专属执行计划",
      "addHoldingCTA": "录入持仓获取专属执行计划"
    },
    "percentile": {
      "veryHigh": "近一年偏高",
      "high": "近一年中高",
      "mid": "近一年中位",
      "low": "近一年中低",
      "veryLow": "近一年偏低"
    }
  }
}
```

zh 和 en 同步维护。

## 10. 权限矩阵实现

PRD SS9.2 的用户状态权限矩阵在前端的实现策略：

| 页面/区域 | 未登录 | 已登录无持仓 | 已登录有持仓 |
|-----------|--------|-------------|-------------|
| MarketOverviewPage | 完整展示 + RegisterCTA | 完整展示 | 完整展示 |
| AssetDetailPage [分析] | 完整展示 | 完整展示 | 完整展示 |
| AssetDetailPage [风险] | 完整展示 | 完整展示 | 完整展示 |
| AssetDetailPage [执行] | Demo Plan + 注册引导 | Demo Plan + 录入引导 | 完整执行计划 |
| PortfolioListPage | 重定向 /login | 空状态 + 添加引导 | 持仓列表 |
| ResearchBriefingPage | 重定向 /login | 空状态 | 简报卡片列表 |

实现原则：
- 公开页面不做前端路由拦截，API 不需要 token
- 登录保护页面通过 AuthGuard 重定向
- 执行 Tab 的三态切换在组件内部判断，不做路由级拦截

## 11. 组件库使用规范

### 11.1 图表库选型

K 线图和趋势线需要图表库。选型：

- K 线图（OhlcvChart）：使用 `lightweight-charts`（TradingView 开源库）。当前安装版本 ^4.2.2，使用 v4 API（`chart.addCandlestickSeries()`）。v2 实现沿用 v4 API，如需升级 v5 须在 plan 阶段单独评估 API 迁移工作量
- 趋势线/Sparkline：使用已安装的 `echarts` + `echarts-for-react`，与现有项目技术栈统一

两者通过 lazy import 引入，避免首屏加载负担。

### 11.2 Sticky Header 实现

使用 CSS `position: sticky` + Ant Design 的 `token.colorBgContainer` 背景色，不使用 JS scroll listener。

```css
.sticky-header {
  position: sticky;
  top: 0;
  z-index: 10;
  border-bottom: 1px solid var(--ant-color-border);
  backdrop-filter: blur(8px);
}
```

### 11.3 评分区间条

PRD SS3.4 要求展示置信区间色带：

```typescript
function ScoreBar({ score, bandLow, bandHigh }: Props) {
  // 0-100 scale bar with:
  // - Filled segment from bandLow to bandHigh (gradient)
  // - Point marker at score position
  // - 5-zone background colors (0-19 red, 20-39 orange, 40-59 gray, 60-79 green, 80-100 dark green)
}
```

## 12. SEO 与社交分享

### 12.1 动态 meta 标签

Market Overview 和标的详情页需要 SEO 友好的 meta 标签（PRD SS4.3）。

SPA 方案：使用 `@dr.pogodin/react-helmet` 动态设置 `<title>` 和 `<meta>` 标签。此为新增依赖，需 `pnpm add @dr.pogodin/react-helmet`（原 react-helmet-async 已停维，此 fork 显式支持 React 19）。

```typescript
// AssetDetailPage
<Helmet>
  <title>{`${assetName} | ${overallScore}/100 ${signalLabel} | 百万 Richman`}</title>
  <meta name="description" content={marketInterpretation.slice(0, 160)} />
  <meta property="og:title" content={`${assetName} ${signalLabel}`} />
  <meta property="og:description" content={marketInterpretation.slice(0, 100)} />
</Helmet>
```

注意：SPA 的 meta 标签对搜索引擎爬虫支持有限。后续可考虑 SSR（Next.js）或 prerender 方案。MVP 先用客户端 meta 标签，满足社交分享 Open Graph 需求。

### 12.2 分享功能

标的详情页"分享"按钮，MVP 仅支持"复制链接"。微信分享图片、微博、Twitter 等渠道后续迭代。

## 13. 废弃代码清理

### 13.1 移除列表

| 模块/文件 | 原因 |
|-----------|------|
| pages/onboarding/* | 零 onboarding，全部移除 |
| pages/onboarding/state.tsx | onboarding 状态管理 |
| pages/onboarding/use-onboarding-nav.ts | onboarding 导航 |
| domain/auth/onboarding-guard.tsx | onboarding 路由守卫 |
| features/user-settings/use-onboarding-status.ts | 保留但不再影响路由 |
| features/user-settings/use-mark-onboarding-completed.ts | 保留但标记 deprecated |
| features/user-settings/use-reset-onboarding.ts | 保留但标记 deprecated |
| features/user-settings/use-skip-onboarding.ts | 保留但标记 deprecated |
| pages/dashboard/* | 重构为 ResearchBriefingPage |

### 13.2 保留列表

| 模块 | 原因 |
|------|------|
| features/decision-card | v1 决策卡片仍有历史数据展示需求 |
| features/dashboard-summary | 改名为 research-briefing，API 切换到 v2 |
| features/market-quote | 保留，标的详情页 StickyHeader 用于获取实时价格做新鲜度计算 |

## 14. 免责声明展示

所有 LLM 生成的面向用户文本位置需展示免责声明（PRD SS13.2）：

| 位置 | 方式 |
|------|------|
| 标的详情页底部 | 固定灰色小字，不可关闭 |
| 执行计划区顶部 | 一行简短提示 |
| 注册流程 | RegisterPage 中"注册"按钮上方增加 Checkbox："已阅读并理解免责声明"（文字链接可展开全文）。未勾选时注册按钮 disabled |
| 首次查看执行计划 | Modal 确认（localStorage key = `richman_disclaimer_confirmed`，仅触发一次） |

免责声明文本通过 i18n key `common.disclaimer.*` 管理。注册勾选使用 `common.disclaimer.registerCheckbox`。

## 15. 响应式与主题兼容

### 15.1 响应式设计

MVP 以桌面端为主设计（>= 1024px），但核心公开页面（Market Overview、标的详情页）需支持移动端基本可用：

| 断点 | 适配策略 |
|------|----------|
| >= 1024px | 完整布局 |
| 768-1023px | 卡片墙两列，Tab 内容单列 |
| < 768px | 卡片墙单列，StickyHeader 简化（仅评分+方向），图表宽度自适应 |

实现方式：使用 Ant Design 的 Grid 响应式（`Col xs/sm/md/lg`），不引入额外 CSS 框架。

### 15.2 暗色主题

PRD SS16.1 要求支持亮色/暗色主题。实现策略：

- 使用 Ant Design 的 `ConfigProvider` 主题切换（`algorithm: theme.darkAlgorithm`）（当前 antd ^5.24，待 @ant-design/pro-components v3 正式发布后统一升级 antd 6）
- 自定义 token 通过 CSS 变量注入，涨跌颜色、评分区间颜色在暗色主题下保持语义一致但调整明度
- 主题偏好存储在 localStorage，key = `richman_theme_preference`（沿用现有 `richman_` 前缀惯例）
- 默认跟随系统偏好（`prefers-color-scheme`），用户可在 Settings 中手动覆盖
- K 线图和趋势图的颜色方案需适配暗色背景（lightweight-charts 支持 dark theme 配置）

## 16. 已知问题与编码阶段必须处理项

以下问题已在设计审查中识别，必须在编码阶段解决，不可跳过。

### 16.1 HTTP client 迁移

现有 `domain/http/client.ts` 将 `/api/v1` 硬编码在 `API_BASE` 中。SS8.1 设计了 `requestV1`/`requestV2`/`requestPublic` 三个函数。

处理方案：将 `API_BASE` 拆为 host-only + 版本前缀常量。现有 `request()` 重命名为内部 `_request()`。所有现有 feature 的调用点从 `request()` 迁移到 `requestV1()`。逐文件替换，不可批量 sed。

### 16.2 @dr.pogodin/react-helmet barrel 豁免

新引入的 `@dr.pogodin/react-helmet` 不是 UI 组件库，不应通过 `ui-kit/eat` barrel 导出。

处理方案：在 Biome lint 规则中为 `@dr.pogodin/react-helmet` 添加直接导入豁免（或在 `domain/seo/` 中封装 Helmet 组件后统一导入）。需明确豁免规则避免 lint 报错。

### 16.3 v1/v2 决策卡片展示区分

v1 历史决策卡片的 v2 新列（action、scenarios 等）为 NULL。TRD SS17 声明"不影响只读展示"，但未定义前端如何区分 v1/v2 卡片。

处理方案：前端根据 `action` 字段是否为 null 判断卡片版本。v1 卡片继续使用现有 DecisionCardDetailPage 渲染（trend/position/catalyst + recommendation_json）；v2 卡片使用新布局（条件分支执行计划）。两套渲染逻辑在同一组件中通过条件分支实现。

### 16.4 i18n namespace 拆分评估

SS9.1 中 `app` namespace 已 14KB，v2 新增大量 key（market overview、asset detail、briefing 等）会进一步增长。

处理方案：编码阶段评估是否需要将 `app` 拆分为 `market.json`、`asset-detail.json`、`briefing.json` 等独立 namespace。拆分阈值：单个 namespace 文件超过 500 个 key。

### 16.5 涨跌颜色逻辑国际化

`getPriceChangeColor` 用正则 `/^\d{6}$/` 判定 A 股（红涨绿跌），未来港股等纯数字代码会误判。

处理方案：后端 API 响应中的 `assetType` 或 `market` 字段已有市场信息。前端据此判定颜色惯例，而非正则匹配代码格式。

### 16.6 richson 503 时的前端 error boundary

frontend TRD 未定义 richson 不可用（richman 代理返回 503）时的 UI 状态。

处理方案：asset detail page 的各 Tab 内容区域使用 TanStack Query 的 `isError` 状态展示 "数据暂时不可用，请稍后重试" + 重试按钮。Score trend chart 降级为空状态 + 提示文本。不影响页面框架和 header 渲染。

### 16.7 localStorage key 前缀统一

现有代码中 localStorage key 使用混合命名：部分无前缀（`auth_token`、`theme_mode`），部分用 `richman_` 前缀（`richman_onboarding_nudge_dismissed`）。v2 新增 key 统一使用 `richman_` 前缀。

处理方案：v2 新增的 key 全部使用 `richman_` 前缀（已在本 TRD 中修正）。现有无前缀 key 在编码阶段逐步迁移到 `richman_` 前缀，需做 fallback 兼容读取（先读新 key，不存在时读旧 key 并迁移）。不使用 `rm_` 短前缀以保持与现有 `richman_` 惯例一致。

### 16.8 dashboard-llm-status feature 处置

现有 `features/dashboard-llm-status` 模块在 DashboardPage 中使用，v2 中 DashboardPage 改名为 ResearchBriefingPage。

处理方案：评估该模块是否仍需要。如 ResearchBriefingPage 不再需要 LLM 健康状态展示，标记 deprecated 并在 Phase 3 删除。如仍需要，迁移到 research-briefing feature 中。

### 16.9 Market Overview 页 richson 503 降级

SS16.6 定义了标的详情页 Tab 内容区域的 503 降级策略，但 Market Overview 页面的 MarketRegimeBar（SS4.2）和 EventRadarSection（SS4.6）同样依赖 richson 数据（经 richman 代理）。richson 不可用时这两个组件的 UI 状态未定义。

处理方案：MarketRegimeBar 在 `isError` 状态下隐藏整个组件（不占位），避免空条幅影响页面布局。EventRadarSection 展示 "事件数据暂时不可用" 占位文本 + 重试按钮，保留区域占位。两个组件均使用 TanStack Query 的 `retry: 2` + `retryDelay` 指数退避。

### 16.10 邮件模板 CTA 链接目标

richman-backend-v2-trd SS7.4 的 HTML 邮件模板中包含 CTA 按钮（如"录入持仓获取专属建议"），但 CTA 的 href 链接目标（前端路由 URL）未在前端 TRD 或后端 TRD 中明确定义。

处理方案：邮件中 CTA 链接统一使用前端路由的绝对 URL（从环境变量 `FRONTEND_BASE_URL` 拼接）。具体映射：简报邮件 CTA -> `/briefing`，持仓分析邮件 CTA -> `/holdings`，得分变化邮件 CTA -> `/asset/{code}`。编码阶段在邮件模板渲染时注入。
