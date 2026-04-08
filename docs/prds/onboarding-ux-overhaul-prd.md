# Onboarding UX Overhaul 设计规范

## 背景与目标

Richman 当前的 4 步 onboarding（welcome / categories / first-holding / first-analysis）只支持单向前进。用户诉求：

1. 支持回退到上一步，保留已填状态
2. 视觉动效更生动、更有高级感
3. 允许跳过整个 onboarding 直接进入 Dashboard
4. 允许在网站内部重新发起 onboarding
5. 支持键盘左右键前进/后退

本规范定义前后端的完整改造方案。所有设计决策都经过 `docs/standards/design-review.md` 定义的 5 Pass 审查，审查产物在文末附录。

## 需求清单与决策记录

| 需求 | 决策 | 备注 |
|---|---|---|
| 跳过语义 | 后端新增 `onboarding_skipped_at` 独立字段，与 `onboarding_completed_at` 互斥 | 不用 session-only bypass，避免反复追问用户 |
| API 路由 | `POST /api/v1/onboarding/skip`，与现有 `POST /complete` 对称 | 语义明确，RESTful |
| 重入口 | Dashboard 顶部 nudge 条（skipped 时） + Settings AccountTab「重新走一遍引导」按钮（投放生产） | 双入口保证 dismiss 后仍可回归 |
| 动效方案 | framer-motion | 成熟、声明式、reduced-motion 支持完整 |
| 回退状态保留 | Context + sessionStorage | 刷新也保留，跨会话不保留 |
| 步骤圆点 | 已达步骤可点击回退 | 与左上角「← 上一步」按钮互补 |
| 跳过确认 | `antd Modal.confirm` | 防误触 |
| Welcome 视觉锚点 | 统一浅色底 + 细网格 + 慢速 radial glow + 仅 Welcome 页的自转光环 + R logo | Linear-风轻量高级感 |

## 1 数据模型变更

### 1.1 schema 迁移

