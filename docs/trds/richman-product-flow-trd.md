# Richman 产品动线 TRD

## 0. 文档定位

本文档是 `richman-product-flow-prd.md` 的技术设计文档，聚焦主功能的架构与接口设计。核心模块给出完整接口和数据结构；非核心细节只讲设计原则和实现注意点，具体实现在编码阶段完成。

本次动线重构的技术焦点有 6 个：

1. 结构化 Recommendation 数据模型（替换当前 decision_cards 的扁平 action_advice 字段）
2. 变化徽章 diff 算法（与上次分析对比后写入当前分析）
3. 截图 OCR 识别管线（LLM Vision + 置信度分级）
4. total_capital 隐私边界（用户字段 + 过滤守卫）
5. Onboarding 状态跟踪
6. 帮助页内容管理

前端 IA 重构、路由调整、页面拆分属于工程改造，不在技术难点里，放在第 7 节概述。

与现有后端代码的集成原则：
- 复用 `backend/internal/analysis` 包下的 trend/position/catalyst 计算器与 matrix 决策矩阵
- 扩展 `backend/internal/analysis/synthesis` 增加结构化 recommendation 生成
- 新增的 badge_diff、screenshot_ocr、onboarding 等为独立 service 模块

## 1. 架构总览

### 1.1 后端新增/修改模块

```
backend/internal/
├── analysis/
│   ├── synthesis/            # 现有：LLM 生成卡片文本
│   │   └── synthesizer.go    # [改] 追加结构化 recommendation 输出
│   ├── recommendation/       # [新] 结构化 Recommendation 类型和解析
│   │   └── types.go
│   └── diff/                 # [新] 徽章 diff 算法
│       └── badge.go
├── service/
│   ├── analysis/             # 现有：分析管线编排
│   │   └── service.go        # [改] 落卡前调用 diff 计算 badge_state
│   ├── decision_card/        # 现有
│   │   └── service.go        # [改] 读写新字段
│   ├── screenshot/           # [新] 截图 OCR
│   │   ├── service.go
│   │   └── prompts.go
│   ├── onboarding/           # [新] onboarding 状态管理
│   │   └── service.go
│   ├── user_settings/        # [新] 总资金 / 风险偏好
│   │   └── service.go
│   └── help/                 # [新] 帮助页内容（静态 i18n JSON）
│       └── service.go
├── api/v1/
│   ├── screenshot.go         # [新] POST /api/v1/portfolio/import-screenshot
│   ├── onboarding.go         # [新] GET/PATCH /api/v1/onboarding
│   ├── user_settings.go      # [新] GET/PATCH /api/v1/user/settings
│   ├── help.go               # [新] GET /api/v1/help/{section}
│   └── decision_card.go      # [改] 新字段响应
├── llm/
│   └── vision.go             # [新] Vision 能力抽象接口
├── model/
│   └── user_settings.go      # [新] 用户设置模型
└── repo/
    ├── user_repo.go          # [改] 新增字段读写
    └── decision_card_repo.go # [改] 新增字段读写
```

### 1.2 前端新增/修改模块

```
frontend/src/
├── pages/
│   ├── onboarding/           # [新] Onboarding 4 屏
│   │   ├── WelcomePage.tsx
│   │   ├── CategoriesPage.tsx
│   │   ├── FirstHoldingPage.tsx
│   │   └── FirstAnalysisPage.tsx
│   ├── dashboard/
│   │   └── DashboardPage.tsx # [改] 三区结构重写
│   ├── portfolio/
│   │   └── PortfolioListPage.tsx # [改] 新增截图批量导入入口
│   ├── decision-cards/
│   │   └── DecisionCardDetailPage.tsx # [改] 5 区块重写
│   ├── settings/
│   │   └── SettingsPage.tsx  # [改] 4 tab 重构
│   ├── help/                 # [新]
│   │   └── HelpPage.tsx
│   └── auth/
│       └── LoginPage.tsx     # [改] 左右双栏布局
├── features/
│   ├── decision-card/        # [新] 决策卡组件库（卡面 + 徽章 + 执行计划条）
│   │   ├── components/
│   │   │   ├── DecisionCardSummary.tsx
│   │   │   ├── ChangeBadge.tsx
│   │   │   ├── DimensionBadges.tsx
│   │   │   └── ExecutionPlanStrip.tsx
│   │   ├── api.ts
│   │   ├── useDecisionCard.ts
│   │   └── index.ts
│   ├── portfolio/            # 现有
│   │   ├── components/
│   │   │   ├── AddHoldingDrawer.tsx        # [新]
│   │   │   ├── ScreenshotImportModal.tsx   # [新]
│   │   │   └── HoldingTable.tsx            # [改]
│   │   └── ...
│   ├── onboarding/           # [新]
│   │   ├── api.ts
│   │   ├── useOnboarding.ts
│   │   └── index.ts
│   └── user-settings/        # [新]
│       ├── api.ts
│       ├── useTotalCapital.ts
│       └── index.ts
├── domain/
│   ├── auth/
│   │   └── auth-guard.tsx    # [改] 加 onboarding 守卫
│   └── money/                # [新] 金额换算 hook（设置了总资金后才有值）
│       └── useMoney.ts
└── config/
    ├── routes.ts             # [改] 新增 onboarding / help 路由、重定向规则
    └── theme.ts              # 已完成
```

