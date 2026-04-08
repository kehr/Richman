# Onboarding UX Overhaul TRD

## 0. 文档定位

本文档是 `docs/prds/onboarding-ux-overhaul-prd.md` 的技术架构设计文档，承载代码级细节：数据结构、接口签名、SQL 模式、组件 props、hook 契约、动效参数。PRD 侧重交互规格和决策记录，本 TRD 侧重实现接口和数据流。

技术焦点 6 个：

1. 互斥的 onboarding 状态字段（`onboarding_completed_at` / `onboarding_skipped_at`）
2. 后端 service 方法的原子写入契约
3. 前端 `OnboardingStateProvider` 的 sessionStorage 持久化与级联清理
4. `useOnboardingNav` 统一导航 hook 的 canGoNext 注册机制
5. framer-motion 组件封装（Background / PageTransition / Layout）
6. OnboardingGuard 三态放行逻辑（new / completed / skipped）

与既有代码的集成原则：

- 复用 `onboarding` service 的既有 `MarkCompleted` / `GetStatus` / `Reset` 方法形态
- 沿用 `usePatchUserSettings` 的 mutation hook 范式
- 复用 `useOnboardingStatus` 作为 guard 和 nudge 的唯一数据源
- 新增的 `OnboardingStateProvider` 与 `OnboardingLayout` 是完全前端新增，后端无感知

## 1. 架构总览

### 1.1 后端变更模块

```
backend/
├── db/migration/
│   ├── 010_onboarding_skipped.up.sql      # 新增字段
│   └── 010_onboarding_skipped.down.sql
└── internal/
    ├── model/
    │   └── user.go                         # User 结构体追加 OnboardingSkippedAt
    ├── repo/
    │   └── user_repo.go                    # userSelectColumns 追加列、新增 MarkOnboardingSkipped、ClearOnboardingCompleted 改名 ResetOnboarding
    ├── service/onboarding/
    │   └── service.go                      # Status 追加 Skipped/SkippedAt、新增 MarkSkipped
    └── api/v1/
        └── onboarding.go                   # 新增 POST /skip handler
```

### 1.2 前端变更模块

```
frontend/src/
├── domain/
│   └── auth/
│       ├── types.ts                        # User 追加 onboardingSkippedAt
│       └── onboarding-guard.tsx            # 三态放行逻辑
├── features/
│   └── user-settings/
│       ├── types.ts                        # OnboardingStatus 追加 skipped/skippedAt
│       ├── api.ts                          # skipOnboarding 函数
│       ├── index.ts                        # 导出新 hook
│       ├── use-reset-onboarding.ts         # onSuccess 扩展
│       └── use-skip-onboarding.ts          # 新
├── pages/
│   ├── onboarding/
│   │   ├── state.tsx                       # 新：OnboardingStateProvider + useOnboardingState
│   │   ├── use-onboarding-nav.ts           # 新：useOnboardingNav hook
│   │   ├── components/
│   │   │   ├── OnboardingLayout.tsx        # 重写：三段式 header + 键盘 + skip Modal
│   │   │   ├── OnboardingBackground.tsx    # 新：grid + glow + ring hero
│   │   │   ├── OnboardingPageTransition.tsx # 新：AnimatePresence 包装
│   │   │   └── StepIndicator.tsx           # 改：可点击 + pulse
│   │   ├── WelcomePage.tsx                 # 接入 nav + stagger
│   │   ├── CategoriesPage.tsx              # 接入 nav + stagger + state
│   │   ├── FirstHoldingPage.tsx            # 接入 nav + 按钮语义修正
│   │   └── FirstAnalysisPage.tsx           # analysisFired 持久化
│   ├── dashboard/
│   │   ├── DashboardPage.tsx               # flex 重构容纳 nudge
│   │   └── components/
│   │       ├── OnboardingSkippedNudge.tsx  # 新
│   │       └── EmptyHoldingsHero.tsx       # 次级文字链
│   └── settings/
│       └── tabs/AccountTab.tsx             # 重入 CTA 投放生产
├── routes.tsx                              # OnboardingShell 挂载 StateProvider
└── test/setup.ts                           # matchMedia mock
```