新增迁移 `backend/db/migration/010_onboarding_skipped.up.sql`：

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS onboarding_skipped_at TIMESTAMPTZ NULL;
```

对应 `.down.sql`：

```sql
ALTER TABLE users DROP COLUMN IF EXISTS onboarding_skipped_at;
```

无数据回填。新字段 nullable 默认 NULL，既有用户原样不变：已完成用户 `onboarding_completed_at` 有值、新字段 NULL；未完成用户两列都 NULL。

### 1.2 互斥契约

两列语义互斥：`completed` 表示走完正常流程，`skipped` 表示用户主动放弃引导。同一用户不应同时拥有两个时间戳。这个不变量通过所有写入路径的 SQL 强制：

- `MarkOnboardingCompleted`：原子地 `SET completed_at = COALESCE(completed_at, NOW()), skipped_at = NULL`
- `MarkOnboardingSkipped`：原子地 `SET skipped_at = COALESCE(skipped_at, NOW()), completed_at = NULL`
- `ResetOnboarding`：原子地 `SET completed_at = NULL, skipped_at = NULL`

所有三个操作都在单条 UPDATE 中完成，保证互斥。

## 2 后端契约变更

### 2.1 Go 模型

`backend/internal/model/user.go` 的 `User` 结构体新增字段，紧贴 `OnboardingCompletedAt`：

```go
OnboardingCompletedAt *time.Time `json:"onboardingCompletedAt,omitempty"`
OnboardingSkippedAt   *time.Time `json:"onboardingSkippedAt,omitempty"`
```

字段名不含 `amount` / `capital` 等敏感关键词，privacy guard 天然放行。测试补一条用例覆盖确认。

### 2.2 Repo 层

`backend/internal/repo/user_repo.go` 修改：

- `userSelectColumns` 常量追加 `onboarding_skipped_at` 列
- `scanUser` 函数追加 `var skippedAt *time.Time` 扫描目标，末尾赋值 `u.OnboardingSkippedAt = skippedAt`
- `MarkOnboardingCompleted` UPDATE 子句追加 `onboarding_skipped_at = NULL`
- 新增 `MarkOnboardingSkipped(ctx, userID)` 方法，SQL 对称：`SET skipped_at = COALESCE(skipped_at, NOW()), completed_at = NULL`
- `ClearOnboardingCompleted` 改名 `ResetOnboarding` 并扩展为同时清两列；所有调用点同步改

### 2.3 Service 层

`backend/internal/service/onboarding/service.go`：

- `Status` 结构体追加字段：

```go
type Status struct {
    Completed   bool       `json:"completed"`
    CompletedAt *time.Time `json:"completedAt,omitempty"`
    Skipped     bool       `json:"skipped"`
    SkippedAt   *time.Time `json:"skippedAt,omitempty"`
}
```

- `GetStatus` 从 user 填充两对字段
- 新增 `MarkSkipped(ctx, userID) (*Status, error)` 调用 repo 的 `MarkOnboardingSkipped`，成功返回更新后的 status
- `MarkCompleted` 保持外部签名不变，底层 SQL 已经原子清 skipped_at
- `Reset(ctx, userID)` 内部改调 `repo.ResetOnboarding`

### 2.4 API 层

`backend/internal/api/v1/onboarding.go` 新增 handler：

```go
// POST /api/v1/onboarding/skip
func (h *OnboardingHandler) MarkSkipped(c *gin.Context) {
    userID := middleware.GetUserID(c)
    status, err := h.service.MarkSkipped(c.Request.Context(), userID)
    if err != nil {
        handleServiceError(c, err)
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": status})
}
```

在 `RegisterRoutes` 注册 `group.POST("/skip", h.MarkSkipped)`。

### 2.5 测试

- `onboarding_test.go` 新增 `TestOnboardingAPI_SkipEndpoint`（200 响应 + status 字段正确）
- `service_test.go` 新增 `TestMarkSkipped_Idempotent`、`TestMarkSkipped_ClearsCompleted`、`TestReset_ClearsBothColumns`
- `user_repo_test.go`（如果存在，否则放 service 测试）验证 `userSelectColumns` 与 `scanUser` 对齐
- `privacy_guard_test.go` 新增一条用例扫描更新后的 `User` 结构体确认无泄漏

## 3 前端架构

### 3.1 OnboardingStateProvider（新）

`frontend/src/pages/onboarding/state.tsx` 新建 Context Provider：

```ts
interface OnboardingState {
    categories: string[];
    holdingDraft: {
        mode: "quick" | "detail" | "screenshot";
        assetCode?: string;
        assetName?: string;
        assetType?: string;
        costPrice?: number;
        positionRatio?: number;
        quantity?: number;
    };
    reachedStep: 1 | 2 | 3 | 4;
    analysisFired: boolean;
}
```

持久化到 `sessionStorage` key `richman_onboarding_draft`。Provider mount 时：

1. 读 `useOnboardingStatus()`，如果 `completed || skipped` 为 true 直接清空 sessionStorage 并用默认 state 初始化（避免跨会话读到陈旧 draft）
2. 否则 try/catch 读 sessionStorage，失败降级为内存 state
3. 读后端已有的 `UserSettings.categories` 作为 categories 初始值，覆盖 sessionStorage 中可能存在的旧值（防止组合 #3 的用户看到空状态）
4. `holdingDraft.assetType` 如果不在 categories 列表内，自动清空（级联清理）

state 改动同步 throttled 写回 sessionStorage。清理时机：`useMarkOnboardingCompleted` / `useSkipOnboarding` / `useResetOnboarding` 的 onSuccess 清 sessionStorage key。

### 3.2 useOnboardingNav hook（新）

`frontend/src/pages/onboarding/use-onboarding-nav.ts` 提供集中的导航 API：

```ts
interface UseOnboardingNavReturn {
    prev: () => void;
    next: () => Promise<void>;
    skip: () => Promise<void>;
    jumpTo: (step: 1 | 2 | 3 | 4) => void;
    canGoNext: boolean;
    registerCanGoNext: (predicate: () => boolean) => () => void;
}
```

- `prev` 根据当前路径回退到上一步，不修改 sessionStorage
- `next` 验证 `canGoNext` 后前进；验证失败触发 shake 动画
- `skip` 弹 `Modal.confirm` → await `skipOnboarding` mutation → navigate `/dashboard`
- `jumpTo(n)` 要求 `n <= reachedStep`，否则 no-op
- `registerCanGoNext` 是每个页面注册自己校验谓词的 API，页面 unmount 时自动注销

### 3.3 OnboardingLayout 改造

`frontend/src/pages/onboarding/components/OnboardingLayout.tsx` 重写为三段式：

```
┌──────────────────────────────────────────────┐
│ [← 上一步]    [● ● ○ ○  第 2 / 4 步]   [跳过] │
│                                              │
│                  标题                        │
│                  副标题                      │
│                                              │
│         <framer AnimatePresence>             │
│            <motion.div>                      │
│              {children}                     │
│            </motion.div>                    │
│         </AnimatePresence>                   │
└──────────────────────────────────────────────┘
```

- 「← 上一步」在 step 1 隐藏
- 「跳过」在所有 step 显示，点击弹 Modal.confirm
- Step indicator 圆点可点击回退（`step <= reachedStep`），当前圆点 pulse 动画
- 全局 `keydown` 监听处理 `←` / `→` / `Esc`，`e.target.tagName` in `["INPUT","TEXTAREA","SELECT"]` 时跳过
- Modal 打开时键盘事件由 antd 接管，不与 layout 级监听冲突
- Skip Modal 触发前 `setTimeout(0)` 让当前 framer-motion 动画队列清空再打开（防止 focus trap 竞态）

### 3.4 OnboardingBackground 组件（新）

`frontend/src/pages/onboarding/components/OnboardingBackground.tsx` 提供装饰层：

- 层 1：细网格（64px 间距，`#0000000a`）
- 层 2：慢速漂移的 radial glow（90s 循环，`#00000008` 中心）
- 层 3：仅 `step === 1` 时渲染的光环 hero
  - 外层 `div` 120×120、`border-radius: 50%`
  - 用 conic-gradient + mask-composite 实现发光环效
  - framer-motion `animate={{ rotate: 360 }}` 30s 线性循环
  - 中心静态 Richman R logo（`/logo.svg`）
  - `will-change: transform` 仅在 mount 时启用，避免 GPU 层爆炸
- 所有装饰层 `useReducedMotion()` 为 true 时静态不动

### 3.5 OnboardingPageTransition（新）

`frontend/src/pages/onboarding/components/OnboardingPageTransition.tsx` 封装 AnimatePresence + 方向感知 variants：

- 前进：新页面 `x: 40 → 0, opacity: 0 → 1`，旧页面 `x: 0 → -40, opacity: 1 → 0`
- 回退：方向相反
- duration 0.35s, ease-out
- `useReducedMotion()` 时降级为 opacity-only

### 3.6 页面改造要点

**WelcomePage**：stagger 进场（标题 / 副标题 / 三张维度卡片依次出现，间隔 80ms），光环 hero 仅在这里显示。

**CategoriesPage**：4 张类型卡片 stagger fade-up，点选时 scale 1 → 1.02 + 黑边闪烁。`canGoNext = categories.length >= 1`。next 点击仍然调 `usePatchUserSettings({ categories })` 持久化（保持现有逐步 PATCH 语义）。

**FirstHoldingPage**：
- 表单字段 stagger 进场
- 检测已有持仓时的「跳过直接分析」按钮文案改为「用已有持仓直接分析 →」（与 header 的「跳过引导」区分）
- 这个按钮的行为改为 `nav.next()`（前进到 step 4），由 step 4 统一处理 `markCompleted`，不再直接调用
- 三 tab 状态（quick / detail / screenshot）作为 holdingDraft 一部分持久化

**FirstAnalysisPage**：
- 4 个步骤项 stagger 进场
- 每项完成时 checkmark draw-in（SVG path length 动画）
- `useEffect` 先读 `state.analysisFired`；为 true 时跳过后端触发，只演示动画；为 false 时触发分析并 `setState.analysisFired = true`
- 动画结束后调 `markCompleted` + navigate（保持现有时序，去掉对 `rerunAnalysis.isPending` 的 gating —— 那是之前已经做过的修复）

### 3.7 Dashboard nudge 组件（新）

`frontend/src/pages/dashboard/components/OnboardingSkippedNudge.tsx`：

- 样式：inline alert bar（非 sticky），主色黑底白字
- 条件渲染：`status.skipped && !dismissed`
- 主 CTA「开始引导」：**纯前端 navigate**（不调 mutation），`navigate("/onboarding/welcome")`
- 次要按钮「不再提示」：写 `localStorage.setItem("richman_onboarding_nudge_dismissed", "1")` 并隐藏
- 文案：「你跳过了引导流程，走一遍可以更好理解决策卡」

这里的关键决策：**重入不清 `skipped_at`**。用户重入中途想退出时，guard 因为 `skipped=true` 仍然会放行 dashboard。这样保持了「重入是临时回顾」的心智，也避免了「中途退出被反弹」的死循环。

### 3.8 OnboardingGuard 语义扩展

`frontend/src/domain/auth/onboarding-guard.tsx`：

```ts
const isBypassed = data?.completed || data?.skipped;

// Allow access to onboarding routes if:
//   - user hasn't bypassed (new user)
//   - user skipped but is revisiting from the nudge (skipped=true, not completed)
// Block access (redirect to dashboard) only when completed=true.
if (isBypassed && !isOnboardingRoute) {
    // OK, render app shell
}
if (data?.completed && isOnboardingRoute) {
    navigate(POST_ONBOARDING_HOME, { replace: true });
}
if (!isBypassed && !isOnboardingRoute) {
    navigate(ONBOARDING_ENTRY, { replace: true });
}
// skipped=true && onOnboardingRoute → allow (nudge re-entry)
```

新逻辑：completed 用户访问 onboarding 路由时弹回 dashboard；skipped 但未 completed 的用户访问 onboarding 路由被允许（支持重入）；新用户访问非 onboarding 路由弹回 welcome。

### 3.9 DashboardPage 改造

`frontend/src/pages/dashboard/DashboardPage.tsx`：

- 当前 `holdings.length === 0` 时 early return 到 EmptyHoldingsHero 的逻辑被打破
- 新结构：顶层 flex column 包裹 `<OnboardingSkippedNudge />` + 主内容区
- 主内容区内部保留原有的 early return 分支：`holdings.length === 0` 时渲染 EmptyHoldingsHero（flex: 1 自适应），否则渲染三区布局
- EmptyHoldingsHero 加次级文字链「或者先走一遍引导」，作为 dismissed + empty 组合下的 regret 路径（点击后直接 navigate `/onboarding/welcome`，guard 会因 skipped=true 放行）

### 3.10 AccountTab 改造

`frontend/src/pages/settings/tabs/AccountTab.tsx`：

- 去掉 `import.meta.env.DEV` 门控，「重新走一遍引导」按钮在所有环境可见
- 按钮行为：`Popconfirm` → `useResetOnboarding.mutateAsync()` → navigate `/onboarding/welcome`
- 文案调整：主标签「重新走一遍引导」，提示「将清空当前引导状态并从头开始，当前持仓和决策卡不受影响」

### 3.11 useResetOnboarding 扩展

`frontend/src/features/user-settings/use-reset-onboarding.ts`：

- `onSuccess` 追加 invalidate `["auth","me"]`（`useCurrentUser` 缓存）
- `onSuccess` 追加 `localStorage.removeItem("richman_onboarding_nudge_dismissed")` 清理 dismissal 标记
- `onSuccess` 追加 `sessionStorage.removeItem("richman_onboarding_draft")` 清理草稿

### 3.12 useSkipOnboarding（新）

`frontend/src/features/user-settings/use-skip-onboarding.ts`：

```ts
export function useSkipOnboarding() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => request<ApiResponse<OnboardingStatus>>("/onboarding/skip", {
            method: "POST",
        }),
        onSuccess: async () => {
            sessionStorage.removeItem("richman_onboarding_draft");
            await queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
            await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
            await queryClient.refetchQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
        },
    });
}
```

强制 `refetchQueries` 保证 navigate 前 guard 能拿到最新 skipped 状态。失败时 Modal 保持打开并 toast 错误。

## 4 视觉与动效规范

### 4.1 色彩与字体

- 背景：保持现有浅色 `#fafafa`，不引入深色 overlay
- 网格线：`#0000000a`，间距 64px
- Radial glow：`#00000008` 中心 → 透明，90s 漂移循环
- Welcome 标题：48px / 700 / letter-spacing -0.02em
- 其他步骤标题：32px / 700
- 副标题：`#595959`

### 4.2 卡片样式

所有 onboarding 内的卡片统一：

```css
border: 1px solid #0000000f;
box-shadow: 0 1px 2px #0000000a;
border-radius: 12px;
transition: all 0.2s ease-out;
```

hover 时：

```css
box-shadow: 0 4px 12px #0000000d;
transform: translateY(-2px);
```

### 4.3 动效清单

| 元素 | 动效 | 时长 |
|---|---|---|
| Page transition 前进 | `x:40→0, opacity:0→1` + 旧页 `x:0→-40` | 0.35s ease-out |
| Page transition 回退 | 方向相反 | 0.35s ease-out |
| Stagger 进场 | 子元素间隔 80ms fade-up | 0.4s each |
| 光环自转 | rotate 0→360 | 30s linear infinite |
| Radial glow 漂移 | background-position 循环 | 90s ease-in-out infinite |
| 当前圆点 pulse | scale 1→1.15→1 | 1.5s ease-in-out infinite |
| 卡片 hover | shadow + translateY | 0.2s ease-out |
| 完成 checkmark | SVG pathLength 0→1 | 0.4s ease-out |
| shake（canGoNext=false） | translateX -4/4/-4/0 | 0.3s |

### 4.4 reduced motion 降级

使用 `useReducedMotion()` hook 在 OnboardingLayout 和 OnboardingBackground 读取，降级规则：

- Page transition → opacity-only
- Stagger → 同时出现（无间隔）
- 光环自转 → 静止
- Radial glow → 静止
- Pulse → 无动画
- Shake → 无动画

## 5 依赖变更

新增 npm 包：

```bash
pnpm add framer-motion
```

导入路径：`import { motion, AnimatePresence, useReducedMotion } from "framer-motion"`

Tree-shaking：仅导入需要的 3 个符号，打包器会剥掉 gesture / layout 等模块。所有 onboarding 页面已经是 `lazy()` 加载，framer-motion 自然进入 onboarding chunk，不影响主 bundle。

## 6 测试策略

### 6.1 后端测试

- `onboarding_test.go::TestOnboardingAPI_SkipEndpoint`：POST /skip 返回 200 + status.skipped=true
- `onboarding_test.go::TestOnboardingAPI_SkipThenCompleteClearSkipped`：skip 后 complete，验证 skipped_at 被清
- `service_test.go::TestMarkSkipped_Idempotent`：连续调两次 skipped_at 时间戳不变
- `service_test.go::TestReset_ClearsBothColumns`：reset 后两列都是 NULL
- `privacy_guard_test.go`：新增 User 结构体扫描覆盖

### 6.2 前端单元测试

- `OnboardingStateProvider`：sessionStorage 读写、completed 时自动清理、categories 级联清理 holdingDraft
- `use-onboarding-nav`：prev / next / skip / jumpTo 边界、canGoNext 注册注销
- `OnboardingLayout`：键盘事件过滤（input focus 时不响应）、skip Modal 触发
- `OnboardingGuard`：skipped + onboarding 路由放行、completed + onboarding 路由反弹、新用户非 onboarding 路由弹回 welcome
- `OnboardingSkippedNudge`：dismissed localStorage、skipped=false 时不渲染
- `DashboardPage`：nudge + empty hero 共存布局

### 6.3 前端集成测试

- Full flow：welcome → categories → holding → analysis → 完成 → dashboard，sessionStorage 最终清空
- Skip flow：任意 step → 点 skip → Modal 确认 → dashboard 看到 nudge
- Back flow：categories 选两个 → next → holding 填一半 → prev 回到 categories → 验证选项回显 → next → holding 表单回显
- Keyboard flow：← / → / Esc 触发正确行为，输入框获焦时不触发
- Skip mutation failure：Modal 保持打开，toast 错误
- Nudge re-entry：dashboard 点 nudge → onboarding welcome → guard 因 skipped=true 放行

### 6.4 测试环境 setup

`frontend/src/test/setup.ts` 新增：

```ts
Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
    })),
});
```

保证 framer-motion 的 `useReducedMotion()` 在 jsdom 里不会炸。

## 7 实施顺序

依赖方向决定从后端到前端：

1. 后端 migration + model + repo + service + API + 测试（一个独立 commit，合入后前端才能对齐）
2. 前端 types + hooks（user-settings feature 层）
3. 前端 OnboardingStateProvider + useOnboardingNav
4. 前端 OnboardingLayout + OnboardingBackground + PageTransition 组件（基础设施）
5. 前端 4 个页面改造 + 测试
6. 前端 Dashboard nudge + DashboardPage 集成
7. 前端 Settings AccountTab 入口投放生产
8. 端到端冒烟：lint / test / build / 手动走 3 条主路径

## 附录 A Pass 1 状态空间表

16 个组合的分类与处理在正文 Pass 1 审查段落中列出。要点：

- `{completed=T, skipped=T, *, *}` 是 Forbidden，通过 SQL 原子清理保证
- `{completed=F, skipped=F, holdings>=1, *}` 是合法的「返回用户」状态，state provider 从后端读 categories 初始化
- `{completed=F, skipped=T, holdings=0, dismissed=T}` 的孤岛场景由 EmptyHoldingsHero 次级文字链 + Settings 入口双重兜底

## 附录 B Pass 2 文件契约影响

正文「File contract impact」表列出了所有待修改文件及其现有契约、改动类型。其中 3 条契约打破警报必须在实施时显式处理：

1. `FirstAnalysisPage.startedRef` 迁移到 sessionStorage 持久 state
2. `FirstHoldingPage` 跳过按钮语义从 markCompleted 改为 nav.next
3. `DashboardPage` early return 结构改造让 nudge 能渲染

## 附录 C Pass 3 替代路径验证

- 主路径 A 完整走通 + back / retry / cross-session / concurrent 四种替代路径
- 主路径 B 跳过 + Modal 失败重试 / navigate loop 检查
- 主路径 C 从 nudge 重入 + 「中途退出」被允许（skipped=true 放行）
- 主路径 D 从 Settings 重入 + nudge dismissed 标记清理
- 主路径 E 回退再前进 FirstAnalysisPage 不重复触发
- 主路径 F 键盘导航边界与焦点过滤

关键决策：nudge / Settings 重入**不清 skipped_at**，只依赖 `DELETE /onboarding`（走 Settings 入口时）清两列；nudge 入口纯前端 navigate 保留 skipped=true，允许用户中途退出不被反弹。

## 附录 D Pass 4 Pre-mortem 缓解

| 潜在 bug | 严重度 | 防御 |
|---|---|---|
| framer-motion 与 Modal focus trap 冲突 | 中 | skip Modal 触发前 setTimeout(0) 让动画队列清空 |
| sessionStorage hydration 污染 | 中 | Provider mount 检查 completed/skipped，任一为 true 先清 sessionStorage |
| `MarkOnboardingCompleted` SQL scan 列数不同步 | 高 | 实施时严格同时改 userSelectColumns 和 scanUser，加集成测试 |
| nudge + hero 滚动穿透 | 低 | DashboardPage flex column + hero `flex:1` 自适应 |
| `useReducedMotion` 在 jsdom 返回 undefined | 低 | setup.ts mock matchMedia 返回 matches:false |
| skip mutation 失败无反馈 | 中 | skip handler try/catch 内 throw 让 Modal 保持打开 + toast |
| 两个 tab 并发 onboarding | 低 | 接受此 corner case，不额外处理 |

## 附录 E 补强清单总览

brainstorming 阶段累计发现并已修复的 gap / 细节共 27 项，包括：

- 5 个 Gap fix（nudge 死路、非原子清理、FirstAnalysis 双触发、nudge+hero 叠加、navigate race）
- 6 个技术细节（User 类型、useCurrentUser 失效、sessionStorage key、framer-motion 路径、reduced-motion 实现、光环 CSS mask）
- 4 个小洞（nudge copy、skip loading、categories per-step PATCH 幂等、010 migration 编号）
- 12 个 Pass 1-4 发现（状态组合、FirstHoldingPage 按钮文案、useResetOnboarding 失效范围、语义拆分、sessionStorage 污染、Modal 焦点竞态、RETURNING 列数、hero flex、matchMedia mock、shake 动画、跨 tab 并发、EmptyHoldingsHero 次级链）
