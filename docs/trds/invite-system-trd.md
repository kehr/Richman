# 邀请裂变系统 TRD

> 版本 1.0 | 关联 PRD: docs/prds/richman-prd-v2.md SS14.3 | 关联 TRD: richman-backend-v2-trd.md, frontend-v2-trd.md

## 1. 文档范围

本 TRD 覆盖 PRD SS14.3 邀请裂变机制的完整技术设计：

- 专属邀请码生成与管理
- 邀请关系追踪
- 双向奖励机制
- 分享卡片附带邀请码
- 邀请码解锁（连续登录）

不在本 TRD 范围：全局邀请码（v1 已有，通过 rm_invite_codes 表管理）、付费订阅系统。

## 2. 与 v1 邀请码系统的关系

v1 的 rm_invite_codes 表管理全局邀请码（如 `RICHMAN2026`），用于注册门控。v2 的专属邀请码是独立系统，与全局邀请码并存：

- 注册时可使用全局邀请码或专属邀请码，任一有效即可通过注册门控
- 专属邀请码额外建立邀请关系，触发双向奖励
- 全局邀请码不建立邀请关系，不触发奖励

## 3. 数据库 Schema

### 3.1 rm_user_invite_codes -- 用户专属邀请码

```sql
CREATE TABLE rm_user_invite_codes (
    invite_code_id    BIGSERIAL PRIMARY KEY,
    user_id           BIGINT NOT NULL,
    code              VARCHAR(16) NOT NULL,
    is_used           BOOLEAN NOT NULL DEFAULT FALSE,
    used_by_user_id   BIGINT,
    used_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator           VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier          VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted        SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmuic_user ON rm_user_invite_codes (user_id) WHERE is_deleted = 0;
CREATE UNIQUE INDEX uq_rmuic_code ON rm_user_invite_codes (code) WHERE is_deleted = 0;
ALTER SEQUENCE rm_user_invite_codes_invite_code_id_seq RESTART WITH 100000;
```

邀请码格式：`RM` + 8 位大写字母数字随机串（如 `RM3K9X7HAB`）。使用 crypto/rand 生成，冲突时重试。

### 3.2 rm_invite_rewards -- 奖励记录

```sql
CREATE TABLE rm_invite_rewards (
    reward_id         BIGSERIAL PRIMARY KEY,
    user_id           BIGINT NOT NULL,
    reward_type       VARCHAR(32) NOT NULL,
    reward_detail     JSONB,
    source_invite_id  BIGINT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator           VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier          VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted        SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmir_user ON rm_invite_rewards (user_id) WHERE is_deleted = 0;
ALTER SEQUENCE rm_invite_rewards_reward_id_seq RESTART WITH 100000;
```

### 3.3 rm_users 新增列（归入 migration 023）

```sql
ALTER TABLE rm_users ADD COLUMN login_streak INT NOT NULL DEFAULT 0;
ALTER TABLE rm_users ADD COLUMN last_login_date DATE;
```

`login_streak`：连续登录天数，用于解锁额外邀请码。`last_login_date`：上次登录日期（日级粒度），login 接口中更新。

## 4. 核心流程

### 4.1 注册时分配专属邀请码

用户注册成功后，系统自动生成 3 个专属邀请码：

```
流程:
1. 用户注册成功（rm_users 插入完成）
2. 在同一事务中，INSERT 3 条 rm_user_invite_codes 记录
3. code 使用 crypto/rand 生成 "RM" + 8 位随机串
4. 生成失败（UNIQUE 冲突）时重试，最多 3 次
```

### 4.2 通过专属邀请码注册

```
流程:
1. 注册请求携带 inviteCode
2. 先查 rm_invite_codes（全局邀请码），匹配则走 v1 逻辑（不建立邀请关系）
3. 不匹配则查 rm_user_invite_codes WHERE code = ? AND is_used = FALSE
4. 匹配 -- 以下步骤在同一数据库事务中执行（BEGIN...COMMIT），任一失败全部回滚，避免孤儿用户：
   a. 创建新用户（INSERT rm_users）
   b. 使用原子操作消费邀请码：`UPDATE rm_user_invite_codes SET is_used = TRUE, used_by_user_id = $1, used_at = NOW() WHERE invite_code_id = $2 AND is_used = FALSE RETURNING *`（防止并发注册消费同一码的竞态）
   c. RETURNING 为空说明已被其他请求消费，回滚事务，返回 409 "邀请码已被使用"
   d. 触发双向奖励（SS4.3）
   e. 为新用户生成 3 个专属邀请码（SS4.1）
   f. COMMIT
5. 不匹配: 返回 400 "无效邀请码"
```

### 4.3 双向奖励

邀请成功时，邀请人和被邀请人各获得一个奖励：

| 角色 | reward_type | MVP 奖励内容 |
|------|------------|-------------|
| 邀请人 | `extra_analysis_refresh` | 额外 1 次执行计划手动刷新额度 |
| 被邀请人 | `extra_analysis_refresh` | 额外 1 次执行计划手动刷新额度 |