### 1.3 依赖新增

- `framer-motion` (package.json 的 `dependencies`，lazy chunk)

## 2. 数据模型扩展

### 2.1 Schema 迁移

`backend/db/migration/010_onboarding_skipped.up.sql`：

```sql
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS onboarding_skipped_at TIMESTAMPTZ NULL;
```

`010_onboarding_skipped.down.sql`：

```sql
ALTER TABLE users DROP COLUMN IF EXISTS onboarding_skipped_at;
```

字段 nullable 默认 NULL，无数据回填。

### 2.2 互斥契约

两列语义互斥：同一用户不可同时持有 `onboarding_completed_at` 和 `onboarding_skipped_at`。这个不变量通过所有 3 个写入路径的 SQL 原子性保证：

| 操作 | SQL 形态 |
|---|---|
| MarkOnboardingCompleted | `SET completed_at = COALESCE(completed_at, NOW()), skipped_at = NULL` |
| MarkOnboardingSkipped | `SET skipped_at = COALESCE(skipped_at, NOW()), completed_at = NULL` |
| ResetOnboarding | `SET completed_at = NULL, skipped_at = NULL` |

所有 3 个 UPDATE 在单条语句内完成，无分步写入。幂等：重复调同一操作时间戳不变。

### 2.3 Go 模型扩展

`backend/internal/model/user.go`：

```go
type User struct {
    UserID                int64      `json:"userId"`
    Email                 string     `json:"email"`
    // ... existing fields ...
    OnboardingCompletedAt *time.Time `json:"onboardingCompletedAt,omitempty"`
    OnboardingSkippedAt   *time.Time `json:"onboardingSkippedAt,omitempty"`
    // ... existing fields ...
}
```

字段名不含 `amount` / `capital` 等敏感关键词，privacy guard 天然放行。

## 3. 后端契约

### 3.1 Repo 层

`backend/internal/repo/user_repo.go` 变更：

**userSelectColumns 常量**追加新列：

```go
const userSelectColumns = `user_id, email, password_hash, role, plan_id,
    risk_preference, total_capital_cny, onboarding_completed_at,
    onboarding_skipped_at, categories, created_at, updated_at`
```

**scanUser 函数**追加扫描目标：

```go
func scanUser(row pgx.Row, u *model.User) error {
    var (
        totalCap      decimal.NullDecimal
        completedAt   *time.Time
        skippedAt     *time.Time
        categoriesRaw []byte
    )
    if err := row.Scan(
        &u.UserID, &u.Email, &u.PasswordHash, &u.Role, &u.PlanID,
        &u.RiskPreference, &totalCap, &completedAt, &skippedAt, &categoriesRaw,
        &u.CreatedAt, &u.UpdatedAt,
    ); err != nil {
        return err
    }
    // ... existing decimal/categories handling ...
    u.OnboardingCompletedAt = completedAt
    u.OnboardingSkippedAt = skippedAt
    return nil
}
```

**MarkOnboardingCompleted 方法**扩展：

```go
func (r *UserRepo) MarkOnboardingCompleted(ctx context.Context, userID int64) (*model.User, error) {
    var u model.User
    row := r.pool.QueryRow(ctx,
        `UPDATE users
         SET onboarding_completed_at = COALESCE(onboarding_completed_at, NOW()),
             onboarding_skipped_at = NULL,
             updated_at = NOW()
         WHERE user_id = $1 AND is_deleted = 0
         RETURNING `+userSelectColumns,
        userID,
    )
    return &u, scanUser(row, &u)
}
```

**新增 MarkOnboardingSkipped 方法**：