### 1.3 数据库迁移清单

新增 3 个 migration：

- `006_recommendation_structured.up.sql` — decision_cards 表增加结构化字段
- `007_user_profile.up.sql` — users 表增加 total_capital_cny / onboarding_completed_at / risk_preference / categories
- `008_holding_category.up.sql` — holdings 表增加可选 category 字段（配合 onboarding 标的类型预选）

各自配套 down 文件。

## 2. 结构化 Recommendation 数据模型

### 2.1 设计目标

当前 decision_cards 表的 recommendation 字段是一个简单字符串（`hold` / `small_add` 等），action_advice 和 detailed_advice 是自由文本。新设计要求：

- 把"建议动作 + 目标仓位 + 分批执行步骤 + 止损止盈"都做成结构化字段
- 前端能根据数据模型渲染 Dashboard 卡面的执行计划摘要条和详情页的完整推理
- 支持变长 steps（无硬上限）
- 保持与现有 analysis pipeline 的向前兼容（旧卡读不到新字段时有降级）

### 2.2 数据模型（Go）

```go
// backend/internal/analysis/recommendation/types.go
package recommendation

// Action 是 5 种基础建议动作
type Action string

const (
    ActionAggressiveAdd Action = "aggressive_add"
    ActionSmallAdd      Action = "small_add"
    ActionHold          Action = "hold"
    ActionGradualReduce Action = "gradual_reduce"
    ActionControl       Action = "control_position"
)

// ActionLevel 是 action 的积极度等级，用于升降级判断
// 积极加仓 = 2, 小幅加仓 = 1, 持有 = 0, 分批减仓 = -1, 控制仓位 = -2
func (a Action) Level() int { ... }

// ExecutionType 是执行计划类型
type ExecutionType string

const (
    ExecutionOneShot ExecutionType = "one-shot" // 一次性
    ExecutionStaged  ExecutionType = "staged"   // 分批
    ExecutionMonitor ExecutionType = "monitor"  // 持有监控（无 steps，只有 stop_loss/take_profit）
)

// TriggerType 是单步触发方式
type TriggerType string

const (
    TriggerPrice TriggerType = "price"
    TriggerTime  TriggerType = "time"
    TriggerEvent TriggerType = "event"
)

// Step 描述执行计划中的一步
type Step struct {
    Order          int            `json:"order"`
    TriggerType    TriggerType    `json:"triggerType"`
    TriggerValue   string         `json:"triggerValue"`   // 展示文案，如 "≤ 4.05" / "财报后 2 周" / "Fed 议息后"
    TriggerPayload TriggerPayload `json:"triggerPayload"` // 结构化（用于前端对比 / 未来自动提醒）
    DeltaPct       float64        `json:"deltaPct"`       // 本批变动仓位（正数 = 加仓，负数 = 减仓）
    Rationale      string         `json:"rationale"`      // 详情页展示的完整推理
}

// TriggerPayload 是可选的结构化触发条件
type TriggerPayload struct {
    PriceOp     string     `json:"priceOp,omitempty"`     // "lte" | "gte"
    PriceValue  float64    `json:"priceValue,omitempty"`
    DeadlineISO *time.Time `json:"deadlineIso,omitempty"`
    EventKey    string     `json:"eventKey,omitempty"`    // 如 "fomc_meeting_2026Q2"
}

// Execution 是完整执行计划
type Execution struct {
    Type       ExecutionType `json:"type"`
    Steps      []Step        `json:"steps,omitempty"`       // staged / one-shot 时有
    StopLoss   *float64      `json:"stopLoss,omitempty"`    // monitor 时有
    TakeProfit *float64      `json:"takeProfit,omitempty"`  // monitor 时有
    ValidDays  int           `json:"validDays"`             // 本计划有效天数
}

// Recommendation 是完整的结构化建议
type Recommendation struct {
    Action            Action    `json:"action"`
    ActionLevel       int       `json:"actionLevel"`
    Label             string    `json:"label"`              // 文案，如 "小幅加仓"
    CurrentPositionPct float64  `json:"currentPositionPct"`
    TargetPositionPct float64   `json:"targetPositionPct"`
    Execution         Execution `json:"execution"`
}
```

