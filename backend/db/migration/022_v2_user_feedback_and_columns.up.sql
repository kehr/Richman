-- 022_v2_user_feedback_and_columns: add rm_user_feedback table, extend rm_users and
-- rm_decision_cards with v2 columns.
-- Runner wraps this file in a single BEGIN/COMMIT transaction.

-- New table: record user ratings for analysis quality tracking (PRD SS6.3)
CREATE TABLE rm_user_feedback (
    feedback_id       BIGSERIAL PRIMARY KEY,
    user_id           BIGINT NOT NULL,
    asset_analysis_id BIGINT NOT NULL,
    rating            VARCHAR(16) NOT NULL,
    comment           TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    creator           VARCHAR(64) NOT NULL DEFAULT 'system',
    modifier          VARCHAR(64) NOT NULL DEFAULT 'system',
    is_deleted        SMALLINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_rmuf_user ON rm_user_feedback (user_id) WHERE is_deleted = 0;
ALTER SEQUENCE rm_user_feedback_feedback_id_seq RESTART WITH 100000;

-- risk_preference already exists (migration 007) with default 'neutral' and
-- CHECK (conservative/neutral/aggressive). v2 changes the allowed set to
-- conservative/moderate/aggressive and updates existing 'neutral' rows.
ALTER TABLE rm_users DROP CONSTRAINT IF EXISTS chk_users_risk_preference;
ALTER TABLE rm_users ALTER COLUMN risk_preference SET DEFAULT 'moderate';
UPDATE rm_users SET risk_preference = 'moderate' WHERE risk_preference = 'neutral';
ALTER TABLE rm_users ADD CONSTRAINT chk_users_risk_preference
    CHECK (risk_preference IN ('conservative', 'moderate', 'aggressive'));

-- Email push opt-out flag (PRD SS7.1)
ALTER TABLE rm_users ADD COLUMN email_push_enabled BOOLEAN NOT NULL DEFAULT TRUE;

-- Subscription tier preset (PRD SS14.2). Named subscription_tier to avoid
-- confusion with the existing plan_id FK column (known issue G2.13).
ALTER TABLE rm_users ADD COLUMN subscription_tier VARCHAR(16) NOT NULL DEFAULT 'invite';

-- Disclaimer acceptance timestamp (PRD SS12). NULL for users registered before v2.
ALTER TABLE rm_users ADD COLUMN disclaimer_accepted_at TIMESTAMPTZ;

-- v2 decision card columns (PRD SS8.1, SS17).
-- v1 columns are preserved for historical read-only access (PRD SS3.9).
ALTER TABLE rm_decision_cards ADD COLUMN action              VARCHAR(32);
ALTER TABLE rm_decision_cards ADD COLUMN action_label        VARCHAR(128);
ALTER TABLE rm_decision_cards ADD COLUMN scenarios           JSONB;
ALTER TABLE rm_decision_cards ADD COLUMN stop_loss           DECIMAL(20,6);
ALTER TABLE rm_decision_cards ADD COLUMN take_profit         DECIMAL(20,6);
ALTER TABLE rm_decision_cards ADD COLUMN valid_days          INT;
ALTER TABLE rm_decision_cards ADD COLUMN concentration_level   VARCHAR(16);
ALTER TABLE rm_decision_cards ADD COLUMN concentration_message TEXT;
ALTER TABLE rm_decision_cards ADD COLUMN default_action      TEXT;
ALTER TABLE rm_decision_cards ADD COLUMN no_trigger_note     TEXT;
ALTER TABLE rm_decision_cards ADD COLUMN model_version       VARCHAR(32);