```go
func (r *UserRepo) MarkOnboardingSkipped(ctx context.Context, userID int64) (*model.User, error) {
    var u model.User
    row := r.pool.QueryRow(ctx,
        `UPDATE users
         SET onboarding_skipped_at = COALESCE(onboarding_skipped_at, NOW()),
             onboarding_completed_at = NULL,
             updated_at = NOW()
         WHERE user_id = $1 AND is_deleted = 0
         RETURNING `+userSelectColumns,
        userID,
    )
    return &u, scanUser(row, &u)
}
```

**ClearOnboardingCompleted 改名 ResetOnboarding 并扩展语义**：

```go
func (r *UserRepo) ResetOnboarding(ctx context.Context, userID int64) (*model.User, error) {
    var u model.User
    row := r.pool.QueryRow(ctx,
        `UPDATE users
         SET onboarding_completed_at = NULL,
             onboarding_skipped_at = NULL,
             updated_at = NOW()
         WHERE user_id = $1 AND is_deleted = 0
         RETURNING `+userSelectColumns,
        userID,
    )
    return &u, scanUser(row, &u)
}
```

### 3.2 Service 层

`backend/internal/service/onboarding/service.go` 的 `Status` 扩展：

```go
type Status struct {
    Completed   bool       `json:"completed"`
    CompletedAt *time.Time `json:"completedAt,omitempty"`
    Skipped     bool       `json:"skipped"`
    SkippedAt   *time.Time `json:"skippedAt,omitempty"`
}
```

**GetStatus 实现**：

```go
func (s *Service) GetStatus(ctx context.Context, userID int64) (*Status, error) {
    u, err := s.userRepo.GetUserByID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }
    if u == nil {
        return nil, model.ErrNotFound
    }
    return &Status{
        Completed:   u.OnboardingCompletedAt != nil,
        CompletedAt: u.OnboardingCompletedAt,
        Skipped:     u.OnboardingSkippedAt != nil,
        SkippedAt:   u.OnboardingSkippedAt,
    }, nil
}
```

**MarkSkipped 新方法**：

```go
func (s *Service) MarkSkipped(ctx context.Context, userID int64) (*Status, error) {
    u, err := s.userRepo.MarkOnboardingSkipped(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("mark skipped: %w", err)
    }
    if u == nil {
        return nil, model.ErrNotFound
    }
    return statusFromUser(u), nil
}
```

**Reset 调用链更新**：

```go
func (s *Service) Reset(ctx context.Context, userID int64) (*Status, error) {
    u, err := s.userRepo.ResetOnboarding(ctx, userID)
    // ... rest stays identical ...
}
```

### 3.3 API 层

`backend/internal/api/v1/onboarding.go` 路由注册：

```go
func (h *OnboardingHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
    group := rg.Group("/onboarding", authMiddleware)
    group.GET("", h.GetStatus)
    group.POST("/complete", h.MarkCompleted)
    group.POST("/skip", h.MarkSkipped)     // 新
    group.DELETE("", h.Reset)
}
```

**MarkSkipped handler**：