### 2.3 数据库 schema 变更

`006_recommendation_structured.up.sql`：

```sql
-- 现有 decision_cards 表增加字段
ALTER TABLE decision_cards
    ADD COLUMN IF NOT EXISTS recommendation_json JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS action_level SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS target_position_ratio DECIMAL(10,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS badge_state VARCHAR(32) NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS confidence_delta DECIMAL(6,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS prev_card_id BIGINT NULL;

CREATE INDEX IF NOT EXISTS idx_dc_badge_state ON decision_cards (is_deleted, badge_state);
CREATE INDEX IF NOT EXISTS idx_dc_prev ON decision_cards (prev_card_id) WHERE is_deleted = 0;
```

设计说明：

- `recommendation_json` 存完整 Recommendation JSON，前端直接反序列化
- `action_level` 冗余提出来方便索引和 diff 快速比较
- `target_position_ratio` 同上
- `badge_state` 是分析完成时已计算好的徽章状态（见第 3 节）
- `confidence_delta` 是与上次比的 confidence 差值（用于"信心度波动"徽章）
- `prev_card_id` 指向此标的上一次的决策卡，方便详情页"此标的历史分析"列表和回溯调试
- 保留现有 recommendation 字符串字段不变，做只读兼容；旧卡 recommendation_json = '{}' 时前端走降级文案渲染

### 2.4 Synthesizer 变更

`backend/internal/analysis/synthesis/synthesizer.go` 扩展：

- SynthesisOutput 增加 `RecommendationJSON json.RawMessage` 字段
- 在现有 buildSynthesisPrompt 的基础上追加"输出结构化 recommendation（含执行计划）"的指令段
- 解析 LLM 响应时多解析一个 recommendation 子对象
- 降级策略：LLM 输出 recommendation_json 解析失败时，用 action level 推导一个 one-shot 默认执行计划（ActionSmallAdd → 1 步加 5%；ActionGradualReduce → 1 步减 10% 等），保证前端永远拿得到非空结构

具体 Prompt 调整不在 TRD 里定稿，在 coding 阶段基于实际 LLM 输出质量迭代。

### 2.5 API 响应

`GET /api/v1/decision-cards/:id` 响应 DTO 在现有基础上加：

- `recommendation`: 结构化对象（来自 recommendation_json）
- `actionLevel`: number
- `badgeState`: string（8 种 enum）
- `confidenceDelta`: number
- `prevCardId`: number | null

Dashboard 列表接口 `GET /api/v1/decision-cards?latest=true` 同步返回这些字段。

## 3. 变化徽章 diff 算法

### 3.1 输入输出

输入：
- `current`：本次分析生成的 recommendation + dimensions + confidence
- `previous`：该 holding 的上一次 decision_card（可能为空 = 首次分析）
- `dataSourceDegraded`：本次分析是否有数据源降级

输出：
- `badgeState` ∈ 8 种 enum（见 PRD 3.4）
- `confidenceDelta`：本次 - 上次 confidence

### 3.2 算法（伪代码）

