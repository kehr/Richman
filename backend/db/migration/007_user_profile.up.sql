-- Add onboarding / profile fields to users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS total_capital_cny DECIMAL(18,2) NULL,
    ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS risk_preference VARCHAR(16) NOT NULL DEFAULT 'neutral',
    ADD COLUMN IF NOT EXISTS categories JSONB NOT NULL DEFAULT '[]';

-- Restrict risk_preference to enum values. Drop first to keep migration idempotent.
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_risk_preference;
ALTER TABLE users
    ADD CONSTRAINT chk_users_risk_preference
    CHECK (risk_preference IN ('conservative', 'neutral', 'aggressive'));