```go
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

Response body 风格对齐既有 `{"data": {...}}` 包裹。HTTP 200 (非 201，因为不是创建资源)。

## 4. 前端状态管理

### 4.1 OnboardingState 数据结构

`frontend/src/pages/onboarding/state.tsx`：

```ts
export interface OnboardingState {
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

const DEFAULT_STATE: OnboardingState = {
    categories: [],
    holdingDraft: { mode: "quick" },
    reachedStep: 1,
    analysisFired: false,
};

const STORAGE_KEY = "richman_onboarding_draft";
```

### 4.2 Provider 初始化顺序

```
1. useOnboardingStatus() 读后端状态
   ├─ completed === true 或 skipped === true
   │    → sessionStorage.removeItem(STORAGE_KEY)
   │    → 用 DEFAULT_STATE 初始化
   └─ 否则
        → try { sessionStorage.getItem(STORAGE_KEY) }
          catch { 降级为 DEFAULT_STATE }
2. useUserSettings() 读后端 categories
   └─ 与 state.categories 不一致 → 以后端为准并写回 state
3. 订阅 state.categories 变化
   └─ holdingDraft.assetType 不在新 categories 时 → 清空 holdingDraft 的 asset-* 字段
4. state 变更 → throttled (500ms) sessionStorage.setItem
```

### 4.3 Context 导出

```ts
interface OnboardingStateContextValue {
    state: OnboardingState;
    update: (patch: Partial<OnboardingState>) => void;
    updateHoldingDraft: (patch: Partial<OnboardingState["holdingDraft"]>) => void;
    clear: () => void;
}

export const OnboardingStateContext = createContext<OnboardingStateContextValue | null>(null);

export function useOnboardingState() {
    const ctx = useContext(OnboardingStateContext);
    if (!ctx) throw new Error("useOnboardingState must be used inside OnboardingStateProvider");
    return ctx;
}
```

### 4.4 useOnboardingNav 契约

`frontend/src/pages/onboarding/use-onboarding-nav.ts`：

```ts
export type OnboardingStep = 1 | 2 | 3 | 4;

export interface UseOnboardingNavReturn {
    currentStep: OnboardingStep;
    reachedStep: OnboardingStep;
    canGoNext: boolean;
    prev: () => void;
    next: () => Promise<void>;
    skip: () => Promise<void>;
    jumpTo: (step: OnboardingStep) => void;
    registerCanGoNext: (predicate: () => boolean) => () => void;
}

const STEP_PATHS: Record<OnboardingStep, string> = {
    1: "/onboarding/welcome",
    2: "/onboarding/categories",
    3: "/onboarding/first-holding",
    4: "/onboarding/first-analysis",
};
```

**行为规范**：

- `prev()`：`currentStep > 1` 时 `navigate(STEP_PATHS[currentStep - 1], { replace: true })`，否则 no-op
- `next()`：遍历所有注册的 `canGoNext` 谓词，全部 true 才前进；否则 dispatch 自定义事件 `onboarding:shake` 供 OnboardingLayout 触发 shake 动画
- `skip()`：调用 `useSkipOnboarding().mutateAsync()`，成功后 `navigate("/dashboard")`，失败 toast
- `jumpTo(step)`：`step > reachedStep` 时 no-op，否则 `navigate(STEP_PATHS[step], { replace: true })`
- `registerCanGoNext`：返回 cleanup 函数，页面 unmount 时自动注销

**canGoNext 聚合**：多个页面同时注册时，只有当前活跃页面的谓词生效（通过 ref 跟踪活跃注册）。

## 5. 前端布局与动效组件

### 5.1 OnboardingLayout 三段式

```tsx
interface OnboardingLayoutProps {
    currentStep: OnboardingStep;
    title: string;
    description?: string;
    children: ReactNode;
}

// 结构：
<div style={{ position: "relative", minHeight: "100vh" }}>
    <OnboardingBackground currentStep={currentStep} />
    <div style={{ position: "relative", zIndex: 1 }}>
        <HeaderBar>
            <BackButton />       {/* currentStep > 1 时可见 */}
            <StepIndicator />     {/* 可点击，当前 pulse */}
            <SkipLink />          {/* 触发 skip Modal */}
        </HeaderBar>
        <TitleArea title={title} description={description} />
        <OnboardingPageTransition stepKey={currentStep} direction={direction}>
            {children}
        </OnboardingPageTransition>
    </div>
</div>
```

**键盘事件处理**：

```ts
useEffect(() => {
    const handler = (e: KeyboardEvent) => {
        const target = e.target as HTMLElement;
        if (["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)) return;
        if (e.key === "ArrowLeft") nav.prev();
        else if (e.key === "ArrowRight") nav.next();
        else if (e.key === "Escape") nav.skip();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
}, [nav]);
```

**skip 确认 Modal**：

```ts
const handleSkip = () => {
    setTimeout(() => {
        Modal.confirm({
            title: "确定跳过引导？",
            content: "跳过后可以在 Dashboard 顶部或 Settings 随时重新开始。",
            okText: "确定跳过",
            cancelText: "继续引导",
            onOk: async () => {
                try {
                    await skipMutation.mutateAsync();
                    navigate("/dashboard", { replace: true });
                } catch (err) {
                    message.error("跳过失败，请稍后重试");
                    throw err; // 阻止 Modal 关闭
                }
            },
        });
    }, 0); // 让 framer-motion 动画队列先清空
};
```

### 5.2 OnboardingBackground

```tsx
interface OnboardingBackgroundProps {
    currentStep: OnboardingStep;
}

// 三层 DOM：
<div style={{ position: "fixed", inset: 0, pointerEvents: "none", zIndex: 0 }}>
    <GridLayer />          {/* 64px 间距，#0000000a 线条 */}
    <GlowLayer />          {/* radial-gradient 漂移 90s */}
    {currentStep === 1 && <RingHero />}  {/* 仅 Welcome */}
</div>
```

**GridLayer CSS**：

```ts
const gridStyle: CSSProperties = {
    position: "absolute",
    inset: 0,
    backgroundImage: `
        linear-gradient(to right, #0000000a 1px, transparent 1px),
        linear-gradient(to bottom, #0000000a 1px, transparent 1px)
    `,
    backgroundSize: "64px 64px",
};
```

**GlowLayer framer-motion**：

```tsx
<motion.div
    style={{
        position: "absolute",
        inset: 0,
        background: "radial-gradient(circle at 50% 50%, #00000008, transparent 60%)",
    }}
    animate={reducedMotion ? undefined : {
        backgroundPosition: ["50% 50%", "55% 45%", "45% 55%", "50% 50%"],
    }}
    transition={{ duration: 90, repeat: Infinity, ease: "easeInOut" }}
/>
```

**RingHero 关键 CSS**（conic-gradient + mask-composite）：

```ts
const ringStyle: CSSProperties = {
    width: 120,
    height: 120,
    borderRadius: "50%",
    background: "conic-gradient(from 0deg, #000 0deg, transparent 120deg, transparent 360deg)",
    mask: "radial-gradient(circle, transparent 58px, #000 60px)",
    WebkitMask: "radial-gradient(circle, transparent 58px, #000 60px)",
    willChange: "transform",
};

<motion.div
    style={ringStyle}
    animate={reducedMotion ? undefined : { rotate: 360 }}
    transition={{ duration: 30, repeat: Infinity, ease: "linear" }}
/>
```

### 5.3 OnboardingPageTransition

```tsx
interface OnboardingPageTransitionProps {
    stepKey: string | number;
    direction: "forward" | "backward";
    children: ReactNode;
}

const variants = {
    forward: {
        initial: { x: 40, opacity: 0 },
        animate: { x: 0, opacity: 1 },
        exit: { x: -40, opacity: 0 },
    },
    backward: {
        initial: { x: -40, opacity: 0 },
        animate: { x: 0, opacity: 1 },
        exit: { x: 40, opacity: 0 },
    },
    reduced: {
        initial: { opacity: 0 },
        animate: { opacity: 1 },
        exit: { opacity: 0 },
    },
};

export function OnboardingPageTransition({ stepKey, direction, children }: OnboardingPageTransitionProps) {
    const reducedMotion = useReducedMotion();
    const v = reducedMotion ? variants.reduced : variants[direction];
    return (
        <AnimatePresence mode="wait">
            <motion.div
                key={stepKey}
                initial={v.initial}
                animate={v.animate}
                exit={v.exit}
                transition={{ duration: 0.35, ease: "easeOut" }}
            >
                {children}
            </motion.div>
        </AnimatePresence>
    );
}
```

### 5.4 Stagger 动效规范

所有页面的主内容区用 staggerChildren 0.08s 容器 + 子元素 fade-up：

```ts
const containerVariants = {
    hidden: { opacity: 0 },
    visible: { opacity: 1, transition: { staggerChildren: 0.08 } },
};
const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: { opacity: 1, y: 0, transition: { duration: 0.4, ease: "easeOut" } },
};
```

`useReducedMotion` 为 true 时 stagger 容器降级为 `transition: {}`，子元素 variants 降级为 `{ opacity: 0 } → { opacity: 1 }`。

## 6. 前端集成

### 6.1 前端 user-settings hook 契约

**OnboardingStatus 类型**（`features/user-settings/types.ts`）：

```ts
export interface OnboardingStatus {
    completed: boolean;
    completedAt?: string | null;
    skipped: boolean;
    skippedAt?: string | null;
}
```

**useSkipOnboarding hook**（`features/user-settings/use-skip-onboarding.ts`）：

```ts
export function useSkipOnboarding() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => request<ApiResponse<OnboardingStatus>>("/onboarding/skip", {
            method: "POST",
        }),
        onSuccess: async () => {
            try {
                sessionStorage.removeItem("richman_onboarding_draft");
            } catch {}
            await queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
            await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
            await queryClient.refetchQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
        },
    });
}
```

**useResetOnboarding onSuccess 扩展**：

```ts
onSuccess: async () => {
    try {
        sessionStorage.removeItem("richman_onboarding_draft");
        localStorage.removeItem("richman_onboarding_nudge_dismissed");
    } catch {}
    await queryClient.invalidateQueries({ queryKey: ONBOARDING_STATUS_QUERY_KEY });
    await queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    await queryClient.invalidateQueries({ queryKey: USER_SETTINGS_QUERY_KEY });
},
```

### 6.2 OnboardingGuard 三态放行

`frontend/src/domain/auth/onboarding-guard.tsx`：

```tsx
const completed = data?.completed ?? false;
const skipped = data?.skipped ?? false;
const isBypassed = completed || skipped;