```go
// backend/internal/analysis/diff/badge.go
package diff

type BadgeState string

const (
    BadgeDataDegraded    BadgeState = "data_degraded"
    BadgeFirstAnalysis   BadgeState = "first_analysis"
    BadgeActionUpgrade   BadgeState = "action_upgrade"
    BadgeActionDowngrade BadgeState = "action_downgrade"
    BadgeSignalFlip      BadgeState = "signal_flip"
    BadgePlanAdjust      BadgeState = "plan_adjust"
    BadgeConfidenceShift BadgeState = "confidence_shift"
    BadgeNone            BadgeState = "none"
)

// Input 是 diff 计算所需的全部字段
type Input struct {
    Current            CardSnapshot
    Previous           *CardSnapshot // nil = 首次
    DataSourceDegraded bool
}

// CardSnapshot 是参与 diff 的卡快照（精简字段）
type CardSnapshot struct {
    ActionLevel        int
    TargetPositionPct  float64
    Confidence         float64
    TrendDirection     string
    PositionDirection  string
    CatalystDirection  string
    ExecutionFingerprint string // 执行计划指纹，见下文
}

// Compute 返回 badge state 和 confidence delta
// 按优先级从高到低检测：
//   1. DataSourceDegraded → BadgeDataDegraded
//   2. Previous == nil    → BadgeFirstAnalysis
//   3. ActionLevel 变化   → BadgeActionUpgrade / BadgeActionDowngrade
//   4. 任一 dimension direction 变化且 ActionLevel 未变 → BadgeSignalFlip
//   5. ExecutionFingerprint 或 TargetPositionPct 变化 → BadgePlanAdjust
//   6. |ConfidenceDelta| >= 10 → BadgeConfidenceShift
//   7. 否则 → BadgeNone
func Compute(in Input) (BadgeState, float64) { ... }
```

### 3.3 执行计划指纹

为了判断"计划调整"，需要比较两次的 execution 是否"实质上相同"。方法：

- 对 execution 的稳定字段计算 SHA-1 指纹
- 稳定字段 = `(type, target_position_pct, stop_loss, take_profit, 每个 step 的 trigger_type + trigger_value + delta_pct)`
- `rationale` 文本不参与指纹（LLM 每次文本会不同，不算实质变化）
- 指纹写入 decision_cards 额外字段 `execution_fingerprint VARCHAR(64)`，方便索引和 diff

迁移 006 同步添加此列：`ADD COLUMN execution_fingerprint VARCHAR(64) NOT NULL DEFAULT ''`。

### 3.4 阈值常量

- `CONFIDENCE_SHIFT_THRESHOLD = 10`（±10 分以上才触发信心度波动徽章）
- `VALIDITY_DEFAULT_DAYS = 7`（execution.valid_days 默认值）

常量写在 `recommendation/types.go` 里暴露出来。

### 3.5 调用时机

`service/analysis/service.go` 在写入新卡前：

1. 查询同 holding 的最近一张 decision_card（按 analyzed_at DESC）
2. 构造 Input 调用 diff.Compute
3. 把 badge_state、confidence_delta、prev_card_id 填入新卡
4. 保存新卡

该顺序必须是事务性的，避免并发分析造成 prev 指向错位。

## 4. 截图 OCR 识别管线

### 4.1 设计目标

用户上传券商 App 持仓截图，系统识别每个标的的"名称 / 代码 / 均价成本 / 仓位比例"，前端展示为双栏对照校对表。要求：

- 识别不存储截图原文（仅分析后即丢弃或仅短期缓存）
- 每个字段带置信度，前端据此高亮低信心字段
- 失败时前端能清晰地提示用户"识别失败，请手动录入"
- LLM Vision 降级时完全不阻塞流程（截图识别本就是可选便利功能）

### 4.2 LLM Vision 抽象

`backend/internal/llm/vision.go`：

```go
package llm

import "context"

// VisionProvider 是带图能力的 LLM 抽象接口（与现有 Provider 解耦）
type VisionProvider interface {
    // AnalyzeImage 接收图像字节 + prompt，返回结构化 JSON 字符串
    AnalyzeImage(ctx context.Context, req VisionRequest) (*VisionResponse, error)
}

type VisionRequest struct {
    SystemPrompt string
    UserPrompt   string
    ImageData    []byte
    ImageMIME    string  // "image/png" / "image/jpeg"
    MaxTokens    int
    Temperature  float64
}

type VisionResponse struct {
    Content   string
    UsageHint map[string]any // token usage 等
}
```

MVP 仅实现 Claude Vision（`claude-sonnet-4-6`，原生支持图像输入）。factory.go 根据 `LLM_VISION_PROVIDER` 环境变量决定具体实现。