奖励通过 rm_invite_rewards 记录。MVP 阶段奖励类型简单，后续可扩展（如预览置灰标的）。

奖励消费逻辑：持仓级分析的手动触发端点检查用户可用的 extra_analysis_refresh 奖励数量。

### 4.4 连续登录解锁邀请码

PRD SS14.3 要求"连续 7 天登录解锁 1 个新邀请码"：

使用原子 SQL 避免读-改-写竞态（多设备同时登录场景）：

```sql
UPDATE rm_users SET
  login_streak = CASE
    WHEN last_login_date = CURRENT_DATE - INTERVAL '1 day' THEN login_streak + 1
    WHEN last_login_date = CURRENT_DATE THEN login_streak
    ELSE 1
  END,
  last_login_date = CURRENT_DATE,
  updated_at = NOW()
WHERE user_id = $1
RETURNING login_streak;
```

应用层根据返回的 login_streak：如果 `login_streak % 7 == 0` 则生成 1 个新专属邀请码。

## 5. API 端点

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | /api/v2/invite/my-codes | JWT | 查询我的专属邀请码列表 |
| GET | /api/v2/invite/my-invites | JWT | 查询我邀请的人列表 |

邀请系统是 v2 新功能，API 统一放 v2 前缀，与 richman-backend-v2-trd.md 的 API 版本策略一致。

### 5.1 GET /api/v2/invite/my-codes

响应：
```json
{
  "data": {
    "codes": [
      { "code": "RM3K9X7HAB", "isUsed": false, "usedAt": null },
      { "code": "RMWP5T2NQR", "isUsed": true, "usedAt": "2026-04-10T08:00:00Z" },
      { "code": "RM8J1M4KCE", "isUsed": false, "usedAt": null }
    ],
    "totalCodes": 3,
    "usedCount": 1,
    "nextUnlockIn": 3
  }
}
```

`nextUnlockIn`：距下次连续登录解锁还需几天（7 - login_streak % 7）。

### 5.2 GET /api/v2/invite/my-invites

响应：
```json
{
  "data": {
    "invites": [
      {
        "invitedUserId": 100023,
        "invitedUserName": "张***",
        "invitedAt": "2026-04-10T08:00:00Z"
      }
    ],
    "totalInvited": 1
  }
}
```

被邀请人姓名做脱敏处理（仅显示首字 + 星号）。

## 6. 分享卡片附带邀请码

### 6.1 后端支持

标的详情页分享链接中自动嵌入用户的第一个未使用的专属邀请码：

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | /api/v2/market/{code}/share | JWT（可选） | 获取分享数据，已登录时附带邀请码 |

响应（已登录）：
```json
{
  "data": {
    "shareUrl": "https://richman.app/market/GLD?ref=RM3K9X7HAB",
    "ogTitle": "黄金分析 | 72/100 中度看涨",
    "ogDescription": "当前看涨，主要因为...",
    "inviteCode": "RM3K9X7HAB"
  }
}
```

未登录时 `inviteCode` 为 null，`shareUrl` 不含 `ref` 参数。

### 6.2 前端处理

注册页检查 URL 中的 `ref` 参数，自动填充到邀请码输入框：

```typescript
// RegisterPage
const searchParams = useSearchParams();
const refCode = searchParams.get("ref");
// 如果 refCode 存在，自动填入邀请码字段
```

## 7. 前端页面

### 7.1 设置页 - 我的邀请

在 SettingsPage 中新增"我的邀请"区块：

```
InviteSection
├── InviteCodeList            # 我的专属邀请码（可复制）
│   ├── CodeItem              # 单个邀请码 + 状态（已用/可用）+ 复制按钮
│   └── UnlockProgress        # "再连续登录 X 天解锁新邀请码"
└── InvitedUserList           # 我邀请的人（列表）
```

### 7.2 分享功能集成

标的详情页"分享"按钮复制链接时，已登录用户自动附带 `ref` 参数。

## 8. Service 层

### 8.1 InviteService

```go
type InviteService struct {
    inviteCodeRepo *repo.UserInviteCodeRepo
    inviteRewardRepo *repo.InviteRewardRepo
    userRepo *repo.UserRepo
    logger *zap.Logger
}

// GenerateCodesForUser generates N invite codes for a user.
func (s *InviteService) GenerateCodesForUser(ctx context.Context, userID int64, count int) error

// UseInviteCode validates and consumes a personal invite code during registration.
func (s *InviteService) UseInviteCode(ctx context.Context, code string, newUserID int64) error

// GetMyCodes returns all invite codes for the authenticated user.
func (s *InviteService) GetMyCodes(ctx context.Context, userID int64) ([]model.UserInviteCode, error)

// GetMyInvites returns users invited by the authenticated user.
func (s *InviteService) GetMyInvites(ctx context.Context, userID int64) ([]model.InvitedUser, error)

// GetFirstAvailableCode returns the first unused invite code for share links.
func (s *InviteService) GetFirstAvailableCode(ctx context.Context, userID int64) (string, error)

// UpdateLoginStreak updates login streak and generates new code if threshold met.
func (s *InviteService) UpdateLoginStreak(ctx context.Context, userID int64) error
```