useEffect(() => {
    if (isLoading || !data) return;
    // 新用户（未完成且未跳过）访问非 onboarding → 强制到 welcome
    if (!isBypassed && !isOnboardingRoute) {
        navigate(ONBOARDING_ENTRY, { replace: true });
        return;
    }
    // 已完成用户访问 onboarding 路由 → 弹回 dashboard
    if (completed && isOnboardingRoute) {
        navigate(POST_ONBOARDING_HOME, { replace: true });
        return;
    }
    // skipped=true && !completed && isOnboardingRoute → 放行（nudge 重入路径）
}, [completed, skipped, data, isLoading, isOnboardingRoute, navigate]);
```

关键区别：老逻辑只看 `completed` 一个字段，新逻辑 `skipped=true` 的用户**既允许访问 app shell**，**也允许访问 onboarding 路由**（用于重入）。

### 6.3 DashboardPage flex 容纳 nudge

`frontend/src/pages/dashboard/DashboardPage.tsx` 结构重构：

```tsx
export default function DashboardPage() {
    const holdings = useHoldings();
    return (
        <PageContainer>
            <Flex vertical gap={16}>
                <OnboardingSkippedNudge />
                {holdings.data?.length === 0 ? (
                    <div style={{ flex: 1 }}>
                        <EmptyHoldingsHero />
                    </div>
                ) : (
                    <ThreeRegionLayout />
                )}
            </Flex>
        </PageContainer>
    );
}
```

原本的 early return 被 inline 三元表达式替代，保证 nudge 永远有机会渲染。

### 6.4 OnboardingSkippedNudge 组件

```tsx
const DISMISS_KEY = "richman_onboarding_nudge_dismissed";

