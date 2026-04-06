-- Users table
CREATE TABLE IF NOT EXISTS users (
    user_id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'user',
    plan_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email ON users (email) WHERE is_deleted = 0;

-- Plans table
CREATE TABLE IF NOT EXISTS plans (
    plan_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    max_holdings INT NOT NULL DEFAULT 5,
    max_daily_analysis INT NOT NULL DEFAULT 10,
    max_push_channels INT NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);

-- Invite codes table
CREATE TABLE IF NOT EXISTS invite_codes (
    invite_code_id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    max_uses INT NOT NULL DEFAULT 1,
    used_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted SMALLINT NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_invite_codes_code ON invite_codes (code) WHERE is_deleted = 0;

-- Seed default plan
INSERT INTO plans (plan_id, name, max_holdings, max_daily_analysis, max_push_channels)
VALUES (100000, 'invite', 5, 10, 3)
ON CONFLICT DO NOTHING;

-- Seed default invite code
INSERT INTO invite_codes (invite_code_id, code, max_uses)
VALUES (90000, 'RICHMAN2026', 100)
ON CONFLICT DO NOTHING;