### 4.3 Screenshot 服务接口

```go
// backend/internal/service/screenshot/service.go
package screenshot

type Service struct { ... }

// Recognize 处理一张图并返回识别结果
// 不持久化图像，不持久化识别结果；持久化发生在后续用户确认导入时
func (s *Service) Recognize(ctx context.Context, userID int64, req RecognizeRequest) (*RecognizeResponse, error)

type RecognizeRequest struct {
    ImageData []byte
    ImageMIME string
}

// RecognizeResponse 是识别结果，与前端双栏对照 UI 直接对齐
type RecognizeResponse struct {
    Holdings      []RecognizedHolding `json:"holdings"`
    OverallStatus string              `json:"overallStatus"` // "ok" | "low_quality" | "failed"
    Warning       string              `json:"warning,omitempty"`
}

type RecognizedHolding struct {
    AssetName       Field   `json:"assetName"`
    AssetCode       Field   `json:"assetCode"`
    CostPrice       Field   `json:"costPrice"`
    PositionPct     Field   `json:"positionPct"`
    AssetTypeGuess  string  `json:"assetTypeGuess"` // 猜测的类型，不带置信度
}

// Field 是单字段的值和置信度
type Field struct {
    Value      string  `json:"value"`
    Confidence float64 `json:"confidence"` // 0.0 - 1.0
}
```

### 4.4 置信度阈值

```go
const (
    ConfidenceHigh = 0.85 // >= 0.85：无高亮，视为可靠
    ConfidenceLow  = 0.60 // < 0.60：前端直接清空字段并高亮要求用户填
                          // 0.60 <= c < 0.85：前端高亮但保留值（黄色提示）
)
```

### 4.5 API 端点

```
POST /api/v1/portfolio/import-screenshot
  Content-Type: multipart/form-data
  Body: image file (max 5 MB)
  Response: RecognizeResponse
  Auth: 需要 JWT
  限流: 每用户 10 次 / 小时
```

该接口只做识别不做持久化。前端拿到 RecognizeResponse 后，用户点"确认导入"时走现有的 `POST /api/v1/holdings` 接口批量创建。

### 4.6 降级策略

- LLM Vision 调用异常（超时 / 5xx）：返回 `overallStatus = "failed"` + warning 文案"识别服务暂时不可用，请手动录入"
- 解析响应 JSON 失败：同上
- 图像过大（> 5 MB）：413 响应 + 前端提示
- 限流触发：429 响应 + 前端提示"请稍后再试"

### 4.7 Prompt 结构（原则）

不在 TRD 定稿完整 prompt，只规定契约：

- System prompt 明确输出 JSON schema（与 RecognizeResponse 对齐）
- 用户 prompt 说明"这是一张持仓截图"并给出需要识别的字段清单
- 指示"无法识别的字段置 value 为空字符串、confidence 为 0"
- 要求"只输出 JSON，不要任何解释性文字"

Prompt 文件放在 `backend/internal/service/screenshot/prompts.go`，coding 阶段基于实际 LLM 输出质量迭代。

## 5. total_capital 隐私边界

### 5.1 字段定义

`007_user_profile.up.sql`：

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS total_capital_cny DECIMAL(18,2) NULL,
    ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS risk_preference VARCHAR(16) NOT NULL DEFAULT 'neutral',
    ADD COLUMN IF NOT EXISTS categories JSONB NOT NULL DEFAULT '[]';