### 8.2 与 AuthService 集成

注册流程修改：在现有 AuthService.Register 方法末尾增加两个调用：
1. `inviteService.UseInviteCode(ctx, code, newUserID)` -- 消费专属邀请码 + 触发奖励
2. `inviteService.GenerateCodesForUser(ctx, newUserID, 3)` -- 为新用户生成 3 个邀请码

登录流程修改：在现有 AuthService.Login 方法末尾增加：
1. `inviteService.UpdateLoginStreak(ctx, userID)` -- 更新连续登录天数

## 9. 数据库迁移

归入 richman migration 023（在 022 之后）：

```sql
-- 023_invite_system.up.sql

CREATE TABLE rm_user_invite_codes (
    invite_code_id    BIGSERIAL PRIMARY KEY,
    user_id           BIGINT NOT NULL,
    code              VARCHAR(16) NOT NULL,
    is_used           BOOLEAN NOT NULL DEFAULT FALSE,
    used_by_user_id   BIGINT,
    used_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator           VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier          VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted        SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmuic_user ON rm_user_invite_codes (user_id) WHERE is_deleted = 0;
CREATE UNIQUE INDEX uq_rmuic_code ON rm_user_invite_codes (code) WHERE is_deleted = 0;
ALTER SEQUENCE rm_user_invite_codes_invite_code_id_seq RESTART WITH 100000;

CREATE TABLE rm_invite_rewards (
    reward_id         BIGSERIAL PRIMARY KEY,
    user_id           BIGINT NOT NULL,
    reward_type       VARCHAR(32) NOT NULL,
    reward_detail     JSONB,
    source_invite_id  BIGINT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator           VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier          VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted        SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmir_user ON rm_invite_rewards (user_id) WHERE is_deleted = 0;
ALTER SEQUENCE rm_invite_rewards_reward_id_seq RESTART WITH 100000;

ALTER TABLE rm_users ADD COLUMN login_streak INT NOT NULL DEFAULT 0;
ALTER TABLE rm_users ADD COLUMN last_login_date DATE;
```

down 迁移：
```sql
-- 023_invite_system.down.sql
ALTER TABLE rm_users DROP COLUMN last_login_date;
ALTER TABLE rm_users DROP COLUMN login_streak;
DROP TABLE IF EXISTS rm_invite_rewards;
DROP TABLE IF EXISTS rm_user_invite_codes;
```

## 10. 目录结构

```
backend/
  internal/
    api/
      v2/
        invite.go              # NEW: invite handlers
        market.go              # MODIFIED: add share endpoint
    service/
      invite/                  # NEW
        service.go
    repo/
      user_invite_code_repo.go # NEW
      invite_reward_repo.go    # NEW
    model/
      invite.go                # NEW: UserInviteCode, InviteReward, InvitedUser
  db/
    migration/
      023_invite_system.up.sql
      023_invite_system.down.sql
    query/
      user_invite_code.sql     # NEW
      invite_reward.sql        # NEW

frontend/
  src/
    features/
      invite/                  # NEW
        api.ts
        types.ts
        use-my-codes.ts
        use-my-invites.ts
        index.ts
    pages/
      settings/
        invite-section.tsx     # NEW
```

## 11. 已知问题与编码阶段必须处理项

以下问题已在设计审查中识别，必须在编码阶段解决，不可跳过。

### 11.1 login_streak 增长无上限

`login_streak` 为 INT 类型，`login_streak % 7 == 0` 在每个 7 的倍数触发新邀请码生成。用户连续登录 700 天会生成 100+ 个邀请码，无上限。

处理方案：在应用层增加上限检查。建议两个约束：(a) 单用户邀请码总数上限 20 个（含初始 3 个）；(b) `login_streak % 7 == 0` 且当前未使用邀请码数 < 3 时才生成新码。

### 11.2 login_streak 时区边界

`CURRENT_DATE` 取决于数据库 `timezone` 设置。跨时区用户（UTC+8 vs UTC）在同一自然日可能被判定为断连。

处理方案：确保 PostgreSQL `timezone` 设置为 `Asia/Shanghai`（与目标用户一致），或在 SQL 中显式使用 `CURRENT_DATE AT TIME ZONE 'Asia/Shanghai'`。

### 11.3 used_by_user_id 悬空引用

账户删除时 `rm_user_invite_codes.used_by_user_id` 指向已删除用户。查询 join user 信息时找不到对应记录。

处理方案：richman-backend-v2-trd SS22.10 已定义处理方案。编码时确保 hard delete 流程按该方案执行：先 `UPDATE ... SET used_by_user_id = NULL`，再 DELETE 用户自身的邀请码。