export function OnboardingSkippedNudge() {
    const { data: status } = useOnboardingStatus();
    const [dismissed, setDismissed] = useState(() => {
        try { return localStorage.getItem(DISMISS_KEY) === "1"; }
        catch { return false; }
    });
    const navigate = useNavigate();

    if (!status?.skipped || dismissed) return null;

    const handleRestart = () => {
        // 纯前端 navigate，不调 mutation。依赖 guard 的 skipped 放行语义
        navigate("/onboarding/welcome");
    };
    const handleDismiss = () => {
        try { localStorage.setItem(DISMISS_KEY, "1"); } catch {}
        setDismissed(true);
    };

    return (
        <Alert
            type="info"
            message="你跳过了引导流程，走一遍可以更好理解决策卡"
            action={
                <Space>
                    <Button type="primary" onClick={handleRestart}>开始引导</Button>
                    <Button type="text" onClick={handleDismiss}>不再提示</Button>
                </Space>
            }
        />
    );
}
```

### 6.5 AccountTab 重入 CTA

```tsx
const { mutateAsync: resetOnboarding, isPending } = useResetOnboarding();
const navigate = useNavigate();

<Popconfirm
    title="确认重新走引导吗？"
    description="将清空当前引导状态并从头开始。当前持仓和决策卡不受影响。"
    onConfirm={async () => {
        try {
            await resetOnboarding();
            navigate("/onboarding/welcome");
        } catch {
            message.error("重置失败，请稍后重试");
        }
    }}