```

说明：
- `total_capital_cny` NULL = 未设置，所有金额位置只显示百分比
- `risk_preference` ∈ `conservative` / `neutral` / `aggressive`，影响 weight manager 的 bias（见 5.4）
- `categories` = onboarding 时选择的类型列表，影响下次推荐 asset catalog 过滤

### 5.2 访问路径隔离（强制要求）

用三个编译期约束 + 一个运行时 lint 守卫保证 total_capital 不泄漏：

**约束 1：分析 pipeline 输入类型**

`backend/internal/service/analysis/service.go` 中所有调用 trend/position/catalyst/synthesis 的函数签名不允许接受含 total_capital 的 model 类型。解决方案：分析函数只收 `analysis.Input`，该类型不含 total_capital 字段。

**约束 2：LLM 上下文构造函数**

`synthesis.buildSynthesisPrompt` 和 `screenshot.buildPrompt` 函数签名不允许接受含 total_capital 的类型。Prompt 构造只使用百分比。

**约束 3：推送消息渲染**

`backend/internal/notification` 包下所有 adapter 的 render 函数只接受 `decision_card.PublicCardSummary` DTO，该 DTO 不含金额字段（只含百分比）。

**运行时守卫**

新增 `backend/internal/service/user_settings/privacy_guard.go`：

```go
// AssertNoCapitalLeakage 是一个运行时断言工具，接受任意 struct
// 通过反射检查 json tag 里不含 "totalCapital" 和 "amount"
// 仅在 debug 构建中启用（`-tags debug`），作为额外防护网
func AssertNoCapitalLeakage(v any) error { ... }
```

在分析完成写卡前、推送消息发送前各调用一次，测试环境开启。

### 5.3 金额换算层级

设计决策：金额换算在**后端 API 响应层完成**（非前端），理由：

- 前端多个页面都要显示金额，在 API 层一次性附加 amount 字段避免前端重复计算
- API 层能统一应用隐私守卫（"金额字段一定是 target DTO 字段，不会混进分析请求"）
- 后端单元测试更容易覆盖"设了 / 没设总资金 → 输出一致性"

响应 DTO 模式：

```go
// 所有需要显示金额的 DTO 都加一组可选字段
type DashboardCardDTO struct {
    ...
    PositionRatio       float64  `json:"positionRatio"`
    PositionAmount      *float64 `json:"positionAmount,omitempty"` // 仅总资金已设时有值
    TargetPositionRatio float64  `json:"targetPositionRatio"`
    TargetPositionAmount *float64 `json:"targetPositionAmount,omitempty"`
    UnrealizedPct       float64  `json:"unrealizedPct"`
    UnrealizedAmount    *float64 `json:"unrealizedAmount,omitempty"`
}
```

API handler 从 user 表读 total_capital_cny 后调用 `money.AttachAmounts(dto, capital)` 统一附加所有 `*Amount` 字段。该工具函数集中在 `backend/internal/service/user_settings/money.go`。

### 5.4 risk_preference 影响权重

`backend/internal/analysis/weight/manager.go` 的 Manager.Adjust 函数增加一个 RiskPreference 参数：

- `conservative`：位置维度 +5%，催化剂 -5%（在原 ±10% 范围内）
- `neutral`：不调整
- `aggressive`：催化剂 +5%，位置 -5%

这是在 LLM 权重微调基础上额外加的 bias，不替换 LLM 判断。实现细节在 coding 阶段决定。

## 6. Onboarding 状态跟踪 + 帮助页内容管理

### 6.1 Onboarding 状态

- `users.onboarding_completed_at TIMESTAMPTZ NULL`
- 完成标记在 `/onboarding/first-analysis` 的进度达到 100% 时写入（通过新接口 `PATCH /api/v1/onboarding/complete`）
- 前端 `domain/auth/auth-guard.tsx` 在 AuthGuard 之后追加 OnboardingGuard：
  - 用户已登录 but `onboarding_completed_at == null` 且当前路径不在 `/onboarding/*` → 重定向到 `/onboarding/welcome`
  - 用户已完成 but 访问 `/onboarding/*` → 重定向到 `/dashboard`
- 开发环境跳过机制：在 Settings → 账户 tab 加一个"重置 Onboarding"按钮，仅 `NEXT_PUBLIC_APP_ENV=dev` 时显示

### 6.2 帮助页内容

结论：**静态 i18n JSON 方案**。

- 内容放在 `frontend/src/i18n/help/zh-CN.json` 和 `en-US.json`
- 结构按 PRD §7.2 的 9 章节组织，每章节一个 key，value 是 markdown 字符串
- 前端 `/help` 页面用 `react-markdown` 渲染，左侧生成锚点导航
- 后端提供 `GET /api/v1/help/{section}?lang=zh-CN` 只是透传包装（MVP 直接前端静态加载即可，接口留作后续动态化空间）

不用 DB 存储的理由：
- 内容由开发者维护，不需要运行时修改
- i18n JSON 与前端代码一起版本化，与 PRD/TRD 同步改动
- 省去 CMS 后台开发成本

## 7. 前端工程改造概述

### 7.1 路由重构

`frontend/src/routes.tsx` 从 6 项路由扩展为：

```
公开：/login /register
onboarding（OnboardingGuard 内）：
    /onboarding/welcome
    /onboarding/categories
    /onboarding/first-holding
    /onboarding/first-analysis
主应用（AuthGuard + OnboardingGuard 完成后）：
    /dashboard
    /portfolio
    /portfolio/:id/transactions
    /decision-cards/:id
    /settings
    /help
兜底：* → /dashboard
```

- AnalysisPage 文件删除（对应 `/analysis` 路由取消）
- DecisionCardListPage 文件删除（对应 `/decision-cards` 列表路由取消；`/decision-cards/:id` 详情页保留）
- NotificationsPage 文件删除（内容迁移到 Settings 子 tab）

### 7.2 MainLayout 菜单改造

`frontend/src/layouts/MainLayout.tsx` 的 menuRoutes 简化为 3 项 + 底部辅助：

- 3 个顶级 menu item：Dashboard / Portfolio / Settings
- ProLayout 的 `menuFooterRender` prop 渲染帮助链接
- 用户卡继续用 actionsRender 放在顶部右侧（现状保留）

### 7.3 features 层拆分

每个 feature 严格遵守现有约束（`docs/standards/frontend.md`）：
- 所有 antd 导入通过 `@/ui-kit/eat` barrel
- features 之间不互相依赖
- feature 内部文件不被 pages 直接依赖，仅通过 `index.ts` barrel

**新增的 feature：**

- `features/decision-card/`：决策卡组件库（摘要卡 / 徽章 / 维度 badge / 执行计划条 / 详情页区块）
- `features/onboarding/`：Onboarding 流程的 api 和 hooks
- `features/user-settings/`：总资金 / 风险偏好 / 语言 / 时区等设置

**改造的 feature：**

- `features/portfolio/`：增加 AddHoldingDrawer、ScreenshotImportModal 两个组件；保留现有 holding CRUD hooks

### 7.4 金额换算 hook

```ts
// frontend/src/domain/money/useMoney.ts
export function useMoney() {
  const { data: settings } = useUserSettings();
  const hasCapital = settings?.totalCapitalCny != null;

  // 如果后端已经附加 amount 字段，直接显示；否则 fallback
  return {
    hasCapital,
    format: (pct: number, amount?: number | null) =>
      hasCapital && amount != null
        ? `${pct}% · ¥${amount.toLocaleString()}`
        : `${pct}%`,
    formatAmountOnly: (amount?: number | null) =>
      hasCapital && amount != null
        ? `¥${amount.toLocaleString()}`
        : null,
  };
}
```

所有需要渲染"百分比 + 金额"的组件调用此 hook，而不是各自重复逻辑。

## 8. 非目标

以下内容本 TRD 明确不覆盖：

- 推送渠道 adapter 的重构（保留现有实现，只把配置页归入 Settings tab）
- Analysis pipeline 内部算法优化（继续用现有的 trend/position/catalyst 计算器）
- LLM prompt 的 A/B 测试框架（不在 MVP 范围）
- 多币种总资金扩展（MVP 仅 CNY，数据模型预留扩展但不实现）
- 截图 OCR 的自动重试 / 并发 / 缓存（MVP 一次调用一次响应，失败由用户重传）
- 帮助页的全文搜索（MVP 只做章节锚点导航）

## 9. 与 PRD 的映射

| PRD 章节 | TRD 对应章节 |
|---|---|
| §1 信息架构重构 | §7.1 路由重构 + §7.2 菜单改造 |
| §2 入口与 Onboarding | §6.1 Onboarding 状态 + §7 前端改造 |
| §3 Dashboard | §2 Recommendation 模型 + §3 徽章算法 + §5.3 金额换算 + §7.3 decision-card feature |
| §4 Portfolio | §4 截图 OCR + §7.3 portfolio feature 改造 |
| §5 决策卡详情页 | §2 Recommendation 模型 + §3 徽章算法 + §7.3 decision-card feature |
| §6 Settings | §5 total_capital + §5.4 risk_preference + §7.3 user-settings feature |
| §7 帮助页 | §6.2 帮助页内容 |
| §8 总资金功能 | §5 total_capital 隐私边界 |
| §9 菜单最终形态 | §7.1 §7.2 |
