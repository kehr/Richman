-- 022_v2_user_feedback_and_columns (down): remove v2 additions from rm_users,
-- rm_decision_cards, and drop rm_user_feedback.
-- Runner wraps this file in a single BEGIN/COMMIT transaction.

-- Remove v2 decision card columns (reverse order of up)
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS model_version;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS no_trigger_note;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS default_action;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS concentration_message;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS concentration_level;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS valid_days;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS take_profit;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS stop_loss;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS scenarios;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS action_label;
ALTER TABLE rm_decision_cards DROP COLUMN IF EXISTS action;

-- Remove v2 user columns
ALTER TABLE rm_users DROP COLUMN IF EXISTS disclaimer_accepted_at;
ALTER TABLE rm_users DROP COLUMN IF EXISTS subscription_tier;
ALTER TABLE rm_users DROP COLUMN IF EXISTS email_push_enabled;

-- Restore v1 risk_preference: default 'neutral', CHECK conservative/neutral/aggressive
ALTER TABLE rm_users DROP CONSTRAINT IF EXISTS chk_users_risk_preference;
UPDATE rm_users SET risk_preference = 'neutral' WHERE risk_preference = 'moderate';
ALTER TABLE rm_users ALTER COLUMN risk_preference SET DEFAULT 'neutral';
ALTER TABLE rm_users ADD CONSTRAINT chk_users_risk_preference
    CHECK (risk_preference IN ('conservative', 'neutral', 'aggressive'));

-- Drop feedback table
DROP TABLE IF EXISTS rm_user_feedback;
