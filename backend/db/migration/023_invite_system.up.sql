-- 023_invite_system: create rm_user_invite_codes and rm_invite_rewards tables,
-- and add login streak tracking columns to rm_users.
-- Runner wraps this file in a single BEGIN/COMMIT transaction.

-- Table: user-specific invite codes (distinct from the global rm_invite_codes table)
-- Format: "RM" + 8 uppercase alphanumeric chars (e.g. "RM3K9X7HAB")
CREATE TABLE rm_user_invite_codes (
    invite_code_id  BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    code            VARCHAR(16) NOT NULL,
    is_used         BOOLEAN NOT NULL DEFAULT FALSE,
    used_by_user_id BIGINT,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator         VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier        VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted      SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmuic_user ON rm_user_invite_codes (user_id) WHERE is_deleted = 0;
CREATE UNIQUE INDEX uq_rmuic_code ON rm_user_invite_codes (code) WHERE is_deleted = 0;
ALTER SEQUENCE rm_user_invite_codes_invite_code_id_seq RESTART WITH 100000;

-- Table: invite reward records (e.g. unlocked asset preview, extra plan refresh)
CREATE TABLE rm_invite_rewards (
    reward_id        BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL,
    reward_type      VARCHAR(32) NOT NULL,
    reward_detail    JSONB,
    source_invite_id BIGINT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator          VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier         VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted       SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmir_user ON rm_invite_rewards (user_id) WHERE is_deleted = 0;
ALTER SEQUENCE rm_invite_rewards_reward_id_seq RESTART WITH 100000;

-- Login streak tracking columns on rm_users (PRD SS6.2)
-- login_streak: consecutive login days; used to unlock additional invite codes
-- last_login_date: date-level granularity; updated on every login
ALTER TABLE rm_users ADD COLUMN login_streak    INT  NOT NULL DEFAULT 0;
ALTER TABLE rm_users ADD COLUMN last_login_date DATE;