>
    <Button loading={isPending}>重新走一遍引导</Button>
</Popconfirm>
```

## 7. 非目标

以下内容明确不在本次 TRD 范围：

- 重写 `usePatchUserSettings` 的 mutation 错误处理策略（使用既有的）
- 修改 `useOnboardingStatus` 的 query key 或 staleTime
- 引入新的动效库（只用 framer-motion）
- 迁移 onboarding 以外的页面到新的动效规范
- 后端 analysis 触发逻辑改造（step14 只改前端 `analysisFired` 持久化，不改 `/analysis/trigger` 行为）
- 多租户或团队级 onboarding 状态
- 移动端手势交互（仅键盘 + 点击）

## 8. 与 PRD 的映射

| PRD 章节 | TRD 章节 | 备注 |
|---|---|---|
| §1.1 schema migration | §2.1 | 完全对齐 |
| §1.2 互斥契约 | §2.2 | TRD 提供具体 SQL |
| §2.1 Go 模型 | §2.3 | 同 |
| §2.2 Repo 层 | §3.1 | TRD 提供完整方法签名 |
| §2.3 Service 层 | §3.2 | 同 |
| §2.4 API 层 | §3.3 | 同 |
| §3.1 OnboardingStateProvider | §4.1-4.3 | TRD 细化初始化顺序 |
| §3.2 useOnboardingNav | §4.4 | TRD 定义 canGoNext 聚合规则 |
| §3.3 OnboardingLayout | §5.1 | 同 |
| §3.4 OnboardingBackground | §5.2 | TRD 给出具体 CSS |
| §3.5 OnboardingPageTransition | §5.3 | 同 |
| §3.6 页面改造 | §5.4 + 各 page step | stagger 规范抽取到 §5.4 |
| §3.7 Dashboard nudge | §6.3-6.4 | 同 |
| §3.8 Guard 扩展 | §6.2 | 同 |
| §3.9 DashboardPage 改造 | §6.3 | 同 |
| §3.10 AccountTab 改造 | §6.5 | 同 |
| §3.11 useResetOnboarding 扩展 | §6.1 | 同 |
| §3.12 useSkipOnboarding | §6.1 | 同 |
| §4 视觉规范 | §5.1-5.4 各组件内 | 拆散到各组件 |
| §5 依赖变更 | §1.3 | 同 |
| §6 测试策略 | 分布在各 step plan | TRD 不含测试用例 |
| §7 实施顺序 | plan 总述文件 | 无重复 |

PRD 的附录 A-E（状态空间表、文件契约表、替代路径、Pre-mortem、补强清单）保留在 PRD 中，作为决策依据和审查记录，不复制到 TRD。
